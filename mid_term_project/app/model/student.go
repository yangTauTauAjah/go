package model

import "time"

type Student struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	NIM       string     `json:"nim"`
	Nama      string     `json:"nama"`
	Prodi     string     `json:"prodi"`
	Angkatan  int        `json:"angkatan"`
	IPK       float64    `json:"ipk_terakhir"`
	DeletedAt *time.Time `json:"-"`
	Email     string     `json:"email,omitempty"`
	Role      string     `json:"role,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
}

type CreateStudentRequest struct {
	NIM         string  `json:"nim"         validate:"required,len=12"`
	Nama        string  `json:"nama"        validate:"required,min=2,max=120"`
	Email       string  `json:"email"       validate:"required,email,max=120"`
	Prodi       string  `json:"prodi"       validate:"required,min=2,max=100"`
	Angkatan    int     `json:"angkatan"    validate:"required,gte=1900"`
	IPKTerakhir float64 `json:"ipk_terakhir" validate:"omitempty,ipkrange"`
}

type UpdateStudentRequest struct {
	Nama        string  `json:"nama"         validate:"required,min=2,max=120"`
	Prodi       string  `json:"prodi"        validate:"required,min=2,max=100"`
	Angkatan    int     `json:"angkatan"     validate:"required,gte=1900"`
	IPKTerakhir float64 `json:"ipk_terakhir" validate:"omitempty,ipkrange"`
}

type Course struct {
	ID        int    `json:"id"`
	KodeMK    string `json:"kode_mk"`
	NamaMK    string `json:"nama_mk"`
	SKS       int    `json:"sks"`
	Semester  int    `json:"semester"`
	Kuota     int    `json:"kuota"`
	Terisi    int    `json:"terisi"`
	SisaKuota int    `json:"sisa_kuota"`
}

type Enrollment struct {
	ID            int       `json:"id"`
	StudentID     int       `json:"student_id"`
	CourseID      int       `json:"course_id"`
	TahunAkademik string    `json:"tahun_akademik"`
	CreatedAt     time.Time `json:"created_at"`
	// Denormalized untuk response detail
	KodeMK string `json:"kode_mk,omitempty"`
	NamaMK string `json:"nama_mk,omitempty"`
	SKS    int    `json:"sks,omitempty"`
}

type CreateEnrollmentRequest struct {
	CourseID      int    `json:"course_id"       validate:"required,gte=1"`
	TahunAkademik string `json:"tahun_akademik"  validate:"required,tahunakademik"`
}

type StudentDetail struct {
	Student
	Enrollments []Enrollment `json:"enrollments"`
	TotalSKS    int          `json:"total_sks"`
	BatasSKS    int          `json:"batas_sks"`
}
