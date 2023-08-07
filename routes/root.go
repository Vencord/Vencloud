package routes

import (
	"github.com/gofiber/fiber/v2"

	g "github.com/vencord/backend/globals"
	"github.com/vencord/backend/util"
)

// /v1

func DELETE(c *fiber.Ctx) error {
	userId := c.Context().UserValue("userId").(string)

	g.RDB.Del(c.Context(), "settings:"+util.Hash(g.PEPPER_SETTINGS+userId))
	g.RDB.Del(c.Context(), "secrets:"+util.Hash(g.PEPPER_SECRETS+userId))

	return c.SendStatus(204)
}

func GET(c *fiber.Ctx) error {
	return c.JSON(&fiber.Map{
		"ping": "pong",
	})
}
