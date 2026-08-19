package main

import "fmt"

// 1. Value Receiver: Hanya membaca data tanpa mengubah state struct
func (s Student) GetInfo() string {
	status := "Inactive"
	if s.IsActive {
		status = "Active"
	}
	return fmt.Sprintf("ID: %s | Name: %s | Grade: %.2f | Status: %s", s.ID, s.Name, s.Grade, status)
}

// 2. Pointer Receiver: Perlu memodifikasi nilai field Grade pada struct asli
func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

// 3. Pointer Receiver: Perlu memodifikasi nilai field IsActive menjadi true
func (s *Student) Activate() {
	s.IsActive = true
}

// 4. Pointer Receiver: Perlu memodifikasi nilai field IsActive menjadi false
func (s *Student) Deactivate() {
	s.IsActive = false
}

func Struct() {

	// test swap
	fmt.Printf("=== Test Modify Object ===\n\n")

	// Inisialisasi data student
	student := Student{
		ID:       "STD-001",
		Name:     "Alex Johnson",
		Grade:    82.5,
		IsActive: false,
	}

	fmt.Printf("[before] %s\n", student.GetInfo())

	student.Activate()
	student.UpdateGrade(91.0)
	student.Deactivate()

	fmt.Printf("[after] %s\n", student.GetInfo())
}
