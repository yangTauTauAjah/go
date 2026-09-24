package route

import (
	"context"
	"students_api/app/service"
	"students_api/helper"
	"students_api/middleware"
	"time"

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

	perms := deps.Permissions
	students.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)
	students.Post("/", middleware.RequirePermission(perms, "student:update:any"), deps.StudentService.Create)
	students.Get("/:id", deps.StudentService.Get)
	students.Put("/:id", middleware.RequirePermission(perms, "student:update:any"), deps.StudentService.Replace)
	students.Patch("/:id", middleware.RequirePermission(perms, "student:update:any"), deps.StudentService.Patch)
	students.Patch("/:id/role", middleware.RequirePermission(perms, "role:assign"), deps.StudentService.AssignRole)
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete)

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
	Permissions        *helper.PermissionSet
	StudentService     *service.StudentService
	AchievementService *service.AchievementService
}
