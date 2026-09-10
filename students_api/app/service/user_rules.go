package service

import (
	"strings"
	"tugas2/app/model"
)

func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.Username) == "" {
		errs["username"] = "wajib diisi"
	}
	if !isValidEmail(req.Email) {
		errs["email"] = "format email tidak valid"
	}
	if len(req.Password) < 8 {
		errs["password"] = "minimal 8 karakter"
	}
	return errs
}

func ValidateReplace(req model.ReplaceStudentRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.Username) == "" {
		errs["username"] = "wajib diisi pada PUT"
	}
	if !isValidEmail(req.Email) {
		errs["email"] = "wajib diisi dan berformat email pada PUT"
	}
	return errs
}

func ApplyPatch(
	current model.Student, req model.PatchStudentRequest,
) (model.Student, map[string]string) {
	errs := map[string]string{}
	if req.Username != nil {
		if strings.TrimSpace(*req.Username) == "" {
			errs["username"] = "tidak boleh kosong"
		} else {
			current.Username = *req.Username
		}
	}
	if req.Email != nil {
		if !isValidEmail(*req.Email) {
			errs["email"] = "format email tidak valid"
		} else {
			current.Email = *req.Email
		}
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current, errs
}

func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.Username == nil && req.Email == nil && req.IsActive == nil
}

func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	at := strings.Index(email, "@")
	dot := strings.LastIndex(email, ".")
	return at > 0 && dot > at+1 && dot < len(email)-1
}
