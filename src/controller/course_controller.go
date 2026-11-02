package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type courseCatalog interface {
	FindCourse(context.Context, string) (domain.LearningCourse, bool, error)
	FindConcept(context.Context, string) (domain.LearningConcept, bool, error)
	ListCourses(context.Context) ([]domain.LearningCourse, error)
	ListConcepts(context.Context, string) ([]domain.LearningConcept, error)
	SaveCourse(context.Context, domain.LearningCourse) error
}

type CourseEnrollmentStore interface {
	RequestCourseEnrollment(context.Context, string, string) error
	FindCourseEnrollment(context.Context, string, string) (domain.CourseEnrollment, bool, error)
	ListStudentEnrollments(context.Context, string) ([]domain.CourseEnrollment, error)
	ListTeacherEnrollmentRequests(context.Context, string) ([]domain.CourseEnrollment, error)
	ReviewCourseEnrollment(context.Context, string, string, string, domain.EnrollmentStatus) error
}

type accountDirectory interface {
	FindAccountByID(context.Context, string) (auth.Account, bool, error)
}

type CourseController struct {
	renderer    *view.Renderer
	courses     courseCatalog
	curriculum  repository.CurriculumRepository
	enrollments CourseEnrollmentStore
	accounts    accountDirectory
}

func NewCourseController(renderer *view.Renderer, courses courseCatalog, curriculum repository.CurriculumRepository, enrollments CourseEnrollmentStore, accounts accountDirectory) CourseController {
	return CourseController{renderer: renderer, courses: courses, curriculum: curriculum, enrollments: enrollments, accounts: accounts}
}

type courseCardView struct {
	Course            domain.LearningCourse
	TeacherName       string
	TeacherEmail      string
	TeacherTitle      string
	TeacherDepartment string
	Enrollment        *domain.CourseEnrollment
}

