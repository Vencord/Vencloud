package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"

	g "github.com/vencord/backend/globals"
	"github.com/vencord/backend/util"
)

type TelemetryBody struct {
	Plugins         []string `json:"plugins"`
	Version         string   `json:"version"`
	OperatingSystem string   `json:"operatingSystem"`
}

func POSTTelemetry(c *fiber.Ctx) error {
	telemetry := new(TelemetryBody)

	if err := c.BodyParser(telemetry); err != nil {
		return err
	}

	telemetryId := util.Hash(g.PEPPER_TELEMETRY + c.IP())

	_, err := g.RDB.HSet(c.Context(), "telemetry:"+telemetryId, map[string]interface{}{
		"plugins":         telemetry.Plugins,
		"version":         telemetry.Version,
		"operatingSystem": telemetry.OperatingSystem,
	}).Result()

	if err != nil {
		panic(err)
	}

	_, err = g.RDB.Expire(c.Context(), "telemetry:"+telemetryId, 3*24*time.Hour).Result()

	if err != nil {
		panic(err)
	}

	return c.SendStatus(204)
}
