package route

import (
	"context"
	"students_api/app/service"
	"students_api/helper"
	"students_api/middleware"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Register(app *fiber.App, deps Dependencies) {
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
	// --- portofolio mahasiswa (MongoDB) ---
	// Portfolio adalah milik satu mahasiswa, jadi endpoint-nya menempel pada
	// /students/:id. Middleware RequirePermission student:update:any dipasang
	// pada operasi tulis karena mengubah portofolio setara mengubah data
	// mahasiswa; pembacaan cukup dengan token yang valid.
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
	// "/mine" didaftarkan SEBELUM "/:id". Fiber mencocokkan route sesuai
	// urutan pendaftaran; bila dibalik, "mine" akan dianggap sebagai :id.
	courses.Get("/", deps.CourseService.List)
	courses.Get("/mine", deps.CourseService.Mine)
	courses.Get("/:id", deps.CourseService.Get)
	courses.Post("/:id/enroll", deps.CourseService.Enroll)
	courses.Delete("/:id/enroll", deps.CourseService.Drop)
	enrollments := api.Group("/enrollments",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	enrollments.Post("/swap", deps.CourseService.Swap)
	courses.Get("/:id/reviews", deps.ReviewService.ListByCourse)
	courses.Get("/:id/reviews/summary", deps.ReviewService.Summary)
	courses.Post("/:id/reviews", deps.ReviewService.Create)
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
	// Group terpisah, bukan /students/:id/... , karena endpoint ini
	// menggabungkan SEMUA portofolio. Group terpisah juga membuat
	// "stats" tidak pernah tertangkap oleh pola /students/:id.
	portfolios := api.Group("/portfolios", middleware.RequireAuth(deps.JWT))
	portfolios.Get("/stats/technologies", deps.PortfolioService.TechStats)

}

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		print("healthCheck called\n")
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
	Pool               *pgxpool.Pool
	MongoClient        *mongo.Client
	JWT                *helper.JWTManager
	AuthService        *service.AuthService
	Permissions        *helper.PermissionSet
	StudentService     *service.StudentService
	AchievementService *service.AchievementService
	CourseService      *service.CourseService
	ReviewService      *service.ReviewService
	PortfolioService   *service.PortfolioService
}
