package service

import (
	"errors"
	"strconv"
	"strings"
	"students_api/app/model"
	"students_api/app/repository"
	"students_api/helper"

	"github.com/gofiber/fiber/v2"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, permissions *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: permissions}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	q := helper.ParseListQuery(c)
	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError,
			"gagal mengambil data student")
	}
	return helper.SuccessList(c, "daftar student berhasil diambil", students, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: CountTotalPages(total, q.Limit),
	})
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	// Pemeriksaan hak akses dilakukan SEBELUM data diambil.
	// Bila dibalik, penyerang tetap dapat menyimpulkan keberadaan sebuah id
	// dari perbedaan waktu tanggap antara 403 dan 404.
	if !CanAccessStudent(current, id, s.perms, "user:read:any") {
		return helper.Fail(c, fiber.StatusForbidden,
			"tidak berhak mengakses data user lain")
	}
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data user")
	}
	return helper.Success(c, fiber.StatusOK, "user ditemukan", user)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest,
			"body harus berupa JSON yang valid")
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}
	newUser, err := s.repo.Create(ctx, model.Student{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IsActive: true,
	})
	if err != nil {
		return translateError(c, err, err.Error())
	}
	return helper.Created(c, "student berhasil dibuat", newUser, "/api/v1/students/"+strconv.Itoa(newUser.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest,
			"body harus berupa JSON yang valid")
	}
	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}
	result, err := s.repo.Update(ctx, model.Student{
		ID:       id,
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		IsActive: req.IsActive,
	})
	if err != nil {
		return translateError(c, err, "gagal memperbarui student")
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diganti seluruhnya", result)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest,
			"body harus berupa JSON yang valid")
	}
	if IsEmptyPatch(req) {
		return helper.Fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}
	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data student")
	}
	updated, errs := ApplyPatch(current, req)
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}
	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(c, err, "gagal memperbarui student")
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diperbarui sebagian", result)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data student")
	}
	if current.ID == id {
		return helper.Fail(c, fiber.StatusForbidden,
			"tidak boleh menghapus akun sendiri")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(c, err, "gagal menghapus user")
	}
	return helper.NoContent(c)
}

func translateError(c *fiber.Ctx, err error, generalMessage string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Fail(c, fiber.StatusConflict, "username sudah dipakai")
	default:
		return helper.Fail(c, fiber.StatusInternalServerError, generalMessage)
	}
}

// ---------- PATCH /users/:id/role ----------
// Dijaga middleware dengan permission role:assign.
func (s *StudentService) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if errs := ValidateAssignRole(current, id, req, s.perms); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}
	result, err := s.repo.UpdateRole(ctx, id, strings.TrimSpace(req.Role))
	if err != nil {
		return translateError(c, err, "gagal mengubah role user")
	}
	return helper.Success(c, fiber.StatusOK, "role user berhasil diubah", result)
}
