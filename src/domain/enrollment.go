package domain

import "time"

type EnrollmentStatus string

const (
	EnrollmentPending  EnrollmentStatus = "pending"
	EnrollmentApproved EnrollmentStatus = "approved"
	EnrollmentDenied   EnrollmentStatus = "denied"
)

type CourseEnrollment struct {
	StudentID         string           `json:"student_id"`
	StudentName       string           `json:"student_name"`
	StudentEmail      string           `json:"student_email"`
	CourseID          string           `json:"course_id"`
	CourseTitle       string           `json:"course_title"`
	Subject           string           `json:"subject"`
	Description       string           `json:"description"`
	TeacherID         string           `json:"teacher_id"`
	TeacherName       string           `json:"teacher_name"`
	TeacherEmail      string           `json:"teacher_email"`
	TeacherTitle      string           `json:"teacher_title,omitempty"`
	TeacherDepartment string           `json:"teacher_department,omitempty"`
	Status            EnrollmentStatus `json:"status"`
	RequestedAt       time.Time        `json:"requested_at"`
	ReviewedAt        *time.Time       `json:"reviewed_at,omitempty"`
}
