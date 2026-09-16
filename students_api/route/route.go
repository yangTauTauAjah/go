package route

import (
	"context"
	"time"
	"tugas2/app/service"
	"tugas2/helper"
	"tugas2/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")
	api.Get("/health", healthCheck(deps.Pool))

	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	students := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	students.Get("/", deps.StudentService.List)
	students.Get("/:id", deps.StudentService.Get)
	students.Post("/", deps.StudentService.Create)
	students.Put("/:id", deps.StudentService.Replace)
	students.Patch("/:id", deps.StudentService.Patch)
	students.Delete("/:id", deps.StudentService.Delete)

	achievements := api.Group("/achievements", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	achievements.Get("/", deps.AchievementService.List)
	achievements.Get("/:id", deps.AchievementService.Get)

}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable,
				"database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", nil)
	}
}

type Dependencies struct {
	Pool               *pgxpool.Pool
	JWT                *helper.JWTManager
	AuthService        *service.AuthService
	StudentService     *service.StudentService
	AchievementService *service.AchievementService
}
