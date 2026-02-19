package binmandb

import (
	"fmt"
	"strings"
)

// getBytes is a helper function to convert types to byte arrays as needed
func GetBytes(i any) []byte {
	switch data := i.(type) {
	case string:
		return []byte(data)
	case int64:
		// I don't love this, but it should work for now
		return fmt.Appendf(nil, "%d", data)
	}
	return nil
}

func ParseKey(key string) [][]byte {
	var r [][]byte

	for k := range strings.SplitSeq(key, "/") {
		r = append(r, GetBytes(k))
	}

	return r
}
