package globals

import (
	"os"

	"github.com/redis/go-redis/v9"
)

// environment variables
var (
	HOST = os.Getenv("HOST")
	PORT = os.Getenv("PORT")

	REDIS_URI = os.Getenv("REDIS_URI")

	ROOT_REDIRECT = os.Getenv("ROOT_REDIRECT")

	DISCORD_CLIENT_ID     = os.Getenv("DISCORD_CLIENT_ID")
	DISCORD_CLIENT_SECRET = os.Getenv("DISCORD_CLIENT_SECRET")
	DISCORD_REDIRECT_URI  = os.Getenv("DISCORD_REDIRECT_URI")

	PEPPER_SETTINGS = os.Getenv("PEPPER_SETTINGS")
	PEPPER_SECRETS  = os.Getenv("PEPPER_SECRETS")

	SIZE_LIMIT int // initialised in main

	ALLOWED_USERS map[string]bool // initialised in main
)

// other app globals, initialised in main
var (
	// redis client
	RDB *redis.Client
)
