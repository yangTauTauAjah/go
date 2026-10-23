package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"students_api/app/repository"
	"students_api/app/service"
	"students_api/config"
	"students_api/database"
	"students_api/helper"
	"students_api/route"
	"syscall"
	"time"
)

const minSecretLength = 32

func main() {

	// 1. Konfigurasi dan logger
	config.LoadEnv()
	logger := config.NewLogger()

	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek",
			slog.Int("minimal_karakter", minSecretLength))
		os.Exit(1)
	}

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	// 2. Database
	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	// Disconnect menutup seluruh koneksi di pool milik driver. Diberi batas
	// waktu sendiri, karena context milik request sudah tidak berlaku saat
	// aplikasi berhenti.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(ctx)
	}()
	mongoDB := mongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))

	userRepository := repository.NewStudentRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)
	roleRepository := repository.NewRoleRepository(pool)
	achievementRepository := repository.NewAchievementRepository(pool)
	courseRepository := repository.NewCourseRepository(pool)
	txManager := repository.NewTxManager(pool)
	reviewRepository := repository.NewReviewRepository(mongoDB)
	if err := reviewRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}

	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		os.Exit(1)
	}
	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	studentService := service.NewStudentService(userRepository, permissions)
	authService := service.NewAuthService(
		userRepository, tokenRepository, jwtManager, permissions,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
	)
	achievementService := service.NewAchievementService(achievementRepository)
	courseService := service.NewCourseService(courseRepository, txManager)
	reviewService := service.NewReviewService(reviewRepository, courseRepository, permissions)

	app := config.NewApp(logger, route.Dependencies{
		Pool:               pool,
		MongoClient:        mongoClient,
		JWT:                jwtManager,
		Permissions:        permissions,
		StudentService:     studentService,
		AuthService:        authService,
		AchievementService: achievementService,
		CourseService:      courseService,
		ReviewService:      reviewService,
	})

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("server berjalan", slog.String("port", port))

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("sinyal berhenti diterima, menutup server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("gagal menutup server dengan rapi",
			slog.String("error", err.Error()))
	}
	logger.Info("server berhenti dengan rapi")
}
