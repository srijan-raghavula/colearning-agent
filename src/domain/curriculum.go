package domain

type CourseModule struct {
	ID          string `json:"id"`
	CourseID    string `json:"course_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

type CourseChapter struct {
	ID          string `json:"id"`
	CourseID    string `json:"course_id"`
	ModuleID    string `json:"module_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

type CourseLesson struct {
	ID        string `json:"id"`
	CourseID  string `json:"course_id"`
	ChapterID string `json:"chapter_id"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Order     int    `json:"order"`
}

type LessonContent struct {
	ID        string `json:"id"`
	CourseID  string `json:"course_id"`
	LessonID  string `json:"lesson_id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Order     int    `json:"order"`
	Published bool   `json:"published"`
}
