package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/agentic"
	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type curriculumSavePayload struct {
	Modules  []domain.CourseModule  `json:"modules"`
	Chapters []domain.CourseChapter `json:"chapters"`
	Lessons  []domain.CourseLesson  `json:"lessons"`
	Contents []domain.LessonContent `json:"contents"`
}

type CurriculumController struct {
	renderer    *view.Renderer
	courses     courseCatalog
	curriculum  repository.CurriculumRepository
	content     repository.LearningContentRepository
	authoring   agentic.CurriculumAuthoringAgent
	enrollments CourseEnrollmentStore
}

func NewCurriculumController(renderer *view.Renderer, courses courseCatalog, curriculum repository.CurriculumRepository, content repository.LearningContentRepository, authoring agentic.CurriculumAuthoringAgent, enrollments CourseEnrollmentStore) CurriculumController {
	return CurriculumController{renderer: renderer, courses: courses, curriculum: curriculum, content: content, authoring: authoring, enrollments: enrollments}
}

type chapterView struct {
	Chapter domain.CourseChapter
	Lessons []lessonView
}

type lessonView struct {
	Lesson  domain.CourseLesson
	Content []domain.LessonContent
}

type moduleView struct {
	Module   domain.CourseModule
	Chapters []chapterView
}

func (c CurriculumController) StructureJSON(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, fmt.Errorf("identity required"))
		return
	}
	course, found, err := c.courses.FindCourse(r.Context(), r.PathValue("course_id"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Errorf("could not load course"))
		return
	}
	if !found {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("course not found"))
		return
	}
	teacher := identity.Role == "teacher" || identity.Role == "faculty"
	if teacher {
		if course.TeacherID != identity.ID {
			writeAPIError(w, http.StatusForbidden, fmt.Errorf("course access denied"))
			return
		}
	} else if c.enrollments != nil {
		enrollment, enrolled, enrollmentErr := c.enrollments.FindCourseEnrollment(r.Context(), identity.ID, course.ID)
		if enrollmentErr != nil {
			writeAPIError(w, http.StatusInternalServerError, fmt.Errorf("could not verify enrollment"))
			return
		}
		if !enrolled || enrollment.Status != domain.EnrollmentApproved {
			writeAPIError(w, http.StatusForbidden, fmt.Errorf("approved enrollment required"))
			return
		}
	}

	structure, err := c.curriculumJSON(r.Context(), course.ID, teacher)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, structure)
}

func (c CurriculumController) curriculumJSON(ctx context.Context, courseID string, includeDrafts bool) (apiCurriculum, error) {
	modules, err := c.curriculum.ListModules(ctx, courseID)
	if err != nil {
		return apiCurriculum{}, fmt.Errorf("could not load modules")
	}
	result := apiCurriculum{CourseID: courseID, Modules: make([]apiCurriculumModule, 0, len(modules))}
	for _, module := range modules {
		item := apiCurriculumModule{ID: module.ID, Title: module.Title, Description: module.Description, Order: module.Order}
		chapters, err := c.curriculum.ListChapters(ctx, module.ID)
		if err != nil {
			return apiCurriculum{}, fmt.Errorf("could not load chapters")
		}
		for _, chapter := range chapters {
			chapterItem := apiCurriculumChapter{ID: chapter.ID, Title: chapter.Title, Description: chapter.Description, Order: chapter.Order}
			lessons, err := c.curriculum.ListLessons(ctx, chapter.ID)
			if err != nil {
				return apiCurriculum{}, fmt.Errorf("could not load lessons")
			}
			for _, lesson := range lessons {
				lessonItem := apiCurriculumLesson{ID: lesson.ID, Title: lesson.Title, Summary: lesson.Summary, Order: lesson.Order}
				contents, err := c.curriculum.ListLessonContents(ctx, lesson.ID)
				if err != nil {
					return apiCurriculum{}, fmt.Errorf("could not load lesson content")
				}
				for _, content := range contents {
					if !includeDrafts && !content.Published {
						continue
					}
					lessonItem.Contents = append(lessonItem.Contents, apiCurriculumContent{ID: content.ID, Kind: content.Kind, Title: content.Title, Body: content.Body, Published: content.Published})
				}
				chapterItem.Lessons = append(chapterItem.Lessons, lessonItem)
			}
			item.Chapters = append(item.Chapters, chapterItem)
		}
		result.Modules = append(result.Modules, item)
	}
	return result, nil
}

