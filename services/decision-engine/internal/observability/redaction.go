package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const Redacted = "[redacted]"

func HashForLog(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func LenForLog(value string) int {
	return len([]rune(value))
}
