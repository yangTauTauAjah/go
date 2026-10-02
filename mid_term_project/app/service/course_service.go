package service

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"siakad/app/repository"
	"siakad/helper"
)

type CourseService struct {
	repo repository.CourseRepository
}

func NewCourseService(repo repository.CourseRepository) *CourseService {
	return &CourseService{repo: repo}
}

func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	semester, _ := strconv.Atoi(c.Query("semester", "0"))
	search := c.Query("search", "")
	available := c.Query("available") == "true"
	rows, err := s.repo.FindAll(ctx, semester, search, available)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK,
		"Daftar mata kuliah berhasil diambil", rows)
}
