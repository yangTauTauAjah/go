package helper

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	_ = v.RegisterValidation("tahunakademik", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		// Format: 2026/2027Ganjil atau 2026/2027Genap
		if s[:9] != "2026/2027" && s[:9] != "2027/2028" {
			return false
		}
		// _, err := time.Parse("2006/2007", s[:9])
		// if err != nil {
		// 	return false
		// }
		akhir := s[9:]
		return akhir == "Ganjil" || akhir == "Genap"
	})
	_ = v.RegisterValidation("ipkrange", func(fl validator.FieldLevel) bool {
		v := fl.Field().Float()
		return v >= 0.00 && v <= 4.00
	})
	return v
}

func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
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

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return fmt.Sprintf("%s minimal %s karakter", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s maksimal %s karakter", fe.Field(), fe.Param())
	case "len":
		return fmt.Sprintf("%s harus %s karakter", fe.Field(), fe.Param())
	case "tahunakademik":
		return "tahun_akademik harus berformat 2026/2027Ganjil atau 2026/2027Genap"
	case "ipkrange":
		return "ipk_terakhir harus antara 0,00 dan 4,00"
	case "gte":
		return fmt.Sprintf("%s minimal %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s maksimal %s", fe.Field(), fe.Param())
	}
	return fe.Field() + " tidak valid"
}
