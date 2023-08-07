package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	g "github.com/vencord/backend/globals"
	"github.com/vencord/backend/util"
)

// /v1/settings

func HEADSettings(c *fiber.Ctx) error {
	userId := c.Context().UserValue("userId").(string)

	written, err := g.RDB.HGet(c.Context(), "settings:"+util.Hash(g.PEPPER_SETTINGS+userId), "written").Result()

	if err == redis.Nil {
		return c.Status(404).Send(nil)
	} else if err != nil {
		panic(err)
	}

	c.Set("ETag", written)
	return c.SendStatus(204)
}

func GETSettings(c *fiber.Ctx) error {
	userId := c.Context().UserValue("userId").(string)

	settings, err := g.RDB.HMGet(c.Context(), "settings:"+util.Hash(g.PEPPER_SETTINGS+userId), "value", "written").Result()

	// we shouldn't expect an error here, HMGet doesn't return one
	if err != nil {
		panic(err)
	}

	if settings[0] == nil {
		return c.Status(404).Send(nil)
	}

	// value is compressed data, written is a timestamp
	value, written := []byte(settings[0].(string)), settings[1].(string)

	if ifm := c.Get("if-none-match"); ifm == written {
		return c.SendStatus(304)
	}

	c.Set("Content-Type", "application/octet-stream")
	c.Set("ETag", written)
	return c.Send(value)
}

func PUTSettings(c *fiber.Ctx) error {
	if c.Get("Content-Type") != "application/octet-stream" {
		return c.Status(415).JSON(&fiber.Map{
			"error": "Content type must be application/octet-stream",
		})
	}

	if len(c.Body()) > g.SIZE_LIMIT {
		return c.Status(413).JSON(&fiber.Map{
			"error": "Settings are too large",
		})
	}

	userId := c.Context().UserValue("userId").(string)

	now := time.Now().UnixMilli()

	_, err := g.RDB.HSet(c.Context(), "settings:"+util.Hash(g.PEPPER_SETTINGS+userId), map[string]interface{}{
		"value":   c.Body(),
		"written": now,
	}).Result()

	if err != nil {
		panic(err)
	}

	return c.JSON(&fiber.Map{
		"written": now,
	})
}

func DELETESettings(c *fiber.Ctx) error {
	userId := c.Context().UserValue("userId").(string)

	g.RDB.Del(c.Context(), "settings:"+util.Hash(g.PEPPER_SETTINGS+userId))

	return c.SendStatus(204)
}
