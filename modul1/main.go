package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

type Student struct {
	ID       string
	Name     string
	Field    string
	Semester int
	Grade    float64
	IsActive bool
	Course   []string
	Address  string
}

func main() {
	app := fiber.New()

	VariableDeclaration()
	Pointer()
	Struct()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
	log.Fatal(app.Listen(":3000"))
}
