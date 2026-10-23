package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// validate dibuat SEKALI untuk seluruh aplikasi.
//
// validator.New() melakukan refleksi dan menyimpan hasilnya dalam cache
// internal. Membuatnya ulang pada setiap request berarti membuang cache
// tersebut berkali-kali — mahal dan tidak ada gunanya.
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	// Tanpa ini, pesan error menyebut nama field Go ("Username"),
	// padahal client mengirim dan membaca nama JSON ("username").
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	// Aturan buatan sendiri. Aturan yang tidak disediakan library tetap
	// ditulis secara deklaratif sebagai tag, bukan dikembalikan menjadi
	// pemeriksaan manual yang tersebar di dalam service.
	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), " \t\n\r")
	})
	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		for _, r := range fl.Field().String() {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
				r != '.' && r != '_' {
				return false
			}
		}
		return true
	})
	_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		// passwordStrength mengembalikan "" untuk sandi yang LOLOS.
		// Tag ini valid ketika sandi kuat, jadi kita uji == "", bukan != "".
		return passwordStrength(fl.Field().String()) == ""
	})
	return v
}

// ValidateStruct menjalankan seluruh aturan pada tag struct dan
// mengembalikan peta nama field ke pesan berbahasa Indonesia.
// Mengembalikan nil berarti tidak ada pelanggaran.
func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	// Terjadi bila yang dikirim bukan struct — itu kesalahan programmer,
	// bukan kesalahan pemakai API. Jangan diam-diam dianggap valid.
	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}
	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}
	result := make(map[string]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		if _, exists := result[fe.Field()]; !exists {
			result[fe.Field()] = messageFor(fe)
		}
	}
	return result
}

// messageFor menerjemahkan satu ValidationError menjadi pesan
// berbahasa Indonesia. Nama aturan di sini (required, email, ...)
// adalah bawaan validator/v10; aturan custom (nospace, username,
// strongpassword) ditangani di cabang default dengan menyalin alasan
// yang dihasilkan oleh rule-nya sendiri.
func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		switch fe.Kind() {
		case reflect.String:
			return "minimal " + fe.Param() + " karakter"
		case reflect.Slice, reflect.Array, reflect.Map:
			return "minimal " + fe.Param() + " item"
		}
		return "nilai minimal " + fe.Param()
	case "max":
		switch fe.Kind() {
		case reflect.String:
			return "maksimal " + fe.Param() + " karakter"
		case reflect.Slice, reflect.Array, reflect.Map:
			return "maksimal " + fe.Param() + " item"
		}
		return "nilai maksimal " + fe.Param()
	case "alphanum":
		return "hanya boleh huruf dan angka"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "username":
		return "hanya boleh huruf, angka, titik, dan garis bawah"
	case "strongpassword":
		return "password terlalu lemah (minimal 8 karakter, huruf dan angka)"
	case "omitnil":
		// omitnil tidak pernah gagal saat ini — pointer nol berarti
		// "tidak dikirim", yang memang aturan main PATCH.
		return ""
	default:
		return "tidak valid"
	}
}

// passwordStrength mengembalikan "" untuk sandi yang lolos,
// atau pesan kesalahan untuk yang tidak. Aturannya sama dengan yang
// dipakai auth_rules.checkPasswordStrength, diletakkan di sini
// agar tag "strongpassword" pada struct dapat memanfaatkannya.
func passwordStrength(s string) string {
	const minPasswordLength = 8
	if len(s) < minPasswordLength {
		return "minimal 8 karakter"
	}
	var hasLetter, hasDigit bool
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "harus memuat huruf dan angka"
	}
	return ""
}
