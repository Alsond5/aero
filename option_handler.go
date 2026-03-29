package aero

import "net/http"

type OptionsHandler func(allowed string, c *Ctx)

func defaultOptionsHandler(allowed string, c *Ctx) {
	c.SetHeader("Allow", allowed)
	c.SendStatus(http.StatusNoContent)
}
