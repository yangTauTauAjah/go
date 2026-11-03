package model

import "time"

// Portfolio adalah dokumen portofolio seorang mahasiswa yang disimpan di
// MongoDB.
//
// Perhatikan: tidak ada satu pun tipe dari driver MongoDB di sini. ID disimpan
// sebagai string, bukan bson.ObjectID, dan technologies sebagai []string biasa.
// Package model tetap bebas dari apa pun yang bukan Go murni — aturan yang
// sama sejak Modul 4. Menerjemahkan ke bentuk BSON adalah tugas repository.
type Portfolio struct {
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Project disimpan DI DALAM dokumen Portfolio (embedding), bukan di collection
// terpisah. Proyek tidak pernah dibaca tanpa portofolio induknya, jadi
// menyimpan terpisah hanya menambah satu join yang tidak perlu.
//
// Technologies sengaja []string, bukan []bson.ObjectID: dari sudut pandang
// client, technologies adalah label bebas, bukan referensi ke dokumen lain.
type Project struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Technologies []string  `json:"technologies"`
	URL          string    `json:"url"`
	Year         int       `json:"year"`
	CreatedAt    time.Time `json:"created_at"`
}

// Certificate juga di-embed. Tidak ada endpoint terpisah untuk menambah
// sertifikat pada tugas ini, sehingga field ini hanya dibaca lewat Portfolio
// dan hanya ditulis lewat PUT (penggantian penuh). Bentuknya tetap modelled
// penuh agar portofolio utuh saat dibaca.
type Certificate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Issuer    string    `json:"issuer"`
	IssuedAt  time.Time `json:"issued_at"`
	CreatedAt time.Time `json:"created_at"`
}

// UpsertPortfolioRequest adalah body untuk PUT /students/:id/portfolio.
// PUT berarti penggantian penuh: field yang tidak dikirim akan hilang, bukan
// dipertahankan.
type UpsertPortfolioRequest struct {
	Headline     string        `json:"headline" validate:"required,min=3,max=120"`
	Bio          string        `json:"bio" validate:"required,min=10,max=1000"`
	Projects     []Project     `json:"projects" validate:"max=20,dive"`
	Certificates []Certificate `json:"certificates" validate:"max=20,dive"`
}

// AddProjectRequest adalah body untuk POST /students/:id/portfolio/projects.
type AddProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=3,max=120"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=10,dive,required,min=1,max=40"`
	URL          string   `json:"url" validate:"omitempty,url,max=300"`
	Year         int      `json:"year" validate:"required,min=1990,max=2100"`
}

// TechnologyStat adalah satu baris hasil agregasi GET
// /portfolios/stats/technologies.
type TechnologyStat struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}
