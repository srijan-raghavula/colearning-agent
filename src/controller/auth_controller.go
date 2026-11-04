package controller

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type AuthController struct {
	renderer        *view.Renderer
	accounts        auth.UserStore
	profileStore    accountProfileStore
	credentialStore accountCredentialStore
	enrollments     CourseEnrollmentStore
}

type accountProfileStore interface {
	FindAccountByID(context.Context, string) (auth.Account, bool, error)
	UpdateAccountProfile(context.Context, auth.Account) error
}

type accountCredentialStore interface {
	UpdateAccountPassword(context.Context, string, string, string) error
}

func NewAuthController(renderer *view.Renderer, accounts auth.UserStore) AuthController {
	controller := AuthController{renderer: renderer, accounts: accounts}
	controller.profileStore, _ = accounts.(accountProfileStore)
	controller.credentialStore, _ = accounts.(accountCredentialStore)
	return controller
}

func (c AuthController) WithEnrollmentStore(store CourseEnrollmentStore) AuthController {
	c.enrollments = store
	return c
}

func (c AuthController) LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "login.html", map[string]any{"Error": r.URL.Query().Get("error")})
}

func (c AuthController) SignupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "signup.html", map[string]any{"Error": r.URL.Query().Get("error")})
}

type signupRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	Role            string `json:"role"`
	Name            string `json:"name"`
	AcademicProgram string `json:"academic_program"`
	Branch          string `json:"branch"`
	Semester        int    `json:"semester"`
	AcademicYear    string `json:"academic_year"`
	Department      string `json:"department"`
	TeachingTitle   string `json:"teaching_title"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (c AuthController) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c.accounts == nil {
		writeAuthFailure(w, r, http.StatusServiceUnavailable, "Account service is unavailable.")
		return
	}
	var req signupRequest
	if err := decodeAuthRequest(w, r, &req); err != nil {
		writeAuthFailure(w, r, http.StatusBadRequest, "invalid request")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if _, err := mail.ParseAddress(req.Email); err != nil || len(req.Email) > 254 || req.Name == "" || len([]rune(req.Name)) > 100 || len(req.Password) < 8 || len(req.Password) > 72 {
		writeAuthFailure(w, r, http.StatusBadRequest, "Enter a valid email, name and password (8 to 72 characters).")
		return
	}
	if req.Role != "student" && req.Role != "teacher" {
		writeAuthFailure(w, r, http.StatusBadRequest, "Choose student or teacher as your role.")
		return
	}
	req.AcademicProgram = strings.TrimSpace(req.AcademicProgram)
	req.Branch = strings.TrimSpace(req.Branch)
	req.AcademicYear = strings.TrimSpace(req.AcademicYear)
	req.Department = strings.TrimSpace(req.Department)
	req.TeachingTitle = strings.TrimSpace(req.TeachingTitle)
	if req.Role == string(domain.RoleStudent) && (req.AcademicProgram == "" || req.Branch == "" || req.Semester < 1 || req.AcademicYear == "") {
		writeAuthFailure(w, r, http.StatusBadRequest, "Student workspaces require academic program, branch, semester and academic year.")
		return
	}
	if req.Role == string(domain.RoleTeacher) && (req.Department == "" || req.TeachingTitle == "") {
		writeAuthFailure(w, r, http.StatusBadRequest, "Teacher workspaces require department and teaching title.")
		return
	}
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		writeAuthFailure(w, r, http.StatusInternalServerError, "Account creation failed. Please try again.")
		return
	}
	id := fmt.Sprintf("%s-%x", req.Role, randomID[:])
	account := auth.Account{User: domain.User{ID: id, Email: req.Email, Role: domain.UserRole(req.Role), Name: req.Name}, AcademicProgram: req.AcademicProgram, Branch: req.Branch, Semester: req.Semester, AcademicYear: req.AcademicYear, Department: req.Department, TeachingTitle: req.TeachingTitle}
	if err := account.Validate(); err != nil {
		writeAuthFailure(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := auth.SignupWithStore(r.Context(), c.accounts, account, req.Password); err != nil {
		writeAuthFailure(w, r, http.StatusConflict, "Could not create account. This email may already be registered.")
		return
	}
	token, err := auth.GenerateToken(id, req.Role, req.Email, req.Name)
	if err != nil {
		writeAuthFailure(w, r, http.StatusInternalServerError, "Account was created but sign-in could not be completed.")
		return
	}
	setAuthCookie(w, r, token)
	if wantsHTML(r) {
		redirectByRole(w, r, req.Role)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]any{"user_id": id, "email": req.Email, "role": req.Role, "name": req.Name}})
}

func (c AuthController) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if c.accounts == nil {
		writeAuthFailure(w, r, http.StatusServiceUnavailable, "Account service is unavailable.")
		return
	}
	var req loginRequest
	if err := decodeAuthRequest(w, r, &req); err != nil {
		writeAuthFailure(w, r, http.StatusBadRequest, "Invalid sign-in request.")
		return
	}
	var user map[string]any
	var token string
	var err error
	user, token, err = auth.LoginWithStore(r.Context(), c.accounts, req.Email, req.Password)
	if err != nil {
		writeAuthFailure(w, r, http.StatusUnauthorized, "Email or password was incorrect.")
		return
	}
	setAuthCookie(w, r, token)
	if wantsHTML(r) {
		if user["role"] == "teacher" {
			http.Redirect(w, r, "/teacher", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/student", http.StatusSeeOther)
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"user": user})
}

func (c AuthController) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeAuthFailure(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	store, _ := c.accounts.(auth.IdentityStore)
	identity, err := auth.IdentityFromRequestWithStore(r, store)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
		return
	}
	if c.profileStore == nil {
		http.Error(w, `{"error":"account service unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	account, found, lookupErr := c.profileStore.FindAccountByID(r.Context(), identity.ID)
	if lookupErr != nil || !found {
		http.Error(w, `{"error":"account not found"}`, http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"user": profileDTO(account)})
}

