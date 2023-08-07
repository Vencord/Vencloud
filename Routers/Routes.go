package Routers

import(
		"github.com/gofiber/fiber/v2"
		"github.com/Vencord/Backend/Handlers"
		"github.com/gofiber/fiber/v2/middleware/cors"
		"github.com/gofiber/fiber/v2/middleware/logger"
)

func Initalize(app *fiber.App){
	app.Use(cors.New(cors.Config{
		ExposeHeaders: "ETag",
        AllowOrigins: "https://discord.com,https://ptb.discord.com,https://canary.discord.com",
	}))
	app.Use(logger.New())

	app.All("/v1/settings", Handlers.RequireAuth)
	app.Head("/v1/settings", Handlers.V1SettingsHead)
	app.Get("/v1/settings", Handlers.V1SettingsGet)
	app.Put("/v1/settings", Handlers.V1SettingsPut)
	app.Delete("/v1/settings", Handlers.V1SettingsDelete)

	app.Get("/v1/oauth/callback", Handlers.V1OauthCallback)
	app.Get("/v1/oauth/settings", Handlers.V1OauthSettings)

	app.Delete("/v1", Handlers.RequireAuth, Handlers.V1Delete)

	app.Get("/v1", Handlers.V1Get)
	app.Get("/", Handlers.Root)

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"code":    404,
			"message": "404: Not Found",
		})
	})

}
