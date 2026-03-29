package main

import "github.com/Alsond5/aero"

func main() {
	app := aero.New()

	app.GET("/api/users/:id", func(c *aero.Ctx) error {
		val := c.Param("id")

		return c.JSON(val)
	})

	app.POST("/api/users/:id", func(c *aero.Ctx) error {
		val := c.Param("id")

		return c.JSON(val)
	})

	app.Listen(":8585")
}
