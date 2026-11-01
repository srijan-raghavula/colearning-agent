package controller

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type AuthController struct {
	renderer *view.Renderer
	accounts auth.UserStore
}

func NewAuthController(renderer *view.Renderer, accounts auth.UserStore) AuthController {
	return AuthController{renderer: renderer, accounts: accounts}
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
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Name     string `json:"name"`
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
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		writeAuthFailure(w, r, http.StatusInternalServerError, "Account creation failed. Please try again.")
		return
	}
	id := fmt.Sprintf("%s-%x", req.Role, randomID[:])
	account := auth.Account{ID: id, Email: req.Email, Role: req.Role, Name: req.Name}
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
	secure := r.TLS != nil || strings.EqualFold(os.Getenv("COLEARN_COOKIE_SECURE"), "true")
	http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (c AuthController) ProfilePage(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "app/profile.html", identity)
}
