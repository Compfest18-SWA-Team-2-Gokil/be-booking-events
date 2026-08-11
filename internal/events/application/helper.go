package application

import (
	"fmt"
	"time"
)

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("format tanggal tidak valid, gunakan RFC3339 (contoh: 2026-12-31T19:00:00+07:00)")
	}
	return t, nil
}