type createCourseRequest struct {
	Title       string `json:"title"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

func (c CourseController) CreateCourseJSON(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, fmt.Errorf("teacher identity required"))
		return
	}
	var req createCourseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Description = strings.TrimSpace(req.Description)
	if req.Title == "" || req.Subject == "" || len([]rune(req.Title)) > 160 || len([]rune(req.Subject)) > 100 || len([]rune(req.Description)) > 4000 {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("title and subject are required; description must be under 4,000 characters"))
		return
	}
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Errorf("could not create course"))
		return
	}
	course := domain.LearningCourse{
		ID:          "course-" + hex.EncodeToString(randomID[:]),
		TeacherID:   identity.ID,
		TeacherName: identity.Name,
		Title:       req.Title,
		Subject:     req.Subject,
		Description: req.Description,
		TutorPolicy: domain.TutorPolicy{
			DefaultMode:   domain.TutorModeGuided,
			AllowedModes:  []domain.TutorMode{domain.TutorModeGuided, domain.TutorModeSocratic, domain.TutorModeDirect, domain.TutorModePractice, domain.TutorModeHint, domain.TutorModeDiagnostic},
			SourcePolicy: "Use published teacher material as the primary source.",
		},
	}
	if err := c.courses.SaveCourse(r.Context(), course); err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Errorf("could not save course"))
		return
	}
	writeJSON(w, http.StatusCreated, toAPICourse(course))
}

func (c CourseController) StudentCatalog(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	courses, err := c.courses.ListCourses(r.Context())
	if err != nil {
		http.Error(w, "Could not load courses", http.StatusInternalServerError)
		return
	}
	cards := make([]courseCardView, 0, len(courses))
	for _, course := range courses {
		concepts, listErr := c.courses.ListConcepts(r.Context(), course.ID)
		if listErr != nil {
			http.Error(w, "Could not load course outline", http.StatusInternalServerError)
			return
		}
		course.ConceptCount = len(concepts)
		card := courseCardView{Course: course, TeacherName: course.TeacherName}
		if teacher, found, lookupErr := c.accounts.FindAccountByID(r.Context(), course.TeacherID); lookupErr == nil && found {
			card.TeacherName, card.TeacherEmail, card.TeacherTitle, card.TeacherDepartment = teacher.Name, teacher.Email, teacher.TeachingTitle, teacher.Department
			if card.TeacherTitle == "" {
				card.TeacherTitle = teacher.Department
			}
		}
		if enrollment, found, lookupErr := c.enrollments.FindCourseEnrollment(r.Context(), identity.ID, course.ID); lookupErr != nil {
			http.Error(w, "Could not load enrollment status", http.StatusInternalServerError)
			return
		} else if found {
			card.Enrollment = &enrollment
		}
		cards = append(cards, card)
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "student/courses.html", map[string]any{"Identity": identity, "Courses": cards, "Notice": r.URL.Query().Get("notice")})
}

func (c CourseController) StudentCourseDetails(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	id := r.PathValue("course_id")
	course, found, err := c.courses.FindCourse(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load course", http.StatusInternalServerError)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	teacherName, teacherEmail, teacherTitle, teacherDepartment := course.TeacherName, "", "", ""
	if teacher, exists, lookupErr := c.accounts.FindAccountByID(r.Context(), course.TeacherID); lookupErr == nil && exists {
		teacherName, teacherEmail, teacherTitle, teacherDepartment = teacher.Name, teacher.Email, teacher.TeachingTitle, teacher.Department
		if teacherTitle == "" {
			teacherTitle = teacher.Department
		}
	}
	enrollment, enrolled, err := c.enrollments.FindCourseEnrollment(r.Context(), identity.ID, course.ID)
	if err != nil {
		http.Error(w, "Could not load enrollment", http.StatusInternalServerError)
		return
	}
	concepts, err := c.courses.ListConcepts(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "Could not load course outline", http.StatusInternalServerError)
		return
	}
	course.ConceptCount = len(concepts)
	modules, err := c.curriculum.ListModules(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "Could not load course outline", http.StatusInternalServerError)
		return
	}
	outline := make([]moduleView, 0, len(modules))
	for _, module := range modules {
		moduleItem := moduleView{Module: module}
		chapters, listErr := c.curriculum.ListChapters(r.Context(), module.ID)
		if listErr != nil {
			http.Error(w, "Could not load course outline", http.StatusInternalServerError)
			return
		}
		for _, chapter := range chapters {
			lessons, listErr := c.curriculum.ListLessons(r.Context(), chapter.ID)
			if listErr != nil {
				http.Error(w, "Could not load course outline", http.StatusInternalServerError)
				return
			}
			moduleItem.Chapters = append(moduleItem.Chapters, chapterView{Chapter: chapter, Lessons: make([]lessonView, len(lessons))})
			for i, lesson := range lessons {
				moduleItem.Chapters[len(moduleItem.Chapters)-1].Lessons[i] = lessonView{Lesson: lesson}
			}
		}
		outline = append(outline, moduleItem)
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "student/course.html", map[string]any{
		"Identity": identity, "Course": course, "TeacherName": teacherName, "TeacherEmail": teacherEmail, "TeacherTitle": teacherTitle, "TeacherDepartment": teacherDepartment,
		"Enrollment": enrollment, "HasEnrollment": enrolled, "Modules": outline,
	})
}

func (c CourseController) RequestEnrollment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	id := r.PathValue("course_id")
	if _, found, err := c.courses.FindCourse(r.Context(), id); err != nil {
		http.Error(w, "Could not load course", http.StatusInternalServerError)
		return
	} else if !found {
		http.NotFound(w, r)
		return
	}
	if err := c.enrollments.RequestCourseEnrollment(r.Context(), identity.ID, id); err != nil {
		http.Error(w, "Could not request enrollment", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/student/courses?notice="+url.QueryEscape("Enrollment request sent"), http.StatusSeeOther)
}

func (c CourseController) TeacherRequests(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	requests, err := c.enrollments.ListTeacherEnrollmentRequests(r.Context(), identity.ID)
	if err != nil {
		http.Error(w, "Could not load enrollment requests", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "faculty/enrollments.html", map[string]any{"Identity": identity, "ActiveNav": "enrollments", "Requests": requests, "Notice": r.URL.Query().Get("notice"), "Error": r.URL.Query().Get("error")})
}

func (c CourseController) ReviewEnrollment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/teacher/enrollments?error=Invalid+review+form", http.StatusSeeOther)
		return
	}
	status := domain.EnrollmentStatus(strings.TrimSpace(r.FormValue("decision")))
	if status != domain.EnrollmentApproved && status != domain.EnrollmentDenied {
		http.Redirect(w, r, "/teacher/enrollments?error=Choose+approve+or+deny", http.StatusSeeOther)
		return
	}
	err := c.enrollments.ReviewCourseEnrollment(r.Context(), r.FormValue("student_id"), r.FormValue("course_id"), identity.ID, status)
	if err != nil {
		http.Redirect(w, r, "/teacher/enrollments?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	message := fmt.Sprintf("Enrollment %s", status)
	http.Redirect(w, r, "/teacher/enrollments?notice="+url.QueryEscape(message), http.StatusSeeOther)
}

func (c CourseController) TeacherCourseList(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	courses, err := c.courses.ListCourses(r.Context())
	if err != nil {
		http.Error(w, "Could not load courses", http.StatusInternalServerError)
		return
	}
	owned := make([]domain.LearningCourse, 0)
	for _, course := range courses {
		if course.TeacherID == identity.ID {
			concepts, err := c.courses.ListConcepts(r.Context(), course.ID)
			if err != nil {
				http.Error(w, "Could not load course outline", http.StatusInternalServerError)
				return
			}
			course.ConceptCount = len(concepts)
			owned = append(owned, course)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "faculty/courses.html", map[string]any{"Identity": identity, "ActiveNav": "courses", "Courses": owned, "Notice": r.URL.Query().Get("notice")})
}

func (c CourseController) TeacherCourseEditor(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	course := domain.LearningCourse{TeacherID: identity.ID, TeacherName: identity.Name, TutorPolicy: domain.TutorPolicy{DefaultMode: domain.TutorModeGuided, AllowedModes: []domain.TutorMode{domain.TutorModeGuided, domain.TutorModeSocratic, domain.TutorModeDirect, domain.TutorModePractice, domain.TutorModeHint, domain.TutorModeDiagnostic}, SourcePolicy: "Use published teacher material as the primary source."}}
	if id := r.PathValue("course_id"); id != "" {
		var found bool
		var err error
		course, found, err = c.courses.FindCourse(r.Context(), id)
		if err != nil {
			http.Error(w, "Could not load course", http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}
		if course.TeacherID != identity.ID {
			http.Error(w, "course belongs to another teacher", http.StatusForbidden)
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderTeacherCourseEditor(w, identity, course, course.ID == "", r.URL.Query().Get("notice"), r.URL.Query().Get("error"))
}

func (c CourseController) renderTeacherCourseEditor(w http.ResponseWriter, identity auth.Identity, course domain.LearningCourse, isNew bool, notice, errMessage string) {
	c.renderer.Render(w, "faculty/course_form.html", map[string]any{
		"Identity": identity, "ActiveNav": "courses", "Course": course, "IsNew": isNew,
		"Notice": notice, "Error": errMessage,
	})
}

func (c CourseController) SaveTeacherCourse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil {
		location := "/teacher/courses/new"
		if id := r.PathValue("course_id"); id != "" {
			location = "/teacher/courses/" + url.PathEscape(id)
		}
		http.Redirect(w, r, location+"?error="+url.QueryEscape("The course form could not be read. Please try again."), http.StatusSeeOther)
		return
	}
	id := r.PathValue("course_id")
	var course domain.LearningCourse
	if id != "" {
		var found bool
		var err error
		course, found, err = c.courses.FindCourse(r.Context(), id)
		if err != nil {
			http.Error(w, "Could not load course", http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}
		if course.TeacherID != identity.ID {
			http.Error(w, "course belongs to another teacher", http.StatusForbidden)
			return
		}
	} else {
		var randomID [16]byte
		if _, err := rand.Read(randomID[:]); err != nil {
			http.Error(w, "could not create course", http.StatusInternalServerError)
			return
		}
		course.ID = "course-" + hex.EncodeToString(randomID[:])
		course.TeacherID = identity.ID
		course.TutorPolicy = domain.TutorPolicy{DefaultMode: domain.TutorModeGuided, AllowedModes: []domain.TutorMode{domain.TutorModeGuided, domain.TutorModeSocratic, domain.TutorModeDirect, domain.TutorModePractice, domain.TutorModeHint, domain.TutorModeDiagnostic}, SourcePolicy: "Use published teacher material as the primary source."}
	}
	title := strings.TrimSpace(r.FormValue("title"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	description := strings.TrimSpace(r.FormValue("description"))
	course.Title, course.Subject, course.Description = title, subject, description
	if title == "" || subject == "" || len([]rune(title)) > 160 || len([]rune(subject)) > 100 || len([]rune(description)) > 4000 {
		c.renderTeacherCourseEditor(w, identity, course, id == "", "", "Enter a course title and subject; keep the description under 4,000 characters.")
		return
	}
	course.TeacherName = identity.Name
	if err := c.courses.SaveCourse(r.Context(), course); err != nil {
		c.renderTeacherCourseEditor(w, identity, course, id == "", "", "Could not save this course. Your changes are still in the form; please try again.")
		return
	}
	http.Redirect(w, r, "/teacher/courses/"+url.PathEscape(course.ID)+"?notice="+url.QueryEscape("Course saved"), http.StatusSeeOther)
}
