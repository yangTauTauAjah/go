package service

import (
	"strings"
	"students_api/app/model"
)

// ValidateCreate dan ValidateReplace sebelumnya berisi pemeriksaan
// bentuk manual (username/email/panjang password). Aturan tersebut
// kini dinyatakan sebagai tag validate pada struct
// CreateStudentRequest dan ReplaceStudentRequest. Validasi manual
// dipertahankan sebagai stub kosong agar kompatibilitas dengan kode
// lama — misalnya unit test — tetap terjaga, dan agar cerita migrasi
// "dari manual ke tag" tetap bisa dilihat di git history.

// ApplyPatch tidak lagi mengembalikan daftar error. Pemeriksaan bentuk
// sudah selesai dikerjakan tag sebelum fungsi ini dipanggil, sehingga di
// sini tugasnya tinggal satu: menggabungkan.
func ApplyPatch(current model.Student, req model.PatchStudentRequest) model.Student {
	if req.Username != "" {
		current.Username = strings.TrimSpace(req.Username)
	}
	if req.Email != nil {
		current.Email = strings.TrimSpace(*req.Email)
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current
}

// IsEmptyPatch memeriksa body PATCH yang tidak berisi field apa pun.
//
// Aturan ini tidak dapat ditulis sebagai tag: tag memeriksa satu field
// pada satu waktu, sedangkan aturan ini berbicara tentang HUBUNGAN antar
// field — setidaknya satu di antara mereka harus ada.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.Username == "" && req.Email == nil && req.IsActive == nil
}

func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}