func (c CurriculumController) Page(w http.ResponseWriter, r *http.Request) {
	identity, course, ok := c.ownedCourse(w, r)
	if !ok {
		return
	}
	modules, err := c.curriculum.ListModules(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "Could not load course modules", http.StatusInternalServerError)
		return
	}
	concepts, err := c.content.ListConcepts(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "Could not load course concepts", http.StatusInternalServerError)
		return
	}
	views := make([]moduleView, 0, len(modules))
	for _, module := range modules {
		item := moduleView{Module: module}
		chapters, err := c.curriculum.ListChapters(r.Context(), module.ID)
		if err != nil {
			http.Error(w, "Could not load course chapters", http.StatusInternalServerError)
			return
		}
		for _, chapter := range chapters {
			chapterItem := chapterView{Chapter: chapter}
			lessons, err := c.curriculum.ListLessons(r.Context(), chapter.ID)
			if err != nil {
				http.Error(w, "Could not load course lessons", http.StatusInternalServerError)
				return
			}
			for _, lesson := range lessons {
				content, err := c.curriculum.ListLessonContents(r.Context(), lesson.ID)
				if err != nil {
					http.Error(w, "Could not load lesson content", http.StatusInternalServerError)
					return
				}
				chapterItem.Lessons = append(chapterItem.Lessons, lessonView{Lesson: lesson, Content: content})
			}
			item.Chapters = append(item.Chapters, chapterItem)
		}
		views = append(views, item)
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "faculty/curriculum.html", map[string]any{"Identity": identity, "ActiveNav": "courses", "Course": course, "Modules": views, "CanGenerate": len(modules) == 0 && len(concepts) == 0, "Notice": r.URL.Query().Get("notice"), "Error": r.URL.Query().Get("error")})
}

