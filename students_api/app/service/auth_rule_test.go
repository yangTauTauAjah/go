package service

import (
	"strings"
	"testing"

	"tugas2/app/model"
)

// TestValidateRegister menjamin ValidateRegister menerima permintaan
// pendaftaran yang valid dan menolak semua bentuk pelanggaran satu per satu.
// Inilah sebabnya ValidateRegister ditulis sebagai pure function: dapat
// diuji dengan satu struct + satu map[string]string, tanpa HTTP, DB, atau
// goroutine.
func TestValidateRegister(t *testing.T) {
	t.Run("valid_request_returns_no_errors", func(t *testing.T) {
		// Pemakaian hash nyata tidak relevan di sini — yang diuji adalah
		// aturan bentuk input, bukan hasil bcrypt.
		errs := ValidateRegister(model.RegisterRequest{
			Username: "johndoe",
			Email:    "john@example.com",
			Password: "rahasia123",
		})
		if len(errs) != 0 {
			t.Fatalf("permintaan valid seharusnya tidak mengembalikan error, dapat %v", errs)
		}
	})

	t.Run("weak_password_is_rejected", func(t *testing.T) {
		// "password1" ada di daftar password lemah internal. Meskipun panjang
		// sudah 9 karakter dan memenuhi aturan huruf+angka, ia tetap ditolak.
		// Pemisahan aturan "memuat huruf+angka" dan "tidak lemah" penting agar
		// penggugat tidak asal menerima semua sandi yang panjangnya cukup.
		errs := ValidateRegister(model.RegisterRequest{
			Username: "johndoe",
			Email:    "john@example.com",
			Password: "password1",
		})
		if msg, ok := errs["password"]; !ok {
			t.Errorf("password lemah seharusnya ditolak: %v", errs)
		} else if !strings.Contains(msg, "lemah") && !strings.Contains(msg, "umum") {
			t.Errorf("pesan error lemah tidak spesifik: %q", msg)
		}
	})

	t.Run("username_with_invalid_chars_is_rejected", func(t *testing.T) {
		// Username hanya boleh huruf, angka, titik, dan garis bawah.
		// Tanda dash (-) terlihat wajar bagi manusia tetapi akan merusak
		// banyak use case seperti slug URL. Inilah sebabnya whitelist
		// chars dipakai daripada blacklist.
		errs := ValidateRegister(model.RegisterRequest{
			Username: "john-doe",
			Email:    "john@example.com",
			Password: "rahasia123",
		})
		if _, ok := errs["username"]; !ok {
			t.Errorf("username dengan tanda dash seharusnya ditolak: %v", errs)
		}
	})
}

// TestValidateLogin menegaskan bahwa ValidateLogin TIDAK memeriksa
// kekuatan password. Aturan ini penting dan tercantum di kode:
// "password lama mungkin dibuat sebelum aturannya berubah."
// Bila ValidateLogin ternyata menolak password lemah, semua akun lama
// akan terkunci keluar (lockout massal) ketika aturan diperketat.
func TestValidateLogin(t *testing.T) {
	t.Run("weak_password_passes_login_validation", func(t *testing.T) {
		// "123" jelas bukan sandi kuat. ValidateLogin hanya memastikan
		// field ada — keputusan benar/salah diberikan oleh bcrypt.VerifyPassword.
		errs := ValidateLogin(model.LoginRequest{
			Username: "johndoe",
			Password: "123",
		})
		if len(errs) != 0 {
			t.Errorf("ValidateLogin seharusnya tidak menilai kekuatan, dapat %v", errs)
		}
	})

	t.Run("empty_username_is_rejected", func(t *testing.T) {
		errs := ValidateLogin(model.LoginRequest{
			Username: "   ",
			Password: "anything",
		})
		if _, ok := errs["username"]; !ok {
			t.Errorf("username kosong seharusnya ditolak: %v", errs)
		}
	})
}

// TestIsValidUsername bersifat table-driven untuk menjaga keyakinan bahwa
// implementasi saat ini memilih karakter mana yang masuk whitelist.
// Implementasi menggunakan unicode.IsLetter, jadi karakter Latin
// ber-diakritik (mis. jõhn) diterima, sementara spasi dan simbol yang
// bukan huruf/angka/titik/garis bawah ditolak.
func TestIsValidUsername(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"johndoe", true},
		{"john.doe", true},
		{"john_doe", true},
		{"a1b2c3", true},
		{"jõhn", true}, // unicode.IsLetter menerima Latin Extended
		{"", true},     // string kosong — tidak ada karakter yang melanggar
		{"john-doe", false},
		{"john doe", false},
		{"john@doe", false},
		{"john/doe", false},
	}
	for _, tc := range cases {
		if got := isValidUsername(tc.in); got != tc.want {
			t.Errorf("isValidUsername(%q) = %v, harap %v", tc.in, got, tc.want)
		}
	}
}
