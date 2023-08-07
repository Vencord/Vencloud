package Handlers

import(
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	reqHttp "github.com/imroc/req/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/Vencord/Backend/Utils"
)

var ALLOWED_USERS map[string]bool

var rdb *redis.Client
var (
    accountsRegistered = promauto.NewGaugeFunc(prometheus.GaugeOpts{
        Name: "vencord_accounts_registered",
        Help: "The total number of accounts registered",
    }, func() float64 {
        iter := rdb.Scan(context.Background(), 0, "secrets:*", 0).Iterator()
		var count int64

		for iter.Next(context.Background()) {
			count++
		}

		if err := iter.Err(); err != nil {
			panic(err)
		}

		return float64(count)
    })
)

var req *reqHttp.Client
var REDIS_URI string
var ROOT_REDIRECT string
var DISCORD_CLIENT_ID string
var DISCORD_CLIENT_SECRET string
var DISCORD_REDIRECT_URI string
var PEPPER_SECRETS string
var PEPPER_SETTINGS string
var slRaw int64
var SIZE_LIMIT int64

func init() {
	req = reqHttp.C()
    REDIS_URI = os.Getenv("REDIS_URI")
    ROOT_REDIRECT = os.Getenv("ROOT_REDIRECT")

    DISCORD_CLIENT_ID = os.Getenv("DISCORD_CLIENT_ID")
    DISCORD_CLIENT_SECRET = os.Getenv("DISCORD_CLIENT_SECRET")
    DISCORD_REDIRECT_URI = os.Getenv("DISCORD_REDIRECT_URI")

    PEPPER_SECRETS = os.Getenv("PEPPER_SECRETS")
    PEPPER_SETTINGS = os.Getenv("PEPPER_SETTINGS")

    slRaw, _ = strconv.ParseInt(os.Getenv("SIZE_LIMIT"), 10, 64)
    SIZE_LIMIT = slRaw
}

func RequireAuth(c *fiber.Ctx) error {
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

	storedSecret, err := rdb.Get(c.Context(), "secrets:"+Utils.Hash(os.Getenv("PEPPER_SECRETS")+userId)).Result()

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

func V1SettingsHead(c *fiber.Ctx) error {
	userId := c.Context().UserValue("userId").(string)

	written, err := rdb.HGet(c.Context(), "settings:"+Utils.Hash(PEPPER_SETTINGS+userId), "written").Result()

	if err == redis.Nil {
		return c.Status(404).Send(nil)
	} else if err != nil {
		panic(err)
	}

	c.Set("ETag", written)
	return c.SendStatus(204)
}

func V1SettingsGet(c* fiber.Ctx) error {
		userId := c.Context().UserValue("userId").(string)

		settings, err := rdb.HMGet(c.Context(), "settings:"+Utils.Hash(PEPPER_SETTINGS+userId), "value", "written").Result()

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

func V1SettingsPut(c *fiber.Ctx) error {
	if c.Get("Content-Type") != "application/octet-stream" {
		return c.Status(415).JSON(&fiber.Map{
			"error": "Content type must be application/octet-stream",
		})
	}

	bodyLength := int64(len(c.Body())) // Convert length to int64

	if bodyLength > SIZE_LIMIT {
		return c.Status(413).JSON(&fiber.Map{
			"error": "Settings are too large",
		})
	}

	userId := c.Context().UserValue("userId").(string)

	now := time.Now().UnixMilli()

	_, err := rdb.HSet(c.Context(), "settings:"+Utils.Hash(PEPPER_SETTINGS+userId), map[string]interface{}{
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

func V1SettingsDelete(c *fiber.Ctx) error {
	userId := c.Context().UserValue("userId").(string)

	rdb.Del(c.Context(), "settings:"+Utils.Hash(PEPPER_SETTINGS+userId))

	return c.SendStatus(204)
}

func V1OauthCallback(c *fiber.Ctx) error {
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

	secret, err := rdb.Get(c.Context(), "secrets:"+Utils.Hash(PEPPER_SECRETS+userId)).Result()

	if err == redis.Nil {
		key := make([]byte, 48)

		_, err := rand.Read(key)
		if err != nil {
			return c.Status(500).JSON(&fiber.Map{
				"error": "Failed to generate secret",
			})
		}

		secret = hex.EncodeToString(key)
		rdb.Set(c.Context(), "secrets:"+Utils.Hash(PEPPER_SECRETS+userId), secret, 0)
	} else if err != nil {
		panic(err)
	}

	return c.JSON(&fiber.Map{
		"secret": secret,
	})
}

func V1OauthSettings(c *fiber.Ctx) error {
	return c.JSON(&fiber.Map{
		"clientId":    DISCORD_CLIENT_ID,
		"redirectUri": DISCORD_REDIRECT_URI,
	})
}

func V1Delete(c *fiber.Ctx) error {
	userId := c.Context().UserValue("userId").(string)

	rdb.Del(c.Context(), "settings:"+Utils.Hash(PEPPER_SETTINGS+userId))
	rdb.Del(c.Context(), "secrets:"+Utils.Hash(PEPPER_SECRETS+userId))

	return c.SendStatus(204)
}

func V1Get(c *fiber.Ctx) error {
	return c.JSON(&fiber.Map{
		"ping": "pong",
	})
}

func Root(c *fiber.Ctx) error {
	return c.Redirect(ROOT_REDIRECT, 303)
}