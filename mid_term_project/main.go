package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"siakad/app/repository"
	"siakad/app/service"
	"siakad/config"
	"siakad/database"
	"siakad/helper"
	"siakad/route"
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
		config.GetEnv("JWT_ISSUER", "siakad-mini"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 60))*time.Minute,
	)

	// 2. Database
	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Repository
	studentRepo := repository.NewStudentRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	enrollmentRepo := repository.NewEnrollmentRepository(pool)

	// 4. Service
	authService := service.NewAuthService(studentRepo, studentRepo, jwtManager)
	studentService := service.NewStudentService(studentRepo, enrollmentRepo)
	courseService := service.NewCourseService(courseRepo)
	enrollmentService := service.NewEnrollmentService(studentRepo, enrollmentRepo)

	// 5. App
	app := config.NewApp(logger, route.Dependencies{
		Pool:              pool,
		JWT:               jwtManager,
		AuthService:       authService,
		StudentService:    studentService,
		CourseService:     courseService,
		EnrollmentService: enrollmentService,
	})

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("server berjalan", slog.String("port", port))

	// 6. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("sinyal berhenti diterima, menutup server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("gagal menutup server dengan rapi", slog.String("error", err.Error()))
	}
	logger.Info("server berhenti dengan rapi")
}
