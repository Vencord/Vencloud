package util

import (
	"crypto/sha256"
	"fmt"
)

func Hash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}
