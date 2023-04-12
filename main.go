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
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	reqHttp "github.com/imroc/req/v3"
	"github.com/redis/go-redis/v9"
)

type DiscordAccessTokenResult struct {
	AccessToken string `json:"access_token"`
}

type DiscordUserResult struct {
	Id string `json:"id"`
}

var ALLOWED_USERS map[string]bool

var rdb *redis.Client

func hash(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

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

	if ALLOWED_USERS != nil && c.Path() != "/v1" && c.Method() != "DELETE" && !ALLOWED_USERS[userId] {
		return c.Status(403).JSON(&fiber.Map{
			"error": "User is not whitelisted",
		})
	}

	storedSecret, err := rdb.Get(c.Context(), "secrets:"+hash(os.Getenv("PEPPER_SECRETS")+userId)).Result()

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
	HOST := os.Getenv("HOST")
	PORT := os.Getenv("PORT")
	REDIS_URI := os.Getenv("REDIS_URI")
	ROOT_REDIRECT := os.Getenv("ROOT_REDIRECT")

	DISCORD_CLIENT_ID := os.Getenv("DISCORD_CLIENT_ID")
	DISCORD_CLIENT_SECRET := os.Getenv("DISCORD_CLIENT_SECRET")
	DISCORD_REDIRECT_URI := os.Getenv("DISCORD_REDIRECT_URI")

	PEPPER_SECRETS := os.Getenv("PEPPER_SECRETS")
	PEPPER_SETTINGS := os.Getenv("PEPPER_SETTINGS")

	slRaw, _ := strconv.ParseInt(os.Getenv("SIZE_LIMIT"), 10, 0)
	SIZE_LIMIT := int(slRaw)

	auRaw := os.Getenv("ALLOWED_USERS")
	if auRaw != "" {
		ALLOWED_USERS = make(map[string]bool)
		for _, userId := range strings.Split(auRaw, ",") {
			ALLOWED_USERS[userId] = true
		}
	}

	app := fiber.New()
	rdb = redis.NewClient(&redis.Options{
		Addr: REDIS_URI,
	})
	req := reqHttp.C()

	app.Use(cors.New(cors.Config{
		ExposeHeaders: "ETag",
	}))
	app.Use(logger.New())

	// #region settings
	app.All("/v1/settings", requireAuth)

	app.Head("/v1/settings", func(c *fiber.Ctx) error {
		userId := c.Context().UserValue("userId").(string)

		written, err := rdb.HGet(c.Context(), "settings:"+hash(PEPPER_SETTINGS+userId), "written").Result()

		if err == redis.Nil {
			return c.Status(404).Send(nil)
		} else if err != nil {
			panic(err)
		}

		c.Set("ETag", written)
		return c.SendStatus(204)
	})

	app.Get("/v1/settings", func(c *fiber.Ctx) error {
		userId := c.Context().UserValue("userId").(string)

		settings, err := rdb.HMGet(c.Context(), "settings:"+hash(PEPPER_SETTINGS+userId), "value", "written").Result()

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
	})

	app.Put("/v1/settings", func(c *fiber.Ctx) error {
		if c.Get("Content-Type") != "application/octet-stream" {
			return c.Status(415).JSON(&fiber.Map{
				"error": "Content type must be application/octet-stream",
			})
		}

		if len(c.Body()) > SIZE_LIMIT {
			return c.Status(413).JSON(&fiber.Map{
				"error": "Settings are too large",
			})
		}

		userId := c.Context().UserValue("userId").(string)

		now := time.Now().UnixMilli()

		_, err := rdb.HSet(c.Context(), "settings:"+hash(PEPPER_SETTINGS+userId), map[string]interface{}{
			"value":   c.Body(),
			"written": now,
		}).Result()

		if err != nil {
			panic(err)
		}

		return c.JSON(&fiber.Map{
			"written": now,
		})
	})

	app.Delete("/v1/settings", func(c *fiber.Ctx) error {
		userId := c.Context().UserValue("userId").(string)

		rdb.Del(c.Context(), "settings:"+hash(PEPPER_SETTINGS+userId))

		return c.SendStatus(204)
	})
	// #endregion

	// #region discord oauth
	app.Get("/v1/oauth/callback", func(c *fiber.Ctx) error {
		code := c.Query("code")

		if code == "" {
			return c.Status(400).JSON(&fiber.Map{
				"error": "Missing code",
			})
		}

		var accessTokenResult DiscordAccessTokenResult

		res, err := req.R().SetFormData(map[string]string{
			"client_id":     DISCORD_CLIENT_ID,
			"client_secret": DISCORD_CLIENT_SECRET,
			"grant_type":    "authorization_code",
			"code":          code,
			"redirect_uri":  DISCORD_REDIRECT_URI,
			"scope":         "identify",
		}).SetSuccessResult(&accessTokenResult).Post("https://discord.com/api/oauth2/token")

		if err != nil {
			return c.Status(500).JSON(&fiber.Map{
				"error": "Failed to request access token",
			})
		}

		if res.IsErrorState() {
			return c.Status(400).JSON(&fiber.Map{
				"error": "Invalid code",
			})
		}

		accessToken := accessTokenResult.AccessToken

		var userResult DiscordUserResult

		res, err = req.R().SetHeaders(map[string]string{
			"Authorization": "Bearer " + accessToken,
		}).SetSuccessResult(&userResult).Get("https://discord.com/api/users/@me")

		if err != nil {
			return c.Status(500).JSON(&fiber.Map{
				"error": "Failed to request user",
			})
		}

		if res.IsErrorState() {
			return c.Status(500).JSON(&fiber.Map{
				"error": "Failed to request user",
			})
		}

		userId := userResult.Id

		if ALLOWED_USERS != nil && !ALLOWED_USERS[userId] {
			return c.Status(403).JSON(&fiber.Map{
				"error": "User is not whitelisted",
			})
		}

		secret, err := rdb.Get(c.Context(), "secrets:"+hash(PEPPER_SECRETS+userId)).Result()

		if err == redis.Nil {
			key := make([]byte, 48)

			_, err := rand.Read(key)
			if err != nil {
				return c.Status(500).JSON(&fiber.Map{
					"error": "Failed to generate secret",
				})
			}

			secret = hex.EncodeToString(key)
			rdb.Set(c.Context(), "secrets:"+hash(PEPPER_SECRETS+userId), secret, 0)
		} else if err != nil {
			panic(err)
		}

		return c.JSON(&fiber.Map{
			"secret": secret,
		})
	})

	app.Get("/v1/oauth/settings", func(c *fiber.Ctx) error {
		return c.JSON(&fiber.Map{
			"clientId":    DISCORD_CLIENT_ID,
			"redirectUri": DISCORD_REDIRECT_URI,
		})
	})
	// #endregion

	// #region erase all
	app.Delete("/v1", requireAuth, func(c *fiber.Ctx) error {
		userId := c.Context().UserValue("userId").(string)

		rdb.Del(c.Context(), "settings:"+hash(PEPPER_SETTINGS+userId))
        rdb.Del(c.Context(), "secret"+hash(PEPPER_SECRETS+userId))

		return c.SendStatus(204)
	})
	// #endregion

	app.Get("/v1", func(c *fiber.Ctx) error {
		return c.JSON(&fiber.Map{
            "ping": "pong",
        })
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect(ROOT_REDIRECT, 303)
	})

	app.Listen(HOST + ":" + PORT)
}
