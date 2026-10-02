package middleware

import (
	"github.com/gofiber/fiber/v2"

	"siakad/helper"
)

// RequireRole menolak request yang role-nya tidak termasuk dalam daftar.
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
		}
		if _, granted := allowed[user.Role]; !granted {
			return helper.Fail(c, fiber.StatusForbidden,
				"role Anda tidak berhak mengakses endpoint ini")
		}
		return c.Next()
	}
}
