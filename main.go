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

func hash(s string) string {
    return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

func main() {
    // environment
    HOST := os.Getenv("HOST")
    PORT := os.Getenv("PORT")
    REDIS_URI := os.Getenv("REDIS_URI")

    DISCORD_CLIENT_ID := os.Getenv("DISCORD_CLIENT_ID")
    DISCORD_CLIENT_SECRET := os.Getenv("DISCORD_CLIENT_SECRET")
    DISCORD_REDIRECT_URI := os.Getenv("DISCORD_REDIRECT_URI")

    PEPPER_SECRETS := os.Getenv("PEPPER_SECRETS")
    PEPPER_SETTINGS := os.Getenv("PEPPER_SETTINGS")

    slRaw, _ := strconv.ParseInt(os.Getenv("SIZE_LIMIT"), 10, 0)
    SIZE_LIMIT := int(slRaw)

    app := fiber.New()
    rdb := redis.NewClient(&redis.Options{
        Addr:     REDIS_URI,
    })
    req := reqHttp.C()

    app.Use(cors.New(cors.Config{
        ExposeHeaders: "ETag",
    }))
    app.Use(logger.New())

    // #region settings
    app.All("/settings", func(c *fiber.Ctx) error {
        authToken := c.Get("Authorization")

        if authToken == "" {
            return c.Status(401).JSON(&fiber.Map{
                "error": "Missing authorization",
            })
        }

        // decode base64 token and split by :
        // token[0] = username
        // token[1] = password
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

        userId := tokenSplit[0]
        secret := tokenSplit[1]

        storedSecret, err := rdb.Get(c.Context(), "secrets:" + hash(PEPPER_SECRETS + userId)).Result()

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
    })

    app.Head("/settings", func(c *fiber.Ctx) error {
        userId := c.Context().UserValue("userId").(string)

        written, err := rdb.HGet(c.Context(), "settings:" + hash(PEPPER_SETTINGS + userId), "written").Result()

        if err == redis.Nil {
            return c.Status(404).Send(nil)
        } else if err != nil {
            panic(err)
        }

        c.Set("ETag", written)
        return c.SendStatus(204)
    })

    app.Get("/settings", func(c *fiber.Ctx) error {
        userId := c.Context().UserValue("userId").(string)

        settings, err := rdb.HMGet(c.Context(), "settings:" + hash(PEPPER_SETTINGS + userId), "value", "written").Result()

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

    app.Put("/settings", func(c *fiber.Ctx) error {
        if (c.Get("Content-Type") != "application/octet-stream") {
            return c.Status(415).JSON(&fiber.Map{
                "error": "Content type must be application/octet-stream",
            })
        }

        if (len(c.Body()) > SIZE_LIMIT) {
            return c.Status(413).JSON(&fiber.Map{
                "error": "Settings are too large",
            })
        }

        userId := c.Context().UserValue("userId").(string)

        now := time.Now().UnixMilli()

        _, err := rdb.HSet(c.Context(), "settings:" + hash(PEPPER_SETTINGS + userId), map[string]interface{}{
            "value": c.Body(),
            "written": now,
        }).Result()

        if err != nil {
            panic(err)
        }

        return c.JSON(&fiber.Map{
            "written": now,
        })
    })

    app.Delete("/settings", func(c *fiber.Ctx) error {
        userId := c.Context().UserValue("userId").(string)

        rdb.Del(c.Context(), "settings:" + hash(PEPPER_SETTINGS + userId))

        return c.SendStatus(204)
    })
    // #endregion

    // #region discord oauth
    app.Get("/callback", func(c *fiber.Ctx) error {
        code := c.Query("code")

        if code == "" {
            return c.Status(400).JSON(&fiber.Map{
                "error": "Missing code",
            })
        }

        var accessTokenResult DiscordAccessTokenResult

        res, err := req.R().SetFormData(map[string]string{
            "client_id": DISCORD_CLIENT_ID,
            "client_secret": DISCORD_CLIENT_SECRET,
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": DISCORD_REDIRECT_URI,
            "scope": "identify",
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

        secret, err := rdb.Get(c.Context(), "secrets:" + hash(PEPPER_SECRETS + userId)).Result()

        if err == redis.Nil {
            key := make([]byte, 48)

            _, err := rand.Read(key)
            if err != nil {
                return c.Status(500).JSON(&fiber.Map{
                    "error": "Failed to generate secret",
                })
            }

            secret = hex.EncodeToString(key)
            rdb.Set(c.Context(), "secrets:" + hash(PEPPER_SECRETS + userId), secret, 0)
        }

        return c.JSON(&fiber.Map{
            "secret": secret,
        })
    })
    // #endregion

    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendFile("static/index.html")
    })

    app.Listen(HOST + ":" + PORT)
}
