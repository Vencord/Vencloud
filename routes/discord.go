package routes

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"github.com/imroc/req/v3"
	"github.com/redis/go-redis/v9"

	g "github.com/vencord/backend/globals"
	"github.com/vencord/backend/util"
)

// /v1/oauth

type DiscordAccessTokenResult struct {
	AccessToken string `json:"access_token"`
}

type DiscordUserResult struct {
	Id string `json:"id"`
}

// /v1/oauth/callback
func GETOAuthCallback(c *fiber.Ctx) error {
	code := c.Query("code")

	if code == "" {
		return c.Status(400).JSON(&fiber.Map{
			"error": "Missing code",
		})
	}

	var accessTokenResult DiscordAccessTokenResult

	res, err := req.R().SetFormData(map[string]string{
		"client_id":     g.DISCORD_CLIENT_ID,
		"client_secret": g.DISCORD_CLIENT_SECRET,
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  g.DISCORD_REDIRECT_URI,
		"scope":         "identify",
	}).SetSuccessResult(&accessTokenResult).Post("https://discord.com/api/oauth2/token")

	if err != nil {
        c.Context().Logger().Printf("Failed to request access token: %v", err)
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
        c.Context().Logger().Printf("Failed to request user: %v", err)
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

	if g.ALLOWED_USERS != nil && !g.ALLOWED_USERS[userId] {
		return c.Status(403).JSON(&fiber.Map{
			"error": "User is not whitelisted",
		})
	}

	secret, err := g.RDB.Get(c.Context(), "secrets:"+util.Hash(g.PEPPER_SECRETS+userId)).Result()

	if err == redis.Nil {
		key := make([]byte, 48)

		_, err := rand.Read(key)
		if err != nil {
            c.Context().Logger().Printf("Failed to generate secret: %v", err)
			return c.Status(500).JSON(&fiber.Map{
				"error": "Failed to generate secret",
			})
		}

		secret = hex.EncodeToString(key)
		g.RDB.Set(c.Context(), "secrets:"+util.Hash(g.PEPPER_SECRETS+userId), secret, 0)
	} else if err != nil {
		panic(err)
	}

	return c.JSON(&fiber.Map{
		"secret": secret,
	})
}

// /v1/oauth/settings
func GETOAuthSettings(c *fiber.Ctx) error {
	return c.JSON(&fiber.Map{
		"clientId":    g.DISCORD_CLIENT_ID,
		"redirectUri": g.DISCORD_REDIRECT_URI,
	})
}
