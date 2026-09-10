package main

import (
	"errors"
	"strconv"
	"strings"
	"tugas2/app/model"
	"tugas2/app/repository"

	"github.com/gofiber/fiber/v2"
)

var students []model.Student
var nextID = 1

func findStudentIndex(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

func cocokPencarian(u model.Student, kata string) bool {
	kata = strings.ToLower(kata)
	return strings.Contains(strings.ToLower(u.Username), kata) ||
		strings.Contains(strings.ToLower(u.Email), kata)
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

func terjemahkanError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return fail(c, fiber.StatusConflict, "username sudah dipakai")
	default:
		return fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()
	q := parseListQuery(c)
	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil data student")
	}
	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}
	return okList(c, "daftar student berhasil diambil", students, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}
func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	student, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data student")
	}
	return ok(c, "student ditemukan", student)
}
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()
	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	errs := map[string]string{}
	if req.Username == "" {
		errs["username"] = "wajib diisi"
	}
	if !strings.Contains(req.Email, "@") {
		errs["email"] = "format email tidak valid"
	}
	if len(req.Password) < 8 {
		errs["password"] = "minimal 8 karakter"
	}
	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	baru, err := h.repo.Create(ctx, model.Student{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IsActive: true,
	})
	if err != nil {
		return terjemahkanError(c, err, "gagal menyimpan student baru")
	}
	return created(c, "student berhasil dibuat", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	errs := map[string]string{}
	if strings.TrimSpace(req.Username) == "" {
		errs["username"] = "wajib diisi pada PUT"
	}
	if !strings.Contains(req.Email, "@") {
		errs["email"] = "wajib diisi dan berformat email pada PUT"
	}
	if len(errs) > 0 {
		return failValidation(c, errs)
	}
	hasil, err := h.repo.Update(ctx, model.Student{
		ID: id, Username: req.Username, Email: req.Email, IsActive: req.IsActive,
	})
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui student")
	}
	return ok(c, "student berhasil diganti seluruhnya", hasil)
}
func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if req.Username == nil && req.Email == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}
	saatIni, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil data user")
	}
	if req.Username != nil {
		if strings.TrimSpace(*req.Username) == "" {
			return failValidation(c, map[string]string{"username": "tidak boleh kosong"})
		}
		saatIni.Username = *req.Username
	}
	if req.Email != nil {
		if !strings.Contains(*req.Email, "@") {
			return failValidation(c, map[string]string{"email": "format email tidak valid"})
		}
		saatIni.Email = *req.Email
	}
	if req.IsActive != nil {
		saatIni.IsActive = *req.IsActive
	}
	hasil, err := h.repo.Update(ctx, saatIni)
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui user")
	}
	return ok(c, "student berhasil diperbarui sebagian", hasil)
}
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	if err := h.repo.Delete(ctx, id); err != nil {
		return terjemahkanError(c, err, "gagal menghapus student")
	}
	return noContent(c)
}
