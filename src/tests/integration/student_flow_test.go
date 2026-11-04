package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/sqlite"
	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/bootstrap"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/routes"
)

func TestEndToEndStudentFlow(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err != nil {
		t.Fatalf("locate project root: %v", err)
	}
	t.Chdir(projectRoot)
	dataDir := t.TempDir()
	t.Setenv("COLEARN_DB_PATH", filepath.Join(dataDir, "colearning.db"))
	t.Setenv("LEARNING_STORE_PATH", filepath.Join(dataDir, "learning-state.json"))
	t.Setenv("COLEARN_JWT_SECRET", strings.Repeat("t", 32))

	app, err := bootstrap.NewApp()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if app.Close != nil {
		t.Cleanup(func() { _ = app.Close() })
	}
	db, err := sqlite.OpenDB(os.Getenv("COLEARN_DB_PATH"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	courseRepo := sqlite.NewRepo(db)
	additionalCourse := domain.LearningCourse{ID: "course-physics-basics", Title: "Physics basics", Subject: "Physics", Description: "A first course in motion and forces.", TeacherID: "T-001", TeacherName: "Maya Chen", ConceptCount: 3, TutorPolicy: domain.TutorPolicy{DefaultMode: domain.TutorModeGuided}}
	if err := courseRepo.SaveCourse(context.Background(), additionalCourse); err != nil {
		t.Fatalf("seed additional course: %v", err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	routes.Register(mux, app.Routes)

	form := url.Values{"email": {"student@demo.edu"}, "password": {"student123"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("Accept", "application/json")
	loginRecorder := httptest.NewRecorder()
	mux.ServeHTTP(loginRecorder, loginReq)
	loginResp := loginRecorder.Result()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginResp.StatusCode, http.StatusOK)
	}
	var authCookie *http.Cookie
	for _, cookie := range loginResp.Cookies() {
		if cookie.Name == "auth_token" {
			authCookie = cookie
		}
	}
	if authCookie == nil {
		t.Fatal("login response did not set auth cookie")
	}

	doRequest := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.AddCookie(authCookie)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		return resp
	}

	profile := doRequest(http.MethodGet, "/profile", "")
	profileBody, _ := io.ReadAll(profile.Result().Body)
	if profile.Code != http.StatusOK || !strings.Contains(string(profileBody), "student@demo.edu") {
		t.Fatalf("profile response status = %d or authenticated profile details missing", profile.Code)
	}

	dashboard := doRequest(http.MethodGet, "/api/v1/student/dashboard?student_id=S-002", "")
	var dashboardJSON map[string]any
	if dashboard.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d", dashboard.Code, http.StatusOK)
	}
	if err := json.NewDecoder(dashboard.Body).Decode(&dashboardJSON); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	if dashboardJSON["student_id"] != "S-001" || dashboardJSON["student_name"] != "Alex Morgan" {
		t.Fatalf("dashboard identity = (%v, %v), want authenticated account", dashboardJSON["student_id"], dashboardJSON["student_name"])
	}
	curriculumResponse := doRequest(http.MethodGet, "/api/v1/student/courses/course-algebra-foundations/curriculum", "")
	if curriculumResponse.Code != http.StatusOK {
		t.Fatalf("student curriculum status = %d, want %d", curriculumResponse.Code, http.StatusOK)
	}
	var curriculumJSON struct {
		CourseID string `json:"course_id"`
		Modules  []struct {
			Chapters []struct {
				Lessons []struct {
					Contents []struct {
						Published bool `json:"published"`
					} `json:"contents"`
				} `json:"lessons"`
			} `json:"chapters"`
		} `json:"modules"`
	}
	if err := json.NewDecoder(curriculumResponse.Body).Decode(&curriculumJSON); err != nil {
		t.Fatalf("decode student curriculum: %v", err)
	}
	if curriculumJSON.CourseID != "course-algebra-foundations" || len(curriculumJSON.Modules) == 0 {
		t.Fatalf("student curriculum = %+v, want seeded course modules", curriculumJSON)
	}
	for _, module := range curriculumJSON.Modules {
		for _, chapter := range module.Chapters {
			for _, lesson := range chapter.Lessons {
				for _, content := range lesson.Contents {
					if !content.Published {
						t.Fatal("student curriculum exposed unpublished lesson content")
					}
				}
			}
		}
	}

	studentPage := doRequest(http.MethodGet, "/student", "")
	studentHTML, _ := io.ReadAll(studentPage.Body)
	if studentPage.Code != http.StatusOK || !strings.Contains(string(studentHTML), "Account menu for Alex Morgan") || !strings.Contains(string(studentHTML), "course_id=course-algebra-foundations") || !strings.Contains(string(studentHTML), "concept_id=linear-equations") || strings.Contains(string(studentHTML), ">Open</button>") {
		t.Fatalf("student overview status = %d or account/concept navigation missing", studentPage.Code)
	}
	if !strings.Contains(string(studentHTML), "/student/workspace?course_id=course-algebra-foundations") {
		t.Fatal("student Overview course action does not open the selected course learning space")
	}
	workspaceLink := doRequest(http.MethodGet, "/student/workspace", "")
	if workspaceLink.Code != http.StatusSeeOther || !strings.Contains(workspaceLink.Header().Get("Location"), "session_id=") {
		t.Fatalf("learning space navigation status = %d, location %q", workspaceLink.Code, workspaceLink.Header().Get("Location"))
	}

	conceptPage := doRequest(http.MethodGet, "/student/workspace?concept_id=graphing-equations", "")
	if conceptPage.Code != http.StatusSeeOther || !strings.Contains(conceptPage.Header().Get("Location"), "session_id=") {
		t.Fatalf("concept link status = %d, want redirect to selected concept session", conceptPage.Code)
	}

	catalog := doRequest(http.MethodGet, "/student/courses", "")
	if catalog.Code != http.StatusOK || !strings.Contains(catalog.Body.String(), "Physics basics") || !strings.Contains(catalog.Body.String(), "teacher@demo.edu") || !strings.Contains(catalog.Body.String(), "Assistant Professor") {
		t.Fatalf("course catalog status = %d or teacher/course details missing", catalog.Code)
	}
	courseDetails := doRequest(http.MethodGet, "/student/courses/course-physics-basics", "")
	if courseDetails.Code != http.StatusOK || !strings.Contains(courseDetails.Body.String(), "Request enrollment") {
		t.Fatalf("course details status = %d or request action missing", courseDetails.Code)
	}
	enrollmentRequest := doRequest(http.MethodPost, "/student/courses/course-physics-basics/enroll", "")
	if enrollmentRequest.Code != http.StatusSeeOther {
		t.Fatalf("enrollment request status = %d, want redirect", enrollmentRequest.Code)
	}
	pendingWorkspace := doRequest(http.MethodGet, "/student/workspace?course_id=course-physics-basics&concept_id=linear-equations", "")
	if pendingWorkspace.Code != http.StatusSeeOther || !strings.Contains(pendingWorkspace.Header().Get("Location"), "/student/courses/course-physics-basics") {
		t.Fatalf("pending student course access status = %d, want redirect to course details", pendingWorkspace.Code)
	}
	pendingAPISession := doRequest(http.MethodPost, "/api/v1/student/sessions", `{"student_id":"S-002","course_id":"course-physics-basics","concept_id":"linear-equations"}`)
	if pendingAPISession.Code != http.StatusForbidden {
		t.Fatalf("pending API course access status = %d, want forbidden", pendingAPISession.Code)
	}
	if err := courseRepo.RequestCourseEnrollment(context.Background(), "S-002", additionalCourse.ID); err != nil {
		t.Fatalf("seed second learner request: %v", err)
	}

	turn := doRequest(http.MethodPost, "/api/v1/student/sessions/session-s001-linear/turns", `{"student_id":"S-002","message":"I will check by substituting my answer."}`)
	var turnJSON map[string]any
	if turn.Code != http.StatusOK {
		t.Fatalf("tutor turn status = %d, want %d", turn.Code, http.StatusOK)
	}
	if err := json.NewDecoder(turn.Body).Decode(&turnJSON); err != nil {
		t.Fatalf("decode tutor turn: %v", err)
	}
	if _, ok := turnJSON["turn"]; !ok {
		t.Fatal("tutor response has no generated turn")
	}

	profileForm := url.Values{
		"name":             {"Alex Morgan Updated"},
		"academic_program": {"BSc Computer Science"},
		"branch":           {"Computer Science"},
		"semester":         {"3"},
		"academic_year":    {"2026-27"},
	}
	profileUpdate := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader(profileForm.Encode()))
	profileUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	profileUpdate.AddCookie(authCookie)
	profileUpdateRecorder := httptest.NewRecorder()
	mux.ServeHTTP(profileUpdateRecorder, profileUpdate)
	if profileUpdateRecorder.Code != http.StatusSeeOther {
		t.Fatalf("profile update status = %d, want redirect", profileUpdateRecorder.Code)
	}
	profileUpdatedCookie := profileUpdateRecorder.Result().Cookies()
	for _, cookie := range profileUpdatedCookie {
		if cookie.Name == "auth_token" {
			authCookie = cookie
		}
	}
	updatedProfile := doRequest(http.MethodGet, "/profile", "")
	updatedProfileBody, _ := io.ReadAll(updatedProfile.Body)
	if updatedProfile.Code != http.StatusOK || !strings.Contains(string(updatedProfileBody), "BSc Computer Science") || !strings.Contains(string(updatedProfileBody), "value=\"3\"") || !strings.Contains(string(updatedProfileBody), "Alex Morgan Updated") {
		t.Fatalf("updated profile did not persist study details (status %d)", updatedProfile.Code)
	}
	updatedDashboard := doRequest(http.MethodGet, "/api/v1/student/dashboard", "")
	var updatedDashboardJSON map[string]any
	if updatedDashboard.Code != http.StatusOK || json.NewDecoder(updatedDashboard.Body).Decode(&updatedDashboardJSON) != nil || updatedDashboardJSON["student_name"] != "Alex Morgan Updated" {
		t.Fatalf("refreshed session did not reflect profile name: status %d, name %v", updatedDashboard.Code, updatedDashboardJSON["student_name"])
	}
	oldSessionCookie := authCookie
	passwordForm := url.Values{"current_password": {"student123"}, "new_password": {"student456"}, "confirm_password": {"student456"}}
	passwordRequest := httptest.NewRequest(http.MethodPost, "/profile/password", strings.NewReader(passwordForm.Encode()))
	passwordRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	passwordRequest.AddCookie(authCookie)
	passwordRecorder := httptest.NewRecorder()
	mux.ServeHTTP(passwordRecorder, passwordRequest)
	if passwordRecorder.Code != http.StatusSeeOther || !strings.Contains(passwordRecorder.Header().Get("Location"), "Password+updated") {
		t.Fatalf("password update status = %d, location %q", passwordRecorder.Code, passwordRecorder.Header().Get("Location"))
	}
	for _, cookie := range passwordRecorder.Result().Cookies() {
		if cookie.Name == "auth_token" {
			authCookie = cookie
		}
	}
	oldSession := httptest.NewRequest(http.MethodGet, "/profile", nil)
	oldSession.AddCookie(oldSessionCookie)
	oldSessionRecorder := httptest.NewRecorder()
	mux.ServeHTTP(oldSessionRecorder, oldSession)
	if oldSessionRecorder.Code != http.StatusSeeOther || oldSessionRecorder.Header().Get("Location") != "/login" {
		t.Fatalf("old session status = %d, location %q; want revoked session redirect", oldSessionRecorder.Code, oldSessionRecorder.Header().Get("Location"))
	}
	newPasswordLogin := url.Values{"email": {"student@demo.edu"}, "password": {"student456"}}
	newLoginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(newPasswordLogin.Encode()))
	newLoginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	newLoginRequest.Header.Set("Accept", "application/json")
	newLoginRecorder := httptest.NewRecorder()
	mux.ServeHTTP(newLoginRecorder, newLoginRequest)
	if newLoginRecorder.Code != http.StatusOK {
		t.Fatalf("login with updated password status = %d, want %d", newLoginRecorder.Code, http.StatusOK)
	}
	if err := courseRepo.UserCreate(context.Background(), "S-001", "student@demo.edu", "seed-password-hash", "student", "Seeded Name"); err != nil {
		t.Fatalf("repeat account seed: %v", err)
	}
	studentAccount, foundAccount, accountErr := courseRepo.FindAccountByID(context.Background(), "S-001")
	if accountErr != nil || !foundAccount || studentAccount.Name != "Alex Morgan Updated" || !auth.CheckPassword(studentAccount.PasswordHash, "student456") {
		t.Fatalf("repeat seed overwrote saved account details: account=%#v found=%v err=%v", studentAccount, foundAccount, accountErr)
	}

	teacherForm := url.Values{"email": {"teacher@demo.edu"}, "password": {"teacher123"}}
	teacherLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(teacherForm.Encode()))
	teacherLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	teacherLogin.Header.Set("Accept", "application/json")
	teacherLoginRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherLoginRecorder, teacherLogin)
	if teacherLoginRecorder.Code != http.StatusOK {
		t.Fatalf("teacher login status = %d, want %d", teacherLoginRecorder.Code, http.StatusOK)
	}
	var teacherCookie *http.Cookie
	for _, cookie := range teacherLoginRecorder.Result().Cookies() {
		if cookie.Name == "auth_token" {
			teacherCookie = cookie
		}
	}
	if teacherCookie == nil {
		t.Fatal("teacher login did not set auth cookie")
	}
	teacherCurriculum := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/courses/course-algebra-foundations/curriculum", nil)
	teacherCurriculum.AddCookie(teacherCookie)
	teacherCurriculumRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherCurriculumRecorder, teacherCurriculum)
	if teacherCurriculumRecorder.Code != http.StatusOK {
		t.Fatalf("teacher curriculum status = %d, want %d", teacherCurriculumRecorder.Code, http.StatusOK)
	}
	var teacherCurriculumJSON map[string]any
	if err := json.NewDecoder(teacherCurriculumRecorder.Body).Decode(&teacherCurriculumJSON); err != nil {
		t.Fatalf("decode teacher curriculum: %v", err)
	}
	if teacherCurriculumJSON["course_id"] != "course-algebra-foundations" {
		t.Fatalf("teacher curriculum course = %v", teacherCurriculumJSON["course_id"])
	}
	teacherProfileForm := url.Values{"name": {"Maya Chen"}, "department": {"Mathematics"}, "teaching_title": {"Professor"}}
	teacherProfileRequest := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader(teacherProfileForm.Encode()))
	teacherProfileRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	teacherProfileRequest.AddCookie(teacherCookie)
	teacherProfileRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherProfileRecorder, teacherProfileRequest)
	if teacherProfileRecorder.Code != http.StatusSeeOther {
		t.Fatalf("teacher profile update status = %d", teacherProfileRecorder.Code)
	}
	teacherRequest := httptest.NewRequest(http.MethodGet, "/teacher/content", nil)
	teacherRequest.AddCookie(teacherCookie)
	teacherRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherRecorder, teacherRequest)
	if teacherRecorder.Code != http.StatusOK || !strings.Contains(teacherRecorder.Body.String(), "Account menu for Maya Chen") {
		t.Fatalf("teacher content status = %d or account menu missing", teacherRecorder.Code)
	}
	teacherContentSelected := httptest.NewRequest(http.MethodGet, "/teacher/content?course_id=course-algebra-foundations", nil)
	teacherContentSelected.AddCookie(teacherCookie)
	teacherContentSelectedRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherContentSelectedRecorder, teacherContentSelected)
	if teacherContentSelectedRecorder.Code != http.StatusOK || !strings.Contains(teacherContentSelectedRecorder.Body.String(), "Supplement the lesson context") || !strings.Contains(teacherContentSelectedRecorder.Body.String(), "Target lesson or learning concept") {
		t.Fatalf("selected content studio status = %d or targeting explanation missing", teacherContentSelectedRecorder.Code)
	}
	if strings.Contains(teacherContentSelectedRecorder.Body.String(), ">Open</button>") {
		t.Fatal("content studio course selector should apply changes without a duplicate Open button")
	}
	teacherCourses := httptest.NewRequest(http.MethodGet, "/teacher/courses", nil)
	teacherCourses.AddCookie(teacherCookie)
	teacherCoursesRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherCoursesRecorder, teacherCourses)
	if teacherCoursesRecorder.Code != http.StatusOK || !strings.Contains(teacherCoursesRecorder.Body.String(), "Algebra foundations") || !strings.Contains(teacherCoursesRecorder.Body.String(), "Physics basics") {
		t.Fatalf("teacher course list status = %d or owned courses missing", teacherCoursesRecorder.Code)
	}
	teacherSelectedCourse := httptest.NewRequest(http.MethodGet, "/teacher?course_id=course-physics-basics", nil)
	teacherSelectedCourse.AddCookie(teacherCookie)
	teacherSelectedRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherSelectedRecorder, teacherSelectedCourse)
	if teacherSelectedRecorder.Code != http.StatusOK || !strings.Contains(teacherSelectedRecorder.Body.String(), "Physics basics") {
		t.Fatalf("teacher selected course status = %d or selected course missing", teacherSelectedRecorder.Code)
	}
	for _, duplicateAction := range []string{"Review enrollment requests", "Manage courses", "Open content studio", "Review learning context", "Edit tutor policy"} {
		if strings.Contains(teacherSelectedRecorder.Body.String(), duplicateAction) {
			t.Errorf("teacher overview duplicates top navigation with %q", duplicateAction)
		}
	}
	if strings.Count(teacherSelectedRecorder.Body.String(), `href="/teacher/content"`) != 1 || strings.Count(teacherSelectedRecorder.Body.String(), `href="/teacher/enrollments"`) != 1 || strings.Count(teacherSelectedRecorder.Body.String(), `href="/teacher/courses"`) != 1 {
		t.Errorf("teacher overview should expose each studio destination once in shared navigation")
	}
	createCourse := httptest.NewRequest(http.MethodGet, "/teacher/courses/new", nil)
	createCourse.AddCookie(teacherCookie)
	createCourseRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createCourseRecorder, createCourse)
	if createCourseRecorder.Code != http.StatusOK || !strings.Contains(createCourseRecorder.Body.String(), "Create a course") {
		t.Fatalf("course creation page status = %d", createCourseRecorder.Code)
	}
	invalidCourseForm := url.Values{"title": {"Astronomy foundations"}, "subject": {""}, "description": {"Keep these values while correcting the missing subject."}}
	invalidCourseRequest := httptest.NewRequest(http.MethodPost, "/teacher/courses", strings.NewReader(invalidCourseForm.Encode()))
	invalidCourseRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidCourseRequest.AddCookie(teacherCookie)
	invalidCourseRecorder := httptest.NewRecorder()
	mux.ServeHTTP(invalidCourseRecorder, invalidCourseRequest)
	if invalidCourseRecorder.Code != http.StatusOK || !strings.Contains(invalidCourseRecorder.Body.String(), "Enter a course title and subject") || !strings.Contains(invalidCourseRecorder.Body.String(), "Astronomy foundations") {
		t.Fatalf("invalid course form should show an inline error and keep entered values; status = %d", invalidCourseRecorder.Code)
	}
	newCourseForm := url.Values{"title": {"Intro to astronomy"}, "subject": {"Science"}, "description": {"An introductory course on the night sky."}}
	newCourseRequest := httptest.NewRequest(http.MethodPost, "/teacher/courses", strings.NewReader(newCourseForm.Encode()))
	newCourseRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	newCourseRequest.AddCookie(teacherCookie)
	newCourseRecorder := httptest.NewRecorder()
	mux.ServeHTTP(newCourseRecorder, newCourseRequest)
	if newCourseRecorder.Code != http.StatusSeeOther || !strings.HasPrefix(newCourseRecorder.Header().Get("Location"), "/teacher/courses/course-") {
		t.Fatalf("course creation status = %d, location %q", newCourseRecorder.Code, newCourseRecorder.Header().Get("Location"))
	}
	newCourseID := strings.Split(strings.TrimPrefix(newCourseRecorder.Header().Get("Location"), "/teacher/courses/"), "?")[0]
	studentCourseRequest := doRequest(http.MethodPost, "/student/courses/"+newCourseID+"/enroll", "")
	if studentCourseRequest.Code != http.StatusSeeOther {
		t.Fatalf("request new course enrollment status = %d", studentCourseRequest.Code)
	}
	approveNewCourse := url.Values{"student_id": {"S-001"}, "course_id": {newCourseID}, "decision": {"approved"}}
	approveNewCourseRequest := httptest.NewRequest(http.MethodPost, "/teacher/enrollments", strings.NewReader(approveNewCourse.Encode()))
	approveNewCourseRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveNewCourseRequest.AddCookie(teacherCookie)
	approveNewCourseRecorder := httptest.NewRecorder()
	mux.ServeHTTP(approveNewCourseRecorder, approveNewCourseRequest)
	if approveNewCourseRecorder.Code != http.StatusSeeOther {
		t.Fatalf("approve new course enrollment status = %d", approveNewCourseRecorder.Code)
	}
	approvedCourse, foundCourse, findCourseErr := courseRepo.FindCourse(context.Background(), newCourseID)
	if findCourseErr != nil || !foundCourse || approvedCourse.StudentCount != 1 {
		t.Fatalf("approved learner count = %d, found %v, err %v; want 1", approvedCourse.StudentCount, foundCourse, findCourseErr)
	}
	curriculumPage := httptest.NewRequest(http.MethodGet, "/teacher/courses/"+newCourseID+"/curriculum", nil)
	curriculumPage.AddCookie(teacherCookie)
	curriculumRecorder := httptest.NewRecorder()
	mux.ServeHTTP(curriculumRecorder, curriculumPage)
	if curriculumRecorder.Code != http.StatusOK || !strings.Contains(curriculumRecorder.Body.String(), "Add module") || !strings.Contains(curriculumRecorder.Body.String(), "Discard changes") || !strings.Contains(curriculumRecorder.Body.String(), "Save changes") {
		t.Fatalf("curriculum page status = %d", curriculumRecorder.Code)
	}
	postTeacherForm := func(path string, values url.Values) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(teacherCookie)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)
		return recorder
	}
	if response := postTeacherForm("/teacher/curriculum/modules", url.Values{"course_id": {newCourseID}, "title": {"Foundations"}, "description": {"Core ideas and vocabulary."}}); response.Code != http.StatusSeeOther {
		t.Fatalf("create module status = %d", response.Code)
	}
	modules, err := courseRepo.ListModules(context.Background(), newCourseID)
	if err != nil || len(modules) != 1 {
		t.Fatalf("saved modules = %d, err %v", len(modules), err)
	}
	if response := postTeacherForm("/teacher/curriculum/chapters", url.Values{"module_id": {modules[0].ID}, "title": {"Variables"}}); response.Code != http.StatusSeeOther {
		t.Fatalf("create chapter status = %d", response.Code)
	}
	chapters, err := courseRepo.ListChapters(context.Background(), modules[0].ID)
	if err != nil || len(chapters) != 1 {
		t.Fatalf("saved chapters = %d, err %v", len(chapters), err)
	}
	if response := postTeacherForm("/teacher/curriculum/lessons", url.Values{"chapter_id": {chapters[0].ID}, "title": {"Solve a one-step equation"}, "summary": {"Isolate the unknown and check the result."}}); response.Code != http.StatusSeeOther {
		t.Fatalf("create lesson status = %d", response.Code)
	}
	lessons, err := courseRepo.ListLessons(context.Background(), chapters[0].ID)
	if err != nil || len(lessons) != 1 {
		t.Fatalf("saved lessons = %d, err %v", len(lessons), err)
	}
	contentForm := url.Values{"lesson_id": {lessons[0].ID}, "kind": {"example"}, "title": {"Balance example"}, "body": {"Subtract three from both sides, then verify."}}
	if response := postTeacherForm("/teacher/curriculum/content", contentForm); response.Code != http.StatusSeeOther {
		t.Fatalf("create lesson content status = %d", response.Code)
	}
	lessonContents, err := courseRepo.ListLessonContents(context.Background(), lessons[0].ID)
	if err != nil || len(lessonContents) != 1 || lessonContents[0].Published {
		t.Fatalf("draft lesson content = %#v, err %v", lessonContents, err)
	}
	contentForm.Set("id", lessonContents[0].ID)
	contentForm.Set("published", "on")
	if response := postTeacherForm("/teacher/curriculum/content", contentForm); response.Code != http.StatusSeeOther {
		t.Fatalf("publish lesson content status = %d", response.Code)
	}
	lessonContents, err = courseRepo.ListLessonContents(context.Background(), lessons[0].ID)
	materials, materialsErr := courseRepo.ListMaterials(context.Background(), newCourseID, lessons[0].ID)
	if err != nil || materialsErr != nil || len(lessonContents) != 1 || !lessonContents[0].Published || len(materials) != 1 || !materials[0].Published {
		t.Fatalf("published tutor material missing (lesson content %d, materials %d, errs %v/%v)", len(lessonContents), len(materials), err, materialsErr)
	}
	lessons[0].Title = "Solve a one-step equation (revised)"
	lessonContents[0].Body = "Revised: subtract three, divide by two, then verify."
	draftJSON, marshalErr := json.Marshal(map[string]any{"modules": modules, "chapters": chapters, "lessons": lessons, "contents": lessonContents})
	if marshalErr != nil {
		t.Fatalf("encode curriculum draft: %v", marshalErr)
	}
	saveDraftRequest := httptest.NewRequest(http.MethodPost, "/teacher/courses/"+newCourseID+"/curriculum/save", strings.NewReader(string(draftJSON)))
	saveDraftRequest.Header.Set("Content-Type", "application/json")
	saveDraftRequest.AddCookie(teacherCookie)
	saveDraftRecorder := httptest.NewRecorder()
	mux.ServeHTTP(saveDraftRecorder, saveDraftRequest)
	if saveDraftRecorder.Code != http.StatusSeeOther {
		t.Fatalf("save all curriculum changes status = %d: %s", saveDraftRecorder.Code, saveDraftRecorder.Body.String())
	}
	savedLesson, foundLesson, findLessonErr := courseRepo.FindLesson(context.Background(), lessons[0].ID)
	updatedMaterials, updatedMaterialsErr := courseRepo.ListMaterials(context.Background(), newCourseID, lessons[0].ID)
	if findLessonErr != nil || !foundLesson || savedLesson.Title != lessons[0].Title || updatedMaterialsErr != nil || len(updatedMaterials) != 1 || updatedMaterials[0].Body != lessonContents[0].Body {
		t.Fatalf("global curriculum save did not synchronize lesson/material: lesson=%#v found=%v errors=%v/%v materials=%#v", savedLesson, foundLesson, findLessonErr, updatedMaterialsErr, updatedMaterials)
	}
	studentCourseDashboard := doRequest(http.MethodGet, "/student?course_id="+newCourseID, "")
	if studentCourseDashboard.Code != http.StatusOK || !strings.Contains(studentCourseDashboard.Body.String(), "Intro to astronomy") || !strings.Contains(studentCourseDashboard.Body.String(), lessons[0].Title) {
		t.Fatalf("selected course dashboard status = %d or lesson missing", studentCourseDashboard.Code)
	}
	studentMultiCourseDashboard := doRequest(http.MethodGet, "/student?course_id=course-algebra-foundations", "")
	if studentMultiCourseDashboard.Code != http.StatusOK || !strings.Contains(studentMultiCourseDashboard.Body.String(), "data-course-switcher") || !strings.Contains(studentMultiCourseDashboard.Body.String(), "Resume conversation") || strings.Contains(studentMultiCourseDashboard.Body.String(), ">Open</button>") {
		t.Fatalf("multi-course overview status = %d or clear course switch/resume action missing", studentMultiCourseDashboard.Code)
	}
	openCourse := doRequest(http.MethodGet, "/student/workspace?course_id="+newCourseID, "")
	if openCourse.Code != http.StatusSeeOther || !strings.Contains(openCourse.Header().Get("Location"), "session_id=") {
		t.Fatalf("open course action status = %d, location %q", openCourse.Code, openCourse.Header().Get("Location"))
	}
	sessionLocation := openCourse.Header().Get("Location")
	archiveJSON, archiveErr := json.Marshal(map[string]any{"modules": modules, "chapters": chapters, "lessons": []domain.CourseLesson{}, "contents": []domain.LessonContent{}})
	if archiveErr != nil {
		t.Fatalf("encode curriculum archive: %v", archiveErr)
	}
	archiveRequest := httptest.NewRequest(http.MethodPost, "/teacher/courses/"+newCourseID+"/curriculum/save", strings.NewReader(string(archiveJSON)))
	archiveRequest.Header.Set("Content-Type", "application/json")
	archiveRequest.AddCookie(teacherCookie)
	archiveRecorder := httptest.NewRecorder()
	mux.ServeHTTP(archiveRecorder, archiveRequest)
	if archiveRecorder.Code != http.StatusSeeOther {
		t.Fatalf("archive curriculum status = %d: %s", archiveRecorder.Code, archiveRecorder.Body.String())
	}
	archivedConcept, foundArchivedConcept, archivedConceptErr := courseRepo.FindConcept(context.Background(), lessons[0].ID)
	visibleLessons, visibleLessonsErr := courseRepo.ListLessons(context.Background(), chapters[0].ID)
	archivedMaterials, archivedMaterialsErr := courseRepo.ListMaterials(context.Background(), newCourseID, lessons[0].ID)
	if archivedConceptErr != nil || !foundArchivedConcept || !archivedConcept.Archived || visibleLessonsErr != nil || len(visibleLessons) != 0 || archivedMaterialsErr != nil || len(archivedMaterials) != 1 || archivedMaterials[0].Published {
		t.Fatalf("curriculum archive was not synchronized while retaining history: concept=%#v found=%v lessonCount=%d materials=%#v errors=%v/%v/%v", archivedConcept, foundArchivedConcept, len(visibleLessons), archivedMaterials, archivedConceptErr, visibleLessonsErr, archivedMaterialsErr)
	}
	archivedWorkspace := doRequest(http.MethodGet, sessionLocation, "")
	if archivedWorkspace.Code != http.StatusOK || !strings.Contains(archivedWorkspace.Body.String(), "This lesson is archived") || !strings.Contains(archivedWorkspace.Body.String(), lessons[0].Title) {
		t.Fatalf("archived session history status = %d or read-only message missing", archivedWorkspace.Code)
	}
	generatedCourseForm := url.Values{"title": {"Earth science"}, "subject": {"Science"}, "description": {"Explore the systems that shape our planet."}}
	generatedCourseResponse := postTeacherForm("/teacher/courses", generatedCourseForm)
	if generatedCourseResponse.Code != http.StatusSeeOther {
		t.Fatalf("create agentic course status = %d", generatedCourseResponse.Code)
	}
	generatedCourseID := strings.Split(strings.TrimPrefix(generatedCourseResponse.Header().Get("Location"), "/teacher/courses/"), "?")[0]
	generatedResponse := httptest.NewRequest(http.MethodPost, "/teacher/courses/"+generatedCourseID+"/curriculum/generate", nil)
	generatedResponse.AddCookie(teacherCookie)
	generatedRecorder := httptest.NewRecorder()
	mux.ServeHTTP(generatedRecorder, generatedResponse)
	if generatedRecorder.Code != http.StatusSeeOther {
		t.Fatalf("agent curriculum generation status = %d: %s", generatedRecorder.Code, generatedRecorder.Body.String())
	}
	generatedModules, err := courseRepo.ListModules(context.Background(), generatedCourseID)
	if err != nil || len(generatedModules) != 2 {
		t.Fatalf("generated module count = %d, err %v", len(generatedModules), err)
	}
	generatedLessons, generatedContents := 0, 0
	for _, module := range generatedModules {
		generatedChapters, listErr := courseRepo.ListChapters(context.Background(), module.ID)
		if listErr != nil || len(generatedChapters) != 2 {
			t.Fatalf("generated chapter count = %d, err %v", len(generatedChapters), listErr)
		}
		for _, chapter := range generatedChapters {
			lessons, listErr := courseRepo.ListLessons(context.Background(), chapter.ID)
			if listErr != nil {
				t.Fatalf("read generated lessons: %v", listErr)
			}
			generatedLessons += len(lessons)
			for _, lesson := range lessons {
				contents, listErr := courseRepo.ListLessonContents(context.Background(), lesson.ID)
				if listErr != nil {
					t.Fatalf("read generated lesson content: %v", listErr)
				}
				generatedContents += len(contents)
				for _, content := range contents {
					if content.Published {
						t.Fatal("generated lesson content must remain unpublished for teacher review")
					}
				}
			}
		}
	}
	if generatedLessons != 4 || generatedContents != 4 {
		t.Fatalf("generated lesson/content counts = %d/%d, want 4/4", generatedLessons, generatedContents)
	}
	var authoringLogCount int
	if err := db.QueryRow(`SELECT count(*) FROM agent_logs WHERE agent_name='CurriculumAuthoringAgent' AND action='generate_curriculum_draft'`).Scan(&authoringLogCount); err != nil || authoringLogCount != 1 {
		t.Fatalf("authoring execution log count = %d, err %v", authoringLogCount, err)
	}
	teacherEnrollmentPage := httptest.NewRequest(http.MethodGet, "/teacher/enrollments", nil)
	teacherEnrollmentPage.AddCookie(teacherCookie)
	teacherEnrollmentRecorder := httptest.NewRecorder()
	mux.ServeHTTP(teacherEnrollmentRecorder, teacherEnrollmentPage)
	if teacherEnrollmentRecorder.Code != http.StatusOK || !strings.Contains(teacherEnrollmentRecorder.Body.String(), "Alex Morgan") || !strings.Contains(teacherEnrollmentRecorder.Body.String(), "Jordan Lee") {
		t.Fatalf("teacher enrollment requests status = %d or student requests missing", teacherEnrollmentRecorder.Code)
	}
	for _, decision := range []struct {
		studentID string
		status    domain.EnrollmentStatus
	}{{"S-001", domain.EnrollmentApproved}, {"S-002", domain.EnrollmentDenied}} {
		decisionForm := url.Values{"student_id": {decision.studentID}, "course_id": {additionalCourse.ID}, "decision": {string(decision.status)}}
		decisionRequest := httptest.NewRequest(http.MethodPost, "/teacher/enrollments", strings.NewReader(decisionForm.Encode()))
		decisionRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		decisionRequest.AddCookie(teacherCookie)
		decisionRecorder := httptest.NewRecorder()
		mux.ServeHTTP(decisionRecorder, decisionRequest)
		if decisionRecorder.Code != http.StatusSeeOther {
			t.Fatalf("teacher enrollment decision status = %d, want redirect", decisionRecorder.Code)
		}
		storedEnrollment, found, findErr := courseRepo.FindCourseEnrollment(context.Background(), decision.studentID, additionalCourse.ID)
		if findErr != nil || !found || storedEnrollment.Status != decision.status {
			t.Fatalf("enrollment decision for %s = %s, found %v, err %v", decision.studentID, storedEnrollment.Status, found, findErr)
		}
	}
	studentProfile := doRequest(http.MethodGet, "/profile", "")
	if studentProfile.Code != http.StatusOK || !strings.Contains(studentProfile.Body.String(), "Physics basics") || !strings.Contains(studentProfile.Body.String(), "Maya Chen") {
		t.Fatalf("profile enrollment details status = %d or course/teacher details missing", studentProfile.Code)
	}
	studentOverviewAfterApproval := doRequest(http.MethodGet, "/student", "")
	if studentOverviewAfterApproval.Code != http.StatusOK || !strings.Contains(studentOverviewAfterApproval.Body.String(), "Physics basics") || !strings.Contains(studentOverviewAfterApproval.Body.String(), "Intro to astronomy") || !strings.Contains(studentOverviewAfterApproval.Body.String(), "Professor") {
		t.Fatalf("student overview did not show all approved courses: status %d", studentOverviewAfterApproval.Code)
	}
}