func (c CurriculumController) GenerateDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, course, ok := c.ownedCourse(w, r)
	if !ok {
		return
	}
	modules, err := c.curriculum.ListModules(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "Could not inspect current curriculum", http.StatusInternalServerError)
		return
	}
	concepts, err := c.content.ListConcepts(r.Context(), course.ID)
	if err != nil {
		http.Error(w, "Could not inspect current concepts", http.StatusInternalServerError)
		return
	}
	if len(modules) != 0 || len(concepts) != 0 {
		http.Error(w, "Starter generation is only available before curriculum has been added.", http.StatusConflict)
		return
	}
	draft, err := c.authoring.Generate(r.Context(), course)
	if err != nil {
		http.Error(w, "Could not generate a curriculum draft: "+err.Error(), http.StatusBadGateway)
		return
	}
	conceptOrder := 0
	for moduleIndex, moduleDraft := range draft.Modules {
		moduleID, err := newCurriculumID("module")
		if err != nil {
			http.Error(w, "Could not store generated module", http.StatusInternalServerError)
			return
		}
		module := domain.CourseModule{ID: moduleID, CourseID: course.ID, Title: moduleDraft.Title, Description: moduleDraft.Summary, Order: moduleIndex + 1}
		if err := c.curriculum.SaveModule(r.Context(), module); err != nil {
			http.Error(w, "Could not store generated module", http.StatusInternalServerError)
			return
		}
		for chapterIndex, chapterDraft := range moduleDraft.Chapters {
			chapterID, err := newCurriculumID("chapter")
			if err != nil {
				http.Error(w, "Could not store generated chapter", http.StatusInternalServerError)
				return
			}
			chapter := domain.CourseChapter{ID: chapterID, CourseID: course.ID, ModuleID: module.ID, Title: chapterDraft.Title, Description: chapterDraft.Summary, Order: chapterIndex + 1}
			if err := c.curriculum.SaveChapter(r.Context(), chapter); err != nil {
				http.Error(w, "Could not store generated chapter", http.StatusInternalServerError)
				return
			}
			for lessonIndex, lessonDraft := range chapterDraft.Lessons {
				lessonID, err := newCurriculumID("lesson")
				if err != nil {
					http.Error(w, "Could not store generated lesson", http.StatusInternalServerError)
					return
				}
				lesson := domain.CourseLesson{ID: lessonID, CourseID: course.ID, ChapterID: chapter.ID, Title: lessonDraft.Title, Summary: lessonDraft.Summary, Order: lessonIndex + 1}
				if err := c.curriculum.SaveLesson(r.Context(), lesson); err != nil {
					http.Error(w, "Could not store generated lesson", http.StatusInternalServerError)
					return
				}
				conceptOrder++
				concept := domain.LearningConcept{ID: lesson.ID, CourseID: course.ID, Title: lesson.Title, Summary: lesson.Summary, Level: "Lesson", Order: conceptOrder, EstimatedMinutes: 15}
				if err := c.content.SaveConcept(r.Context(), concept); err != nil {
					http.Error(w, "Could not store generated lesson in the learning path", http.StatusInternalServerError)
					return
				}
				if err := c.content.SaveObjective(r.Context(), domain.LearningObjective{ID: "objective-" + lesson.ID, CourseID: course.ID, ConceptID: lesson.ID, Description: lesson.Summary}); err != nil {
					http.Error(w, "Could not store generated lesson objective", http.StatusInternalServerError)
					return
				}
				for contentIndex, contentDraft := range lessonDraft.Contents {
					contentID, err := newCurriculumID("content")
					if err != nil {
						http.Error(w, "Could not store generated lesson content", http.StatusInternalServerError)
						return
					}
					content := domain.LessonContent{ID: contentID, CourseID: course.ID, LessonID: lesson.ID, Kind: contentDraft.Kind, Title: contentDraft.Title, Body: contentDraft.Body, Order: contentIndex + 1}
					if err := c.curriculum.SaveLessonContent(r.Context(), content); err != nil {
						http.Error(w, "Could not store generated lesson content", http.StatusInternalServerError)
						return
					}
					material := domain.LearningMaterial{ID: content.ID, CourseID: course.ID, ConceptID: lesson.ID, Title: content.Title, Body: content.Body, Published: false, Version: 1, Source: domain.LearningSourceRef{ID: content.ID, Title: content.Title, Kind: domain.LearningSourceTeacher}}
					if err := c.content.SaveMaterial(r.Context(), material); err != nil {
						http.Error(w, "Could not add generated content to tutor context", http.StatusInternalServerError)
						return
					}
				}
			}
		}
	}
	course.ConceptCount = conceptOrder
	if conceptOrder > 0 {
		all, err := c.content.ListConcepts(r.Context(), course.ID)
		if err != nil {
			http.Error(w, "Could not update generated course outline", http.StatusInternalServerError)
			return
		}
		course.ConceptCount = len(all)
		course.NextConceptID = all[0].ID
	}
	if err := c.courses.SaveCourse(r.Context(), course); err != nil {
		http.Error(w, "Could not save generated course outline", http.StatusInternalServerError)
		return
	}
	c.curriculumRedirect(w, r, course.ID, "Draft outline generated. Review every lesson and publish the content you approve.")
}

