package helper

import (
	"github.com/gofiber/fiber/v2"

	"students_api/app/model"
)

// LocalsAuthUser adalah kunci penyimpanan identitas pemakai di dalam
// context request. Dibuat sebagai konstanta agar tidak ada salah ketik
// antara tempat menyimpan dan tempat membaca.
const LocalsAuthUser = "authUser"

// localsRequestID adalah kunci yang dipasang oleh middleware requestid.
// Kita bungkus ke fungsi RequestID() agar sisa aplikasi tidak perlu
// mengingat nama kunci tersebut.
const localsRequestID = "requestid"

// RequestID mengembalikan ID permintaan yang dipasang middleware
// requestid. Bila middleware belum dipasang (mis. pada unit test),
// kembalikan string kosong agar log tidak error.
func RequestID(c *fiber.Ctx) string {
	id, _ := c.Locals(localsRequestID).(string)
	return id
}

func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}
