package aero

import "net/http"

type OptionsHandler func(allowed string, c *Ctx) error

func defaultOptionsHandler(allowed string, c *Ctx) error {
	c.SetHeader("Allow", allowed)
	return c.SendStatus(http.StatusNoContent)
}