// SaveAll persists the complete teacher-edited curriculum draft for one owned course.
func (c CurriculumController) SaveAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, course, ok := c.ownedCourse(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload curriculumSavePayload
	if err := decoder.Decode(&payload); err != nil {
		http.Redirect(w, r, "/teacher/courses/"+url.PathEscape(course.ID)+"/curriculum?error="+url.QueryEscape("Could not read curriculum changes"), http.StatusSeeOther)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Redirect(w, r, "/teacher/courses/"+url.PathEscape(course.ID)+"/curriculum?error="+url.QueryEscape("Send one curriculum draft at a time"), http.StatusSeeOther)
		return
	}
	if err := c.persistCurriculumDraft(r, course, payload); err != nil {
		http.Redirect(w, r, "/teacher/courses/"+url.PathEscape(course.ID)+"/curriculum?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	c.curriculumRedirect(w, r, course.ID, "All curriculum changes saved")
}

func (c CurriculumController) persistCurriculumDraft(r *http.Request, course domain.LearningCourse, payload curriculumSavePayload) error {
	if len(payload.Modules) > 100 || len(payload.Chapters) > 500 || len(payload.Lessons) > 2000 || len(payload.Contents) > 5000 {
		return fmt.Errorf("curriculum exceeds the supported size")
	}
	modules := make(map[string]domain.CourseModule, len(payload.Modules))
	for i, item := range payload.Modules {
		item.CourseID, item.Order = course.ID, i+1
		item.Title, item.Description = strings.TrimSpace(item.Title), strings.TrimSpace(item.Description)
		if item.ID == "" || !validCurriculumText(item.Title, 160) || !validOptionalCurriculumText(item.Description, 3000) {
			return fmt.Errorf("check module titles and descriptions")
		}
		if _, duplicate := modules[item.ID]; duplicate {
			return fmt.Errorf("duplicate module in curriculum")
		}
		modules[item.ID] = item
	}
	chapters := make(map[string]domain.CourseChapter, len(payload.Chapters))
	chapterOrder := map[string]int{}
	for _, item := range payload.Chapters {
		if _, exists := modules[item.ModuleID]; !exists {
			return fmt.Errorf("a chapter references a module outside this course")
		}
		chapterOrder[item.ModuleID]++
		item.CourseID, item.Order = course.ID, chapterOrder[item.ModuleID]
		item.Title, item.Description = strings.TrimSpace(item.Title), strings.TrimSpace(item.Description)
		if item.ID == "" || !validCurriculumText(item.Title, 160) || !validOptionalCurriculumText(item.Description, 3000) {
			return fmt.Errorf("check chapter titles and descriptions")
		}
		if _, duplicate := chapters[item.ID]; duplicate {
			return fmt.Errorf("duplicate chapter in curriculum")
		}
		chapters[item.ID] = item
	}
	lessons := make(map[string]domain.CourseLesson, len(payload.Lessons))
	lessonOrder := map[string]int{}
	for _, item := range payload.Lessons {
		if _, exists := chapters[item.ChapterID]; !exists {
			return fmt.Errorf("a lesson references a chapter outside this course")
		}
		lessonOrder[item.ChapterID]++
		item.CourseID, item.Order = course.ID, lessonOrder[item.ChapterID]
		item.Title, item.Summary = strings.TrimSpace(item.Title), strings.TrimSpace(item.Summary)
		if item.ID == "" || !validCurriculumText(item.Title, 160) || !validOptionalCurriculumText(item.Summary, 3000) {
			return fmt.Errorf("check lesson titles and summaries")
		}
		if _, duplicate := lessons[item.ID]; duplicate {
			return fmt.Errorf("duplicate lesson in curriculum")
		}
		lessons[item.ID] = item
	}
	contents := make(map[string]domain.LessonContent, len(payload.Contents))
	contentOrder := map[string]int{}
	for _, item := range payload.Contents {
		if _, exists := lessons[item.LessonID]; !exists {
			return fmt.Errorf("lesson content references a lesson outside this course")
		}
		contentOrder[item.LessonID]++
		item.CourseID, item.Order = course.ID, contentOrder[item.LessonID]
		item.Title, item.Body = strings.TrimSpace(item.Title), strings.TrimSpace(item.Body)
		if item.ID == "" || !validCurriculumText(item.Title, 200) || !validCurriculumText(item.Body, 12000) {
			return fmt.Errorf("check lesson content titles and text")
		}
		if item.Kind == "" {
			item.Kind = "lesson_note"
		}
		switch item.Kind {
		case "lesson_note", "example", "activity", "reference":
		default:
			return fmt.Errorf("choose a supported lesson content type")
		}
		if _, duplicate := contents[item.ID]; duplicate {
			return fmt.Errorf("duplicate content item in curriculum")
		}
		contents[item.ID] = item
	}
	draft := repository.CurriculumDraft{Course: course}
	for _, item := range payload.Modules {
		draft.Modules = append(draft.Modules, modules[item.ID])
	}
	for _, item := range payload.Chapters {
		draft.Chapters = append(draft.Chapters, chapters[item.ID])
	}
	for _, item := range payload.Lessons {
		draft.Lessons = append(draft.Lessons, lessons[item.ID])
	}
	for _, item := range payload.Contents {
		draft.Contents = append(draft.Contents, contents[item.ID])
	}
	draft.Course.NextConceptID = ""
	if len(draft.Lessons) > 0 {
		draft.Course.NextConceptID = draft.Lessons[0].ID
	}
	saver, ok := c.curriculum.(repository.CurriculumDraftRepository)
	if !ok {
		return fmt.Errorf("atomic curriculum save is unavailable")
	}
	if err := saver.SaveCurriculumDraft(r.Context(), draft); err != nil {
		return fmt.Errorf("curriculum draft was not saved: %w", err)
	}
	return nil
}

func (c CurriculumController) SaveModule(w http.ResponseWriter, r *http.Request) {
	if !c.parseForm(w, r) {
		return
	}
	courseID := r.FormValue("course_id")
	if _, course, ok := c.ownedCourseByID(w, r, courseID); !ok {
		return
	} else {
		module := domain.CourseModule{ID: r.FormValue("id"), CourseID: course.ID, Title: strings.TrimSpace(r.FormValue("title")), Description: strings.TrimSpace(r.FormValue("description"))}
		if !validCurriculumText(module.Title, 160) || !validOptionalCurriculumText(module.Description, 3000) {
			c.badCurriculumForm(w, r, course.ID, "Enter a module title and keep the optional description under 3,000 characters.")
			return
		}
		if module.ID == "" {
			var err error
			module.ID, err = newCurriculumID("module")
			if err != nil {
				http.Error(w, "Could not create module", http.StatusInternalServerError)
				return
			}
			items, err := c.curriculum.ListModules(r.Context(), course.ID)
			if err != nil {
				http.Error(w, "Could not order modules", http.StatusInternalServerError)
				return
			}
			module.Order = len(items) + 1
		} else {
			old, found, err := c.curriculum.FindModule(r.Context(), module.ID)
			if err != nil || !found || old.CourseID != course.ID {
				http.Error(w, "Module not found for this course", http.StatusNotFound)
				return
			}
			module.Order = old.Order
		}
		if err := c.curriculum.SaveModule(r.Context(), module); err != nil {
			http.Error(w, "Could not save module", http.StatusInternalServerError)
			return
		}
		c.curriculumRedirect(w, r, course.ID, "Module saved")
	}
}

func (c CurriculumController) SaveChapter(w http.ResponseWriter, r *http.Request) {
	if !c.parseForm(w, r) {
		return
	}
	moduleID := r.FormValue("module_id")
	module, found, err := c.curriculum.FindModule(r.Context(), moduleID)
	if err != nil || !found {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}
	if _, _, ok := c.ownedCourseByID(w, r, module.CourseID); !ok {
		return
	}
	chapter := domain.CourseChapter{ID: r.FormValue("id"), CourseID: module.CourseID, ModuleID: module.ID, Title: strings.TrimSpace(r.FormValue("title")), Description: strings.TrimSpace(r.FormValue("description"))}
	if !validCurriculumText(chapter.Title, 160) || !validOptionalCurriculumText(chapter.Description, 3000) {
		c.badCurriculumForm(w, r, module.CourseID, "Enter a chapter title and keep the optional description under 3,000 characters.")
		return
	}
	if chapter.ID == "" {
		chapter.ID, err = newCurriculumID("chapter")
		if err != nil {
			http.Error(w, "Could not create chapter", http.StatusInternalServerError)
			return
		}
		items, err := c.curriculum.ListChapters(r.Context(), module.ID)
		if err != nil {
			http.Error(w, "Could not order chapters", http.StatusInternalServerError)
			return
		}
		chapter.Order = len(items) + 1
	} else {
		old, exists, err := c.curriculum.FindChapter(r.Context(), chapter.ID)
		if err != nil || !exists || old.CourseID != module.CourseID {
			http.Error(w, "Chapter not found for this course", http.StatusNotFound)
			return
		}
		chapter.Order = old.Order
	}
	if err := c.curriculum.SaveChapter(r.Context(), chapter); err != nil {
		http.Error(w, "Could not save chapter", http.StatusInternalServerError)
		return
	}
	c.curriculumRedirect(w, r, module.CourseID, "Chapter saved")
}

func (c CurriculumController) SaveLesson(w http.ResponseWriter, r *http.Request) {
	if !c.parseForm(w, r) {
		return
	}
	chapterID := r.FormValue("chapter_id")
	chapter, found, err := c.curriculum.FindChapter(r.Context(), chapterID)
	if err != nil || !found {
		http.Error(w, "Chapter not found", http.StatusNotFound)
		return
	}
	if _, _, ok := c.ownedCourseByID(w, r, chapter.CourseID); !ok {
		return
	}
	lesson := domain.CourseLesson{ID: r.FormValue("id"), CourseID: chapter.CourseID, ChapterID: chapter.ID, Title: strings.TrimSpace(r.FormValue("title")), Summary: strings.TrimSpace(r.FormValue("summary"))}
	if !validCurriculumText(lesson.Title, 160) || !validOptionalCurriculumText(lesson.Summary, 3000) {
		c.badCurriculumForm(w, r, chapter.CourseID, "Enter a lesson title and keep the optional summary under 3,000 characters.")
		return
	}
	if lesson.ID == "" {
		lesson.ID, err = newCurriculumID("lesson")
		if err != nil {
			http.Error(w, "Could not create lesson", http.StatusInternalServerError)
			return
		}
		items, err := c.curriculum.ListLessons(r.Context(), chapter.ID)
		if err != nil {
			http.Error(w, "Could not order lessons", http.StatusInternalServerError)
			return
		}
		lesson.Order = len(items) + 1
	} else {
		old, exists, err := c.curriculum.FindLesson(r.Context(), lesson.ID)
		if err != nil || !exists || old.CourseID != chapter.CourseID {
			http.Error(w, "Lesson not found for this course", http.StatusNotFound)
			return
		}
		lesson.Order = old.Order
	}
	if err := c.curriculum.SaveLesson(r.Context(), lesson); err != nil {
		http.Error(w, "Could not save lesson", http.StatusInternalServerError)
		return
	}
	concept := domain.LearningConcept{ID: lesson.ID, CourseID: lesson.CourseID, Title: lesson.Title, Summary: lesson.Summary, Level: "Lesson", Order: lesson.Order, EstimatedMinutes: 15}
	if err := c.content.SaveConcept(r.Context(), concept); err != nil {
		http.Error(w, "Lesson saved but learning path could not be updated", http.StatusInternalServerError)
		return
	}
	if err := c.content.SaveObjective(r.Context(), domain.LearningObjective{ID: "objective-" + lesson.ID, CourseID: lesson.CourseID, ConceptID: lesson.ID, Description: lesson.Summary}); err != nil {
		http.Error(w, "Lesson saved but learning objective could not be updated", http.StatusInternalServerError)
		return
	}
	course, exists, err := c.courses.FindCourse(r.Context(), lesson.CourseID)
	if err != nil || !exists {
		http.Error(w, "Could not update course outline", http.StatusInternalServerError)
		return
	}
	concepts, err := c.content.ListConcepts(r.Context(), lesson.CourseID)
	if err != nil {
		http.Error(w, "Could not count course lessons", http.StatusInternalServerError)
		return
	}
	course.ConceptCount = len(concepts)
	if course.NextConceptID == "" {
		course.NextConceptID = lesson.ID
	}
	if err := c.courses.SaveCourse(r.Context(), course); err != nil {
		http.Error(w, "Lesson saved but course outline could not be updated", http.StatusInternalServerError)
		return
	}
	c.curriculumRedirect(w, r, lesson.CourseID, "Lesson saved")
}

func (c CurriculumController) SaveContent(w http.ResponseWriter, r *http.Request) {
	if !c.parseForm(w, r) {
		return
	}
	lessonID := r.FormValue("lesson_id")
	lesson, found, err := c.curriculum.FindLesson(r.Context(), lessonID)
	if err != nil || !found {
		http.Error(w, "Lesson not found", http.StatusNotFound)
		return
	}
	if _, _, ok := c.ownedCourseByID(w, r, lesson.CourseID); !ok {
		return
	}
	content := domain.LessonContent{ID: r.FormValue("id"), CourseID: lesson.CourseID, LessonID: lesson.ID, Kind: strings.TrimSpace(r.FormValue("kind")), Title: strings.TrimSpace(r.FormValue("title")), Body: strings.TrimSpace(r.FormValue("body")), Published: r.FormValue("published") == "on"}
	if !validCurriculumText(content.Title, 200) || !validCurriculumText(content.Body, 12000) {
		c.badCurriculumForm(w, r, lesson.CourseID, "Enter a content title and text under 12,000 characters.")
		return
	}
	if content.Kind == "" {
		content.Kind = "lesson_note"
	}
	if content.ID == "" {
		content.ID, err = newCurriculumID("content")
		if err != nil {
			http.Error(w, "Could not create lesson content", http.StatusInternalServerError)
			return
		}
		items, err := c.curriculum.ListLessonContents(r.Context(), lesson.ID)
		if err != nil {
			http.Error(w, "Could not order lesson content", http.StatusInternalServerError)
			return
		}
		content.Order = len(items) + 1
	} else {
		old, exists, err := c.curriculum.FindLessonContent(r.Context(), content.ID)
		if err != nil || !exists || old.LessonID != lesson.ID {
			http.Error(w, "Content item not found for this lesson", http.StatusNotFound)
			return
		}
		content.Order = old.Order
	}
	if err := c.curriculum.SaveLessonContent(r.Context(), content); err != nil {
		http.Error(w, "Could not save lesson content", http.StatusInternalServerError)
		return
	}
	materials, err := c.content.ListMaterials(r.Context(), lesson.CourseID, lesson.ID)
	if err != nil {
		http.Error(w, "Could not sync tutor materials", http.StatusInternalServerError)
		return
	}
	version := 1
	for _, material := range materials {
		if material.ID == content.ID {
			version = material.Version + 1
		}
	}
	material := domain.LearningMaterial{ID: content.ID, CourseID: lesson.CourseID, ConceptID: lesson.ID, Title: content.Title, Body: content.Body, Published: content.Published, Version: version, Source: domain.LearningSourceRef{ID: content.ID, Title: content.Title, Kind: domain.LearningSourceTeacher}}
	if err := c.content.SaveMaterial(r.Context(), material); err != nil {
		http.Error(w, "Lesson content saved but tutor source could not be updated", http.StatusInternalServerError)
		return
	}
	c.curriculumRedirect(w, r, lesson.CourseID, "Lesson content saved")
}

func (c CurriculumController) ownedCourse(w http.ResponseWriter, r *http.Request) (auth.Identity, domain.LearningCourse, bool) {
	return c.ownedCourseByID(w, r, r.PathValue("course_id"))
}

func (c CurriculumController) ownedCourseByID(w http.ResponseWriter, r *http.Request, courseID string) (auth.Identity, domain.LearningCourse, bool) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return auth.Identity{}, domain.LearningCourse{}, false
	}
	course, found, err := c.courses.FindCourse(r.Context(), courseID)
	if err != nil {
		http.Error(w, "Could not load course", http.StatusInternalServerError)
		return auth.Identity{}, domain.LearningCourse{}, false
	}
	if !found {
		http.NotFound(w, r)
		return auth.Identity{}, domain.LearningCourse{}, false
	}
	if course.TeacherID != identity.ID {
		http.Error(w, "course belongs to another teacher", http.StatusForbidden)
		return auth.Identity{}, domain.LearningCourse{}, false
	}
	return identity, course, true
}

func (c CurriculumController) parseForm(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 24<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid curriculum form", http.StatusBadRequest)
		return false
	}
	return true
}

func (c CurriculumController) badCurriculumForm(w http.ResponseWriter, r *http.Request, courseID, message string) {
	location := fmt.Sprintf("/teacher/courses/%s/curriculum?error=%s", url.PathEscape(courseID), url.QueryEscape(message))
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func (c CurriculumController) curriculumRedirect(w http.ResponseWriter, r *http.Request, courseID, notice string) {
	http.Redirect(w, r, fmt.Sprintf("/teacher/courses/%s/curriculum?notice=%s", url.PathEscape(courseID), url.QueryEscape(notice)), http.StatusSeeOther)
}

func validCurriculumText(value string, max int) bool {
	return value != "" && len([]rune(value)) <= max
}

func validOptionalCurriculumText(value string, max int) bool {
	return len([]rune(value)) <= max
}

func newCurriculumID(prefix string) (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(id[:]), nil
}
