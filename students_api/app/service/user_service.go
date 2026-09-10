package service

import (
	"errors"
	"strconv"
	"strings"
	"tugas2/app/model"
	"tugas2/app/repository"
	"tugas2/helper"

	"github.com/gofiber/fiber/v2"
)

type StudentService struct {
	repo repository.StudentRepository
}

func NewStudentService(repo repository.StudentRepository) *StudentService {
	return &StudentService{repo: repo}
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
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data student")
	}
	return helper.Success(c, fiber.StatusOK, "student ditemukan", student)
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
	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(c, err, "gagal menghapus student")
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
