package service

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"siakad/app/model"
	"siakad/app/repository"
	"siakad/helper"
)

type StudentService struct {
	repo        repository.StudentRepository
	enrollments repository.EnrollmentRepository
}

func NewStudentService(repo repository.StudentRepository, en repository.EnrollmentRepository) *StudentService {
	return &StudentService{repo: repo, enrollments: en}
}

// BatasSKSByIPK mengembalikan batas SKS sesuai aturan IPK.
func BatasSKSByIPK(ipk float64) int {
	switch {
	case ipk >= 3.00:
		return 24
	case ipk >= 2.50:
		return 21
	default:
		return 18
	}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)
	rows, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}
	meta := &model.Meta{
		CurrentPage: q.Page,
		PerPage:     q.Limit,
		Total:       total,
		LastPage:    (total + q.Limit - 1) / q.Limit,
	}
	return helper.SuccessList(c, "Data mahasiswa berhasil diambil", rows, meta)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Nama = strings.TrimSpace(req.Nama)
	req.Email = strings.TrimSpace(req.Email)
	req.Prodi = strings.TrimSpace(req.Prodi)

	if thisYear := time.Now().Year(); req.Angkatan > thisYear {
		return helper.Unprocessable("angkatan tidak boleh melebihi tahun berjalan")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}
	if !allDigits(req.NIM) || len(req.NIM) != 12 {
		return helper.FailValidation(c, map[string]string{
			"nim": "NIM harus 12 digit angka",
		})
	}

	// Password awal = NIM (di-hash).
	hashed, err := helper.HashPassword(req.NIM)
	if err != nil {
		return helper.Internal(err)
	}
	user := model.User{Email: req.Email, Password: hashed, Role: "mahasiswa", IsActive: true}
	student := model.Student{
		NIM: req.NIM, Nama: req.Nama, Prodi: req.Prodi,
		Angkatan: req.Angkatan, IPK: req.IPKTerakhir,
	}
	created, err := s.repo.CreateWithUser(ctx, user, student)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Conflict("NIM atau email sudah terdaftar")
		}
		return helper.Internal(err)
	}
	return helper.Created(c, "Mahasiswa berhasil ditambahkan", created,
		"/api/v1/students/"+strconv.Itoa(created.ID))
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	// Ambil student target (hanya yang aktif).
	target, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("Mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// Mahasiswa hanya boleh akses data miliknya sendiri.
	if current.Role == "mahasiswa" {
		own, err := s.repo.FindByUserID(ctx, current.UserID)
		if err != nil || own.ID != target.ID {
			return helper.Forbidden("Tidak berhak mengakses data mahasiswa lain")
		}
	}

	enrollments, err := s.enrollments.FindByStudent(ctx, target.ID, "")
	if err != nil {
		return helper.Internal(err)
	}
	totalSKS := 0
	for _, e := range enrollments {
		totalSKS += e.SKS
	}
	detail := model.StudentDetail{
		Student:     target,
		Enrollments: enrollments,
		TotalSKS:    totalSKS,
		BatasSKS:    BatasSKSByIPK(target.IPK),
	}
	return helper.Success(c, fiber.StatusOK, "Detail mahasiswa berhasil diambil", detail)
}

func (s *StudentService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	req.Nama = strings.TrimSpace(req.Nama)
	req.Prodi = strings.TrimSpace(req.Prodi)
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	target, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("Mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}
	target.Nama = req.Nama
	target.Prodi = req.Prodi
	target.Angkatan = req.Angkatan
	target.IPK = req.IPKTerakhir
	updated, err := s.repo.UpdateAdmin(ctx, target)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "Mahasiswa berhasil diperbarui", updated)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("Mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}
	return helper.NoContent(c)
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
