package model

import "time"

type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	// Meta sengaja bertipe any: pagination offset menggunakan *Meta,
	// pagination cursor menggunakan *CursorMeta. Pemilihannya menjadi
	// urusan service per endpoint, bukan kebijakan terpusat.
	Meta   any `json:"meta,omitempty"`
	Errors any `json:"errors,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
}

// Cursor adalah penanda posisi pada keyset pagination.
// Dibaca oleh helper.DecodeCursor dan dibuat oleh repository
// dari baris terakhir halaman yang dikirim.
type Cursor struct {
	CreatedAt time.Time
	ID        int
}

// CursorQuery adalah parameter untuk FindAfterCursor.
// After == nil artinya "halaman pertama".
type CursorQuery struct {
	After    *Cursor
	Limit    int
	Search   string
	IsActive *bool
}
