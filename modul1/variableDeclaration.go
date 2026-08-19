package main

import "fmt"

func VariableDeclaration() {
	var stringExample string = "Hello, World!"
	var intExample int = 42
	var floatExample float64 = 3.14
	var boolExample bool = true
	var sliceExample []int = []int{1, 2, 3, 4, 5}

	fmt.Printf("=== Initialize Variables ===\n\n")
	fmt.Println("String: ", stringExample)
	fmt.Println("Int: ", intExample)
	fmt.Println("Float: ", floatExample)
	fmt.Println("Bool: ", boolExample)
	fmt.Println("Slice: ", sliceExample)

	fmt.Printf("\n")

	students := make(map[string]Student)

	// add
	fmt.Printf("=== Adding Data ===\n\n")
	students["101"] = Student{
		ID:       "101",
		Name:     "Budi Santoso",
		Field:    "Computer Science",
		Semester: 4,
		Grade:    3.85,
		IsActive: true,
		Course:   []string{"Algorithms", "Database Systems"},
		Address:  "Jakarta",
	}

	students["102"] = Student{
		ID:       "102",
		Name:     "Siti Aminah",
		Field:    "Information Systems",
		Semester: 2,
		Grade:    3.92,
		IsActive: true,
		Course:   []string{"UI/UX Design", "Web Development"},
		Address:  "Bandung",
	}

	students["103"] = Student{
		ID:       "103",
		Name:     "Andi Wijaya",
		Field:    "Software Engineering",
		Semester: 6,
		Grade:    3.50,
		IsActive: false,
		Course:   []string{"Software Architecture"},
		Address:  "Surabaya",
	}

	fmt.Printf("Added %d students.\n\n", len(students))

	// check existing
	fmt.Printf("=== Checking Existance ===\n\n")
	checkID := "101"
	if student, exists := students[checkID]; exists {
		fmt.Printf("Found (ID %s):\n- Name: %s\n- Field: %s\n- Grade: %.2f\n- Courses: %v\n",
			checkID, student.Name, student.Field, student.Grade, student.Course)
	} else {
		fmt.Printf("Data ID %s not found.\n", checkID)
	}

	// check non-existing
	unknownID := "999"
	if student, exists := students[unknownID]; exists {
		fmt.Printf("Found: %s\n\n", student.Name)
	} else {
		fmt.Printf("Data ID %s not found.\n\n", unknownID)
	}

	// remove data
	fmt.Printf("=== Removing Data ===\n\n")
	deleteID := "103"
	fmt.Printf("Remove ID %s (%s)...\n", deleteID, students[deleteID].Name)
	delete(students, deleteID)
	fmt.Println()

	// iterate all datas
	fmt.Printf("=== Listing Data ===\n\n")
	for ID, s := range students {
		fmt.Printf("[%s] %-15s | %-20s | Sem: %d | Grade: %.2f | IsActive: %-5t | Address: %s\n",
			ID, s.Name, s.Field, s.Semester, s.Grade, s.IsActive, s.Address)
	}

	fmt.Printf("\n")
}
