package model

import "time"

type Student struct {
	ID           int           `json:"id"`
	Username     string        `json:"username"`
	Email        string        `json:"email"`
	Password     string        `json:"-"`
	Role         string        `json:"role"`
	Name         string        `json:"name"`
	Field        string        `json:"field"`
	Semester     int           `json:"semester"`
	Grade        float64       `json:"grade"`
	IsActive     bool          `json:"is_active"`
	Course       []string      `json:"course"`
	Address      string        `json:"address"`
	CreatedAt    time.Time     `json:"created_at"`
	Achievements []Achievement `json:"achievements,omitempty"`
}

type CreateStudentRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ReplaceStudentRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
}

type PatchStudentRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// Offset menghitung berapa baris yang dilewati untuk halaman ini.
// Perhitungan ini pindah ke sini karena kini dipakai langsung oleh SQL.
func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

// AssignRoleRequest dipakai endpoint PATCH /users/:id/role.
type AssignRoleRequest struct {
	Role string `json:"role"`
}
