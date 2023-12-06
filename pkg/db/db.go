package binmandb

import (
	"fmt"
	"strings"
)

// getBytes is a helper function to convert types to byte arrays as needed
func GetBytes(i interface{}) []byte {
	switch data := i.(type) {
	case string:
		return []byte(data)
	case int64:
		// I don't love this, but it should work for now
		return []byte(fmt.Sprintf("%d", data))
	}
	return nil
}

func ParseKey(key string) [][]byte {
	var r [][]byte

	for _, k := range strings.Split(key, "/") {
		r = append(r, GetBytes(k))
	}

	return r
}