func TestWorkspaceSignupPersistsRoleSpecificProfile(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(projectRoot)
	dataDir := t.TempDir()
	t.Setenv("COLEARN_DB_PATH", filepath.Join(dataDir, "colearning.db"))
	t.Setenv("LEARNING_STORE_PATH", filepath.Join(dataDir, "learning-state.json"))
	t.Setenv("COLEARN_JWT_SECRET", strings.Repeat("s", 32))
	app, err := bootstrap.NewApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	mux := http.NewServeMux()
	routes.Register(mux, app.Routes)
	body := `{"role":"student","name":"New Student","email":"new.student@example.com","password":"student123","academic_program":"Computer Science","branch":"Systems","semester":3,"academic_year":"2026"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("signup status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var cookie *http.Cookie
	for _, candidate := range recorder.Result().Cookies() {
		if candidate.Name == "auth_token" {
			cookie = candidate
		}
	}
	if cookie == nil {
		t.Fatal("signup did not preserve a session token")
	}
	profile := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	profile.AddCookie(cookie)
	profileRecorder := httptest.NewRecorder()
	mux.ServeHTTP(profileRecorder, profile)
	if profileRecorder.Code != http.StatusOK || !strings.Contains(profileRecorder.Body.String(), `"academic_program":"Computer Science"`) {
		t.Fatalf("profile did not contain persisted role-specific data: %s", profileRecorder.Body.String())
	}
}
