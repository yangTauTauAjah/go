package service

import (
	"context"
	"errors"
	"strings"

	"students_api/app/model"
	"students_api/app/repository"
	"students_api/helper"

	"github.com/gofiber/fiber/v2"
)

// PortfolioService memakai DUA database sekaligus, persis seperti
// ReviewService: identitas mahasiswa dibaca dari PostgreSQL, sedangkan
// portofolionya disimpan di MongoDB.
//
// Yang menjadi tanggung jawab service: pemeriksaan "mahasiswa ini benar-benar
// ada". MongoDB tidak mengenal tabel students, jadi validasi lintas database ini
// tidak bisa dijaga index. Service adalah satu-satunya tempat yang bisa
// menjawabnya, karena hanya service yang memegang kedua repository.
type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
}

func NewPortfolioService(
	portfolios repository.PortfolioRepository,
	students repository.StudentRepository,
) *PortfolioService {
	return &PortfolioService{portfolios: portfolios, students: students}
}

// ---------- GET /students/:id/portfolio ----------
//
// Belum ada portofolio itu kondisi normal, bukan error: mahasiswa yang belum
// pernah mengisinya harus mendapat 200 dengan nilai kosong supaya UI bisa
// merender tombol "isi portofolio", bukan halaman error.
func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	if err := s.ensureStudentExists(ctx, studentID); err != nil {
		return err
	}

	portfolio, err := s.portfolios.FindByStudent(ctx, studentID)
	if err == nil {
		return helper.Success(c, fiber.StatusOK,
			"portofolio berhasil diambil", portfolio)
	}
	if !errors.Is(err, repository.ErrPortfolioNotFound) {
		return translatePortfolioError(err)
	}

	// Belum pernah diisi: kembalikan bentuk yang sama supaya kontrak API
	// tidak berubah hanya karena datanya belum ada.
	empty := model.Portfolio{
		StudentID: studentID, Projects: []model.Project{},
		Certificates: []model.Certificate{},
	}
	return helper.Success(c, fiber.StatusOK,
		"portofolio belum diisi", empty)
}

// ---------- PUT /students/:id/portfolio ----------
//
// Melempar 404 bila mahasiswa-nya tidak ada di PostgreSQL — inilah penjaga
// yang diminta soal penolakan mahasiswa tak dikenal.
func (s *PortfolioService) Upsert(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	if err := s.ensureStudentExists(ctx, studentID); err != nil {
		return err
	}

	var req model.UpsertPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Headline = trimSpace(req.Headline)
	req.Bio = trimSpace(req.Bio)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	saved, err := s.portfolios.Upsert(ctx, studentID, model.Portfolio{
		Headline: req.Headline, Bio: req.Bio,
		Projects:     prepareProjects(req.Projects),
		Certificates: prepareCertificates(req.Certificates),
	})
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK,
		"portofolio berhasil disimpan", saved)
}

// ---------- POST /students/:id/portfolio/projects ----------
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	if err := s.ensureStudentExists(ctx, studentID); err != nil {
		return err
	}

	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Title = trimSpace(req.Title)
	req.URL = trimSpace(req.URL)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.portfolios.AddProject(ctx, studentID, model.Project{
		Title: req.Title, Technologies: req.Technologies,
		URL: req.URL, Year: req.Year,
	})
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK,
		"proyek berhasil ditambahkan", updated)
}

// ---------- DELETE /students/:id/portfolio/projects/:projectId ----------
func (s *PortfolioService) RemoveProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	projectID := trimSpace(c.Params("projectId"))
	if projectID == "" {
		return helper.BadRequest("id proyek wajib diisi")
	}
	if err := s.ensureStudentExists(ctx, studentID); err != nil {
		return err
	}

	updated, err := s.portfolios.RemoveProject(ctx, studentID, projectID)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK,
		"proyek berhasil dihapus", updated)
}

// ---------- GET /portfolios/stats/technologies ----------
//
// Endpoint lintas mahasiswa, jadi tidak punya :id dan tidak butuh pemeriksaan
// mahasiswa per DOCUMENT. Answer-nya hanya bentuk JSON success biasa.
func (s *PortfolioService) TechStats(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	limit := c.QueryInt("limit", repository.DefaultTechStatLimit)
	if limit < 1 || limit > 100 {
		limit = repository.DefaultTechStatLimit
	}
	stats, err := s.portfolios.TopTechnologies(ctx, limit)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK,
		"statistik teknologi berhasil diambil", fiber.Map{
			"technologies": stats,
			"limit":        limit,
		})
}

// ---------- private ----------

// ensureStudentExists menjawab "apakah mahasiswa ini ada?" lewat PostgreSQL.
//
// Sengaja memakai FindByID dan bukan Exists(): interface StudentRepository
// sudah punya FindByID, menambah Exists() ke interface berarti mengubah
// semua implementasi sekaligus, termasuk yang dipakai modul lain.
func (s *PortfolioService) ensureStudentExists(
	ctx context.Context, studentID int,
) error {
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}
	return nil
}

func prepareProjects(in []model.Project) []model.Project {
	out := make([]model.Project, 0, len(in))
	for _, p := range in {
		p.Title = trimSpace(p.Title)
		p.URL = trimSpace(p.URL)
		out = append(out, p)
	}
	return out
}

func prepareCertificates(in []model.Certificate) []model.Certificate {
	out := make([]model.Certificate, 0, len(in))
	for _, c := range in {
		c.Name = trimSpace(c.Name)
		c.Issuer = trimSpace(c.Issuer)
		out = append(out, c)
	}
	return out
}

func trimSpace(s string) string { return strings.TrimSpace(s) }

// translatePortfolioError memetakan error repository ke error HTTP. Pola ini
// sama persis dengan translateReviewError.
func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id tidak valid")
	case errors.Is(err, repository.ErrPortfolioNotFound):
		return helper.NotFound("portofolio tidak ditemukan")
	case errors.Is(err, repository.ErrProjectNotFound):
		return helper.NotFound("proyek tidak ditemukan")
	case errors.Is(err, repository.ErrProjectLimit):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT",
			"jumlah proyek sudah mencapai batas 20, hapus proyek lama terlebih dahulu")
	case errors.Is(err, repository.ErrDuplicatePortfolio):
		return helper.NewError(fiber.StatusConflict, "PORTFOLIO_EXISTS",
			"portofolio sudah ada, gunakan PUT untuk menggantinya")
	default:
		return helper.Internal(err)
	}
}
