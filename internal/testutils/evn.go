package testutils

import (
	"os"
	"strconv"
)

func GetIntEvn(env string, defaultVal int) (val int) {
	if s := os.Getenv(env); s != "" {
		if i, err := strconv.ParseInt(s, 10, 32); err == nil {
			return int(i)
		}
	}
	return defaultVal
}
