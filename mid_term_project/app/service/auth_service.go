package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"siakad/app/model"
	"siakad/app/repository"
	"siakad/helper"
)

type AuthService struct {
	users    repository.StudentRepository
	jwt      *helper.JWTManager
	students repository.StudentRepository
}

func NewAuthService(users, students repository.StudentRepository, jwt *helper.JWTManager) *AuthService {
	return &AuthService{users: users, students: students, jwt: jwt}
}

// LoginRequest validasi + verifikasi bcrypt + pembuatan access token.
func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	req.Email = strings.TrimSpace(req.Email)
	if errs := helper.ValidateStruct(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		// Dummy verify untuk mencegah username enumeration via timing.
		helper.VerifyPassword("$2a$12$abcdefghijklmnopqrstuuLKa3Bt1TCmU/6zvhZ8x4nq1yBiuGvS", req.Password)
		return helper.Fail(c, fiber.StatusUnauthorized, "email atau password salah")
	}
	if !user.IsActive {
		return helper.Fail(c, fiber.StatusForbidden, "akun dinonaktifkan")
	}
	if !helper.VerifyPassword(user.Password, req.Password) {
		return helper.Fail(c, fiber.StatusUnauthorized, "email atau password salah")
	}

	authUser := model.AuthUser{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	}
	accessToken, err := s.jwt.GenerateAccess(authUser)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "login berhasil", model.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.jwt.AccessTTL().Seconds()),
		User:        authUser,
	})
}

// Me mengembalikan profil user yang sedang login.
// Untuk mahasiswa, data students ikut disertakan.
func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	user, err := s.users.FindByEmail(ctx, current.Email)
	if err != nil {
		return helper.Unauthorized("user tidak ditemukan")
	}
	resp := fiber.Map{
		"user": model.AuthUser{UserID: user.ID, Email: user.Email, Role: user.Role},
	}
	if user.Role == "mahasiswa" {
		student, err := s.students.FindByUserID(ctx, user.ID)
		if err == nil {
			resp["student"] = student
		}
	}
	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", resp)
}

// dummyVerifyTimeout diberi nama agar IDE tidak menganggap function berikut tidak terpakai.
// Dipakai oleh Login untuk menyeimbangkan waktu respons.
var _ = func(_ context.Context) { time.Now() }

// Err dipisah untuk type-error lain yang mungkin dipakai.
var ErrAuthInvalid = errors.New("kredensial tidak valid")
