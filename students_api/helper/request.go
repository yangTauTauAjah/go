package helper

import (
	"context"
	"strconv"
	"strings"
	"students_api/app/model"
	"time"

	"github.com/gofiber/fiber/v2"
)

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

var allowedSort = map[string]bool{
	"id": true, "username": true, "email": true, "created_at": true,
}

func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.Query("sort", "id"),
		Order:  strings.ToLower(c.Query("order", "asc")),
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}
	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}
	return q
}

// ParseCursorQuery membangun CursorQuery dari query string.
// Parameternya: cursor (opsional), limit (default 10, maks 100),
// search, dan is_active.
//
// Cursor yang dikirim client didecode di sini juga — gagal decode
// langsung menjadi 400 Bad Request, bukan diteruskan ke repository.
func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	q := model.CursorQuery{
		Limit:    limit,
		Search:   strings.TrimSpace(c.Query("search")),
		IsActive: nil,
	}
	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		cur, err := DecodeCursor(raw)
		if err != nil {
			return model.CursorQuery{}, BadRequest("cursor tidak valid")
		}
		q.After = &cur
	}
	return q, nil
}
