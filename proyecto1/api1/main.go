package main

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Crear app Fiber
	app := fiber.New()

	// Ruta raíz
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API1 en la VM1")
	})

	// Ruta para llamar a la API 2 (esta se encuentra en la misma vm)
	app.Get("/api1/202300350/llamar-api2", func(c *fiber.Ctx) error {
		res, err := http.Get("http://localhost:3002/")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error al llamar a la API 2")
		}
		defer res.Body.Close()

		buff := make([]byte, 1024)
		n, _ := res.Body.Read(buff)
		api2Response := string(buff[:n])

		// return JSON con la respuesta de la API 2 y un número aleatorio
		response := map[string]interface{}{
			"mensaje": fmt.Sprintf("Hola, responde la API: %s, desarrollada por el estudiante Abdiel Otzoy con carnet: 202300350", api2Response),
		}
		return c.JSON(response)
	})

	// Ruta para llamar a la API 3 (esta se encuentra en la vm2)
	app.Get("/api1/202300350/llamar-api3", func(c *fiber.Ctx) error {
		res, err := http.Get("http://vm2:3003/")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error al llamar a la API 3")
		}
		defer res.Body.Close()

		buff := make([]byte, 1024)
		n, _ := res.Body.Read(buff)

		api3Response := string(buff[:n])

		// return JSON con la respuesta de la API 3
		response := map[string]interface{}{
			"mensaje": fmt.Sprintf("Hola, responde la API: %s, desarrollada por el estudiante Abdiel Otzoy con carnet: 202300350", api3Response),
		}
		return c.JSON(response)
	})

	// Iniciar servidor en puerto 3001 en 0.0.0.0 que es accesible desde cualquier IP
	app.Listen("0.0.0.0:3001")
}
