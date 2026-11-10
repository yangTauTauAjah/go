package route

import (
	"context"
	"students_api/app/service"
	"students_api/helper"
	"students_api/middleware"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Register(app *fiber.App, deps Dependencies) {
	if deps.EnableSwagger {
		// Swagger UI: http://localhost:3000/swagger/index.html
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	api := app.Group("/api/v1")
	api.Get("/health", healthCheck(deps.Pool, deps.MongoClient))

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
	students.Get("/:id/portfolio", deps.PortfolioService.Get)
	students.Put("/:id/portfolio", middleware.RequirePermission(perms, "student:update:any"), deps.PortfolioService.Upsert)
	students.Post("/:id/portfolio/projects", middleware.RequirePermission(perms, "student:update:any"), deps.PortfolioService.AddProject)
	students.Delete("/:id/portfolio/projects/:projectId", middleware.RequirePermission(perms, "student:update:any"), deps.PortfolioService.RemoveProject)
	achievements := api.Group("/achievements", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	achievements.Get("/", deps.AchievementService.List)
	achievements.Get("/:id", deps.AchievementService.Get)

	// --- kelas dan KRS ---
	courses := api.Group("/courses",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	courses.Get("/", deps.CourseService.List)
	courses.Get("/mine", deps.CourseService.Mine)
	courses.Get("/:id", deps.CourseService.Get)
	courses.Post("/:id/enroll", deps.CourseService.Enroll)
	courses.Delete("/:id/enroll", deps.CourseService.Drop)
	// PATCH /courses/:id mengubah kuota kelas. Route ini HARUS didaftarkan
	// SESUDAH /mine, kalau tidak "/mine" akan tertangkap oleh "/:id".
	courses.Patch("/:id", middleware.RequirePermission(perms, "course:manage"), deps.CourseService.UpdateQuota)
	courses.Get("/:id/reviews", deps.ReviewService.ListByCourse)
	courses.Get("/:id/reviews/summary", deps.ReviewService.Summary)
	courses.Post("/:id/reviews", deps.ReviewService.Create)
	enrollments := api.Group("/enrollments",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	enrollments.Post("/swap", deps.CourseService.Swap)

	// --- ulasan kelas (MongoDB) ---
	reviews := api.Group("/reviews",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	reviews.Get("/:id", deps.ReviewService.Get)
	reviews.Patch("/:id", deps.ReviewService.Update)
	reviews.Delete("/:id", deps.ReviewService.Delete)
	reviews.Post("/:id/replies", deps.ReviewService.Reply)
	reviews.Post("/:id/helpful", deps.ReviewService.MarkHelpful)

	// --- statistik portofolio lintas mahasiswa ---
	portfolios := api.Group("/portfolios", middleware.RequireAuth(deps.JWT))
	portfolios.Get("/stats/technologies", deps.PortfolioService.TechStats)

	// --- notifikasi (diisi oleh worker, dibaca lewat API) ---
	notifications := api.Group("/notifications",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	notifications.Get("/mine", deps.NotificationService.Mine)

	admin := api.Group("/admin",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	admin.Get("/queue", middleware.RequirePermission(perms, "queue:monitor"),
		deps.NotificationService.Monitor)

}

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("PostgreSQL tidak dapat dihubungi")
		}
		if err := mongoClient.Ping(ctx, nil); err != nil {
			return helper.ServiceUnavailable("MongoDB tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", nil)
	}
}

type Dependencies struct {
	Pool                *pgxpool.Pool
	MongoClient         *mongo.Client
	JWT                 *helper.JWTManager
	Permissions         *helper.PermissionSet
	StudentService      *service.StudentService
	AuthService         *service.AuthService
	CourseService       *service.CourseService
	ReviewService       *service.ReviewService
	NotificationService *service.NotificationService
	AchievementService  *service.AchievementService
	PortfolioService    *service.PortfolioService
	// EnableSwagger menyalakan Swagger UI di /swagger. Di production
	// sebaiknya dimatikan: dokumentasi lengkap adalah peta gratis bagi
	// penyerang.
	EnableSwagger bool
}
