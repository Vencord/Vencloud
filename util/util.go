package util

import (
	"crypto/sha1"
	"fmt"
)

func Hash(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}
