// Vencord Cloud, the API for the Vencord client mod
// Copyright (C) 2023 Vendicated and contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"encoding/base64"
	"os"
	"strconv"
	"strings"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"

	g "github.com/vencord/backend/globals"
	"github.com/vencord/backend/routes"
	"github.com/vencord/backend/util"
)

func requireAuth(c *fiber.Ctx) error {
	authToken := c.Get("Authorization")

	if authToken == "" {
		return c.Status(401).JSON(&fiber.Map{
			"error": "Missing authorization",
		})
	}

	// decode base64 token and split by:
	// token[0] = secret
	// token[1] = user id
	token, err := base64.StdEncoding.DecodeString(authToken)

	if err != nil {
		return c.Status(401).JSON(&fiber.Map{
			"error": "Invalid authorization",
		})
	}

	tokenStr := string(token)
	tokenSplit := strings.Split(tokenStr, ":")

	if len(tokenSplit) != 2 {
		return c.Status(401).JSON(&fiber.Map{
			"error": "Invalid authorization",
		})
	}

	secret := tokenSplit[0]
	userId := tokenSplit[1]

	if g.ALLOWED_USERS != nil && c.Path() != "/v1" && c.Method() != "DELETE" && !g.ALLOWED_USERS[userId] {
		return c.Status(403).JSON(&fiber.Map{
			"error": "User is not whitelisted",
		})
	}

	storedSecret, err := g.RDB.Get(c.Context(), "secrets:"+util.Hash(g.PEPPER_SECRETS+userId)).Result()

	if err == redis.Nil {
		return c.Status(401).JSON(&fiber.Map{
			"error": "Invalid authorization",
		})
	} else if err != nil {
		panic(err)
	}

	if storedSecret != secret {
		return c.Status(401).JSON(&fiber.Map{
			"error": "Invalid authorization",
		})
	}

	c.Context().SetUserValue("userId", userId)

	return c.Next()
}

func main() {
	// environment
	slRaw, _ := strconv.ParseInt(os.Getenv("SIZE_LIMIT"), 10, 0)
	g.SIZE_LIMIT = int(slRaw)

	auRaw := os.Getenv("ALLOWED_USERS")
	if auRaw != "" {
		g.ALLOWED_USERS = make(map[string]bool)
		for _, userId := range strings.Split(auRaw, ",") {
			g.ALLOWED_USERS[userId] = true
		}
	}

	app := fiber.New(fiber.Config{
        ProxyHeader: os.Getenv("PROXY_HEADER"),
    })

	g.RDB = redis.NewClient(&redis.Options{
		Addr: g.REDIS_URI,
		Password: g.REDIS_PASS,
	})

	if os.Getenv("PROMETHEUS") == "true" {
		promauto.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "vencord_accounts_registered",
			Help: "The total number of accounts registered",
		}, func() float64 {
			iter := g.RDB.Scan(context.Background(), 0, "secrets:*", 0).Iterator()
			var count int64

			for iter.Next(context.Background()) {
				count++
			}

			if err := iter.Err(); err != nil {
				panic(err)
			}

			return float64(count)
		})

		prometheus := fiberprometheus.New("vencord")
		prometheus.RegisterAt(app, "/metrics")
		app.Use(prometheus.Middleware)
	}

	app.Use(cors.New(cors.Config{
		ExposeHeaders: "ETag",
		AllowOrigins:  "https://discord.com,https://ptb.discord.com,https://canary.discord.com,https://discordapp.com,https://ptb.discordapp.com,https://canary.discordapp.com",
	}))
	app.Use(logger.New())

	// #region settings
	app.All("/v1/settings", requireAuth)

	app.Head("/v1/settings", routes.HEADSettings)
	app.Get("/v1/settings", routes.GETSettings)
	app.Put("/v1/settings", routes.PUTSettings)
	app.Delete("/v1/settings", routes.DELETESettings)
	// #endregion

	// #region discord oauth
	app.Get("/v1/oauth/callback", routes.GETOAuthCallback)
	app.Get("/v1/oauth/settings", routes.GETOAuthSettings)
	// #endregion

	// #region erase all
	app.Delete("/v1", requireAuth, routes.DELETE)
	// #endregion

	app.Get("/v1", routes.GET)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect(g.ROOT_REDIRECT, 303)
	})

	app.Listen(g.HOST + ":" + g.PORT)
}
