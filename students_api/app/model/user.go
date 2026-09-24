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
	Role     string `json:"role"`
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
}

// Pada PATCH, pointer membedakan "tidak dikirim" (nil) dari "dikirim
// bernilai kosong". omitnil dipilih karena ia menyatakan maksud yang
// sebenarnya: lewati hanya bila nil.
type PatchStudentRequest struct {
	Username string  `json:"username,omitempty" validate:"omitnil,min=3,max=30,alphanum"`
	Email    *string `json:"email,omitempty" validate:"omitnil,email,max=120"`
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

type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// Mulai pertemuan ini, aturan validasi ditulis sebagai tag pada struct.
// Aturan dan bentuk data berada pada baris yang sama, sehingga menambah
// satu field tanpa aturannya menjadi kelalaian yang langsung terlihat.
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email    string `json:"email" validate:"required,email,max=120"`
	Password string `json:"password" validate:"required,min=8,max=72,nospace"`
}
type ReplaceUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email    string `json:"email" validate:"required,email,max=120"`
	IsActive bool   `json:"is_active"`
}

// CursorMeta menggantikan Meta pada endpoint yang memakai cursor.
//
// Perhatikan tidak adanya Total dan TotalPages. Keduanya tidak dapat
// disediakan tanpa COUNT(*) atas seluruh tabel — persis biaya yang ingin
// dihindari oleh pagination berbasis cursor.
type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
