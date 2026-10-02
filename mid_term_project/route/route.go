package route

import (
	"context"
	"time"

	"siakad/app/service"
	"siakad/helper"
	"siakad/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")
	api.Get("/health", healthCheck(deps.Pool))

	auth := api.Group("/auth")
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// Endpoint 3-7: student management
	students := api.Group("/students",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))

	students.Get("/", middleware.RequireRole("admin"), deps.StudentService.List)
	students.Post("/", middleware.RequireRole("admin"), deps.StudentService.Create)
	students.Get("/:id", deps.StudentService.Get)
	students.Put("/:id", middleware.RequireRole("admin"), deps.StudentService.Update)
	students.Delete("/:id", middleware.RequireRole("admin"), deps.StudentService.Delete)

	// Endpoint 8: courses (semua role terautentikasi)
	api.Get("/courses",
		middleware.RequireAuth(deps.JWT),
		deps.CourseService.List)

	// Endpoint 9-10: enrollments (khusus mahasiswa)
	enrollments := api.Group("/enrollments",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
		middleware.RequireRole("mahasiswa"))

	enrollments.Post("/", deps.EnrollmentService.Enroll)
	enrollments.Delete("/:id", deps.EnrollmentService.Delete)
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", nil)
	}
}

type Dependencies struct {
	Pool              *pgxpool.Pool
	JWT               *helper.JWTManager
	AuthService       *service.AuthService
	StudentService    *service.StudentService
	CourseService     *service.CourseService
	EnrollmentService *service.EnrollmentService
}
