package config

import (
	"errors"
	"log/slog"
	"students_api/app/model"
	"students_api/helper"
	"students_api/middleware"
	"students_api/route"

	"github.com/gofiber/fiber/v2"
)

func NewApp(logger *slog.Logger, dependencies route.Dependencies) *fiber.App {

	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "Praktikum Backend Lanjut"),
		ErrorHandler: newErrorHandler(logger),
		BodyLimit:    1 * 1024 * 1024,
	})
	middleware.Register(app, logger, GetEnv("ALLOWED_ORIGINS", ""))
	route.Register(app, dependencies)
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})
	return app
}

// newErrorHandler adalah SATU-SATUNYA tempat error berubah menjadi
// response HTTP di seluruh aplikasi.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID := helper.RequestID(c)
		var appErr *helper.AppError
		switch {
		case errors.As(err, &appErr):
		// Kegagalan yang sudah kita rencanakan.
		case errors.Is(err, fiber.ErrRequestEntityTooLarge):
			appErr = &helper.AppError{
				Status:  fiber.StatusRequestEntityTooLarge,
				Code:    "PAYLOAD_TOO_LARGE",
				Message: "ukuran body melebihi batas yang diizinkan",
			}
		default:
			// Kegagalan yang tidak kita duga.
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				appErr = &helper.AppError{
					Status: fiberErr.Code, Code: "HTTP_ERROR",
					Message: fiberErr.Message,
				}
			} else {
				appErr = helper.Internal(err)
			}
		}
		// Hanya kegagalan sisi server yang dicatat sebagai Error.
		// Kegagalan 4xx adalah kesalahan pemakai API, bukan kerusakan
		// sistem; mencatatnya sebagai Error membuat log penuh bising
		// sehingga kerusakan yang sesungguhnya justru tenggelam.
		if appErr.Status < fiber.StatusInternalServerError {
			logger.Error("request_failed",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status),
				slog.String("error", appErr.Cause.Error()))
		} else {
			logger.Warn("request_rejected",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status))
		}
		return c.Status(appErr.Status).JSON(model.ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			Fields:    appErr.Fields,
			RequestID: requestID,
		})
	}
}
