package helper

import (
	"context"
	"strconv"
	"strings"
	"time"

	"siakad/app/model"

	"github.com/gofiber/fiber/v2"
)

const LocalsAuthUser = "authUser"

func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}

func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func RequestID(c *fiber.Ctx) string {
	if v, ok := c.Locals("requestid").(string); ok {
		return v
	}
	return ""
}

var allowedSort = map[string]bool{
	"id": true, "nim": true, "nama": true, "ipk_terakhir": true, "created_at": true,
}

func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	// Terima kedua nama: per_page (standar API) atau limit (alias).
	// Default 10, maksimal 50 untuk menghindari query besar.
	limit := c.QueryInt("per_page", 0)
	if limit <= 0 {
		limit = c.QueryInt("limit", 10)
	}
	if limit <= 0 {
		limit = 10
	}
	q := model.ListQuery{
		Page:  c.QueryInt("page", 1),
		Limit: limit,
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		q.Search = s
	}
	if v := c.Query("prodi"); v != "" {
		q.Prodi = v
	}
	if v := c.Query("angkatan"); v != "" {
		if ang, err := strconv.Atoi(v); err == nil && ang > 0 {
			q.Angkatan = &ang
		}
	}
	q.Sort = c.Query("sort", "id")
	q.Order = strings.ToLower(c.Query("order", "asc"))
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 50 {
		q.Limit = 50
	}
	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}
	return q
}