func (c AuthController) ProfileJSON(w http.ResponseWriter, r *http.Request) {
	identity, err := auth.IdentityFromRequestWithStore(r, mustIdentityStore(c.accounts))
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	if c.profileStore == nil {
		writeAPIError(w, http.StatusServiceUnavailable, errors.New("profile service is unavailable"))
		return
	}
	if r.Method == http.MethodPut || r.Method == http.MethodPatch {
		var request profileJSONRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		account, found, lookupErr := c.profileStore.FindAccountByID(r.Context(), identity.ID)
		if lookupErr != nil || !found {
			writeAPIError(w, http.StatusNotFound, errors.New("account not found"))
			return
		}
		account.Name = strings.TrimSpace(request.Name)
		account.AcademicProgram = strings.TrimSpace(request.AcademicProgram)
		account.Branch = strings.TrimSpace(request.Branch)
		account.Semester = request.Semester
		account.AcademicYear = strings.TrimSpace(request.AcademicYear)
		account.Department = strings.TrimSpace(request.Department)
		account.TeachingTitle = strings.TrimSpace(request.TeachingTitle)
		if account.Name == "" || len([]rune(account.Name)) > 100 || len([]rune(account.AcademicProgram)) > 120 || len([]rune(account.Branch)) > 120 || len([]rune(account.AcademicYear)) > 40 || len([]rune(account.Department)) > 120 || len([]rune(account.TeachingTitle)) > 80 || account.Semester < 0 || account.Semester > 20 {
			writeAPIError(w, http.StatusBadRequest, errors.New("check your profile details and try again"))
			return
		}
		if err := c.profileStore.UpdateAccountProfile(r.Context(), account); err != nil {
			writeAPIError(w, http.StatusInternalServerError, errors.New("could not save profile"))
			return
		}
		token, tokenErr := auth.GenerateToken(account.ID, account.Role, account.Email, account.Name, account.AuthVersion)
		if tokenErr != nil {
			writeAPIError(w, http.StatusInternalServerError, errors.New("profile saved but session could not be refreshed"))
			return
		}
		setAuthCookie(w, r, token)
		writeJSON(w, http.StatusOK, map[string]any{"user": profileDTO(account)})
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut+", "+http.MethodPatch)
		writeAPIError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	account, found, err := c.profileStore.FindAccountByID(r.Context(), identity.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, errors.New("could not load profile"))
		return
	}
	if !found {
		writeAPIError(w, http.StatusNotFound, errors.New("account not found"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": profileDTO(account)})
}

type profileJSONRequest struct {
	Name            string `json:"name"`
	AcademicProgram string `json:"academic_program"`
	Branch          string `json:"branch"`
	Semester        int    `json:"semester"`
	AcademicYear    string `json:"academic_year"`
	Department      string `json:"department"`
	TeachingTitle   string `json:"teaching_title"`
}

func profileDTO(account auth.Account) map[string]any {
	profile := map[string]any{}
	if account.Role == "student" {
		profile = map[string]any{
			"academic_program": account.AcademicProgram,
			"branch":           account.Branch,
			"semester":         account.Semester,
			"academic_year":    account.AcademicYear,
		}
	} else {
		profile = map[string]any{
			"department":     account.Department,
			"teaching_title": account.TeachingTitle,
		}
	}
	return map[string]any{
		"id": account.ID, "name": account.Name, "email": account.Email,
		"role": account.Role, "auth_version": account.AuthVersion, "profile": profile,
	}
}

func mustIdentityStore(accounts auth.UserStore) auth.IdentityStore {
	store, _ := accounts.(auth.IdentityStore)
	return store
}

func decodeAuthRequest(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if mediaType != "application/json" {
		if mediaType != "application/x-www-form-urlencoded" {
			return errors.New("unsupported content type")
		}
		if err := r.ParseForm(); err != nil {
			return err
		}
		switch req := target.(type) {
		case *signupRequest:
			req.Email, req.Password, req.Role, req.Name = r.FormValue("email"), r.FormValue("password"), r.FormValue("role"), r.FormValue("name")
		case *loginRequest:
			req.Email, req.Password, req.Role = r.FormValue("email"), r.FormValue("password"), r.FormValue("role")
		default:
			return errors.New("unsupported form")
		}
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err := rejectDuplicateJSONFields(body); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func rejectDuplicateJSONFields(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return errors.New("request must be a JSON object")
	}
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return errors.New("invalid JSON object key")
		}
		normalizedKey := strings.ToLower(key)
		if _, exists := seen[normalizedKey]; exists {
			return errors.New("duplicate JSON object key")
		}
		seen[normalizedKey] = struct{}{}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
	}
	last, err := decoder.Token()
	if err != nil {
		return err
	}
	if last != json.Delim('}') {
		return errors.New("request must be a JSON object")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON object")
	}
	return nil
}

func writeAuthFailure(w http.ResponseWriter, r *http.Request, status int, message string) {
	if wantsHTML(r) {
		page := "/login"
		if strings.Contains(r.URL.Path, "signup") {
			page = "/signup"
		}
		query := url.Values{"error": []string{message}}
		http.Redirect(w, r, page+"?"+query.Encode(), http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func wantsHTML(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/html") && !strings.Contains(r.Header.Get("Accept"), "application/json")
}

func redirectByRole(w http.ResponseWriter, r *http.Request, role string) {
	if wantsHTML(r) {
		if role == "teacher" {
			http.Redirect(w, r, "/teacher", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/student", http.StatusSeeOther)
		}
	}
}

func setAuthCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := r.TLS != nil || strings.EqualFold(os.Getenv("COLEARN_COOKIE_SECURE"), "true")
	http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 24 * 60 * 60})
}

func (c AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || strings.EqualFold(os.Getenv("COLEARN_COOKIE_SECURE"), "true")
	http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func (c AuthController) ProfilePage(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if c.profileStore == nil {
		http.Error(w, "Profile service is unavailable", http.StatusServiceUnavailable)
		return
	}
	account, found, err := c.profileStore.FindAccountByID(r.Context(), identity.ID)
	if err != nil {
		http.Error(w, "Could not load profile", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}
	var enrollments []domain.CourseEnrollment
	if account.Role == "student" && c.enrollments != nil {
		enrollments, err = c.enrollments.ListStudentEnrollments(r.Context(), account.ID)
		if err != nil {
			http.Error(w, "Could not load course enrollments", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "app/profile.html", map[string]any{
		"Account":     account,
		"Enrollments": enrollments,
		"Role":        account.Role,
		"Notice":      r.URL.Query().Get("notice"),
		"Error":       r.URL.Query().Get("error"),
	})
}

func (c AuthController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if c.profileStore == nil {
		http.Error(w, "Profile service is unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/profile?error=Invalid+profile+form", http.StatusSeeOther)
		return
	}
	account, found, err := c.profileStore.FindAccountByID(r.Context(), identity.ID)
	if err != nil || !found {
		http.Error(w, "Could not load account", http.StatusInternalServerError)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	program := strings.TrimSpace(r.FormValue("academic_program"))
	branch := strings.TrimSpace(r.FormValue("branch"))
	academicYear := strings.TrimSpace(r.FormValue("academic_year"))
	department := strings.TrimSpace(r.FormValue("department"))
	teachingTitle := strings.TrimSpace(r.FormValue("teaching_title"))
	semester := 0
	semesterValue := strings.TrimSpace(r.FormValue("semester"))
	var parseErr error
	if semesterValue != "" {
		semester, parseErr = strconv.Atoi(semesterValue)
	}
	if name == "" || len([]rune(name)) > 100 || len([]rune(program)) > 120 || len([]rune(branch)) > 120 || len([]rune(academicYear)) > 40 || len([]rune(department)) > 120 || len([]rune(teachingTitle)) > 80 || parseErr != nil || semester < 0 || semester > 20 {
		http.Redirect(w, r, "/profile?error=Check+your+profile+details+and+try+again", http.StatusSeeOther)
		return
	}
	account.Name = name
	account.AcademicProgram = program
	account.Branch = branch
	account.Semester = semester
	account.AcademicYear = academicYear
	account.Department = department
	account.TeachingTitle = teachingTitle
	if err := c.profileStore.UpdateAccountProfile(r.Context(), account); err != nil {
		http.Error(w, "Could not save profile", http.StatusInternalServerError)
		return
	}
	token, err := auth.GenerateToken(account.ID, account.Role, account.Email, account.Name, account.AuthVersion)
	if err != nil {
		http.Error(w, "Profile saved but session could not be refreshed", http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, r, token)
	http.Redirect(w, r, "/profile?notice=Profile+saved", http.StatusSeeOther)
}

func (c AuthController) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if c.profileStore == nil || c.credentialStore == nil {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Password service is unavailable"), http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Invalid password form"), http.StatusSeeOther)
		return
	}
	account, found, err := c.profileStore.FindAccountByID(r.Context(), identity.ID)
	if err != nil || !found {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Could not verify your account"), http.StatusSeeOther)
		return
	}
	current := r.FormValue("current_password")
	password := r.FormValue("new_password")
	confirmation := r.FormValue("confirm_password")
	if !auth.CheckPassword(account.PasswordHash, current) {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Current password is incorrect"), http.StatusSeeOther)
		return
	}
	if len(password) < 8 || len(password) > 72 || password != confirmation || password == current {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Choose a new password (8 to 72 characters) and enter it the same way twice"), http.StatusSeeOther)
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Could not secure the new password"), http.StatusSeeOther)
		return
	}
	if err := c.credentialStore.UpdateAccountPassword(r.Context(), account.ID, account.PasswordHash, hash); err != nil {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	account.AuthVersion++
	token, err := auth.GenerateToken(account.ID, account.Role, account.Email, account.Name, account.AuthVersion)
	if err != nil {
		clearAuthCookie(w, r)
		http.Redirect(w, r, "/login?notice="+url.QueryEscape("Password changed. Sign in with your new password."), http.StatusSeeOther)
		return
	}
	setAuthCookie(w, r, token)
	http.Redirect(w, r, "/profile?notice="+url.QueryEscape("Password updated. Other sessions were signed out."), http.StatusSeeOther)
}
