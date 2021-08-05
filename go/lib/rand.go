package lib

import (
	"math/rand"
	"time"
)

func RandomLowercaseString(len int) string {
	rand.Seed(time.Now().UnixNano())
	bytes := make([]byte, len)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = b%26 + 97
	}
	return string(bytes)
}
