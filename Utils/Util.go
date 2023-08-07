package Utils

import(
	"fmt"
	"crypto/sha1"
)

func Hash(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}
