package domain

import "testing"

func TestStudentProfileRequiresStudentContext(t *testing.T) {
	user, err := NewUser("student-1", "student@example.com", "A Student", RoleStudent)
	if err != nil {
		t.Fatal(err)
	}

	profile := StudentProfile{
		User:            user,
		AcademicProgram: "Computer Science",
		Branch:          "Systems",
		Semester:        3,
		AcademicYear:    "2026",
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("valid student profile rejected: %v", err)
	}
	profile.User.Role = RoleTeacher
	if err := profile.Validate(); err == nil {
		t.Fatal("student profile with teacher role was accepted")
	}
}

func TestTeacherProfileRequiresRoleSpecificDetails(t *testing.T) {
	user, err := NewUser("teacher-1", "teacher@example.com", "A Teacher", RoleTeacher)
	if err != nil {
		t.Fatal(err)
	}
	profile := TeacherProfile{User: user, Department: "Mathematics", TeachingTitle: "Professor"}
	if err := profile.Validate(); err != nil {
		t.Fatalf("valid teacher profile rejected: %v", err)
	}
	profile.TeachingTitle = ""
	if err := profile.Validate(); err == nil {
		t.Fatal("teacher profile without title was accepted")
	}
}
