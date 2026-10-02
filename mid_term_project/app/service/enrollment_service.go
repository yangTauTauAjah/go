package service

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"siakad/app/model"
	"siakad/app/repository"
	"siakad/helper"
)

type EnrollmentService struct {
	students    repository.StudentRepository
	enrollments repository.EnrollmentRepository
}

func NewEnrollmentService(students repository.StudentRepository, en repository.EnrollmentRepository) *EnrollmentService {
	return &EnrollmentService{students: students, enrollments: en}
}

// Enroll: business rule (validasi IPK & SKS) + repository atomic action.
func (s *EnrollmentService) Enroll(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	student, err := s.students.FindByUserID(ctx, current.UserID)
	if err != nil {
		return helper.NotFound("Mahasiswa tidak ditemukan")
	}

	// Batas SKS dihitung sebelum enroll agar kita bisa menolak early.
	batasSKS := BatasSKSByIPK(student.IPK)
	totalSKS, err := s.enrollments.SumSKSByStudentAndTahun(ctx, student.ID, req.TahunAkademik)
	if err != nil {
		return helper.Internal(err)
	}

	enrolled, err := s.enrollments.EnrollAtomic(ctx, student.ID, req.CourseID, req.TahunAkademik)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicate):
			return helper.Conflict("Mata kuliah sudah pernah diambil pada tahun akademik ini")
		case errors.Is(err, repository.ErrNotFound):
			return helper.NotFound("Mata kuliah tidak ditemukan")
		case errors.Is(err, repository.ErrKuotaPenuh):
			return helper.Unprocessable("Kuota mata kuliah sudah terpenuhi")
		default:
			return helper.Internal(err)
		}
	}

	// Validasi batas SKS setelah enroll berhasil.
	if totalSKS+enrolled.SKS > batasSKS {
		// Rollback manual: hapus enrollment yang baru saja kita buat.
		if _, derr := s.enrollments.DeleteByStudentAndID(ctx, enrolled.ID, student.ID); derr != nil {
			// Jika rollback gagal, log dan tetap kembalikan di SKS ke client.
		}
		sisa := batasSKS - totalSKS
		if sisa < 0 {
			sisa = 0
		}
		return helper.Unprocessable(fmt.Sprintf(
			"Total SKS akan melebihi batas (%d SKS). Sisa kuota SKS Anda: %d",
			batasSKS, sisa))
	}

	return helper.Created(c, "Mata kuliah berhasil diambil", enrolled,
		fmt.Sprintf("/api/v1/enrollments/%d", enrolled.ID))
}

func (s *EnrollmentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	student, err := s.students.FindByUserID(ctx, current.UserID)
	if err != nil {
		return helper.NotFound("Mahasiswa tidak ditemukan")
	}

	// Pastikan enrollment milik mahasiswa yang sedang login.
	enrolled, err := s.enrollments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("Enrollment tidak ditemukan")
		}
		return helper.Internal(err)
	}
	if enrolled.StudentID != student.ID {
		return helper.Forbidden("Enrollment ini bukan milik Anda")
	}

	if _, err := s.enrollments.DeleteByStudentAndID(ctx, id, student.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("Enrollment tidak ditemukan")
		}
		return helper.Internal(err)
	}
	return helper.NoContent(c)
}
