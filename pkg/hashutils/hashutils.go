package hashutils

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

func generateHash(data string) string {
	hash := sha256.New()
	hash.Write([]byte(data))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func GetCacheKey(startDate, endDate time.Time, textLexems []string) string {
	key := fmt.Sprintf("%s.%s", startDate, endDate)
	key = strings.Join(textLexems, ".")
	return generateHash(key)
}
