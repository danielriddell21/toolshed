package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseDims(s string) (width, height int, err error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad size %q (want WxH, e.g. 31x21)", s)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || w <= 0 {
		return 0, 0, fmt.Errorf("bad width in %q", s)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || h <= 0 {
		return 0, 0, fmt.Errorf("bad height in %q", s)
	}
	return w, h, nil
}

func DefaultSeed(seed int64) int64 {
	if seed != 0 {
		return seed
	}
	return time.Now().UnixNano()
}
