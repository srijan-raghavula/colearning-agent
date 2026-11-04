package domain

import (
	"errors"
	"strings"
)

type UserRole = string

const (
	RoleStudent UserRole = "student"
	RoleTeacher UserRole = "teacher"
)

// User is the shared identity for every CoLearn account. Role-specific
// profiles embed it so student and teacher clients can rely on one base shape.
type User struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        UserRole `json:"role"`
	AuthVersion int      `json:"auth_version"`
}

func NewUser(id, email, name string, role UserRole) (User, error) {
	user := User{ID: strings.TrimSpace(id), Email: strings.ToLower(strings.TrimSpace(email)), Name: strings.TrimSpace(name), Role: role, AuthVersion: 1}
	if err := user.Validate(); err != nil {
		return User{}, err
	}
	return user, nil
}

func (u User) Validate() error {
	if u.ID == "" || u.Email == "" || u.Name == "" {
		return errors.New("user identity requires id, email and name")
	}
	if u.Role != RoleStudent && u.Role != RoleTeacher {
		return errors.New("user role must be student or teacher")
	}
	return nil
}

type StudentProfile struct {
	User
	AcademicProgram string `json:"academic_program"`
	Branch          string `json:"branch"`
	Semester        int    `json:"semester"`
	AcademicYear    string `json:"academic_year"`
}

func (p StudentProfile) Validate() error {
	if err := p.User.Validate(); err != nil {
		return err
	}
	if p.User.Role != RoleStudent {
		return errors.New("student profile requires the student role")
	}
	if strings.TrimSpace(p.AcademicProgram) == "" || strings.TrimSpace(p.Branch) == "" || p.Semester < 1 || strings.TrimSpace(p.AcademicYear) == "" {
		return errors.New("student profile requires academic program, branch, semester and academic year")
	}
	return nil
}

type TeacherProfile struct {
	User
	Department    string `json:"department"`
	TeachingTitle string `json:"teaching_title"`
}

func (p TeacherProfile) Validate() error {
	if err := p.User.Validate(); err != nil {
		return err
	}
	if p.User.Role != RoleTeacher {
		return errors.New("teacher profile requires the teacher role")
	}
	if strings.TrimSpace(p.Department) == "" || strings.TrimSpace(p.TeachingTitle) == "" {
		return errors.New("teacher profile requires department and teaching title")
	}
	return nil
}
