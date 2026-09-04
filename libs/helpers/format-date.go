package helpers

import (
	"fmt"
	"time"
)

var bulan = []string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

func FormatTanggalIndo(t time.Time) string {
	return fmt.Sprintf("%02d %s %d", t.Day(), bulan[int(t.Month())], t.Year())
}

func TimeFromInterface(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case *time.Time:
		if t != nil {
			return *t, true
		}
	case string:
		// coba beberapa layout yang umum
		layouts := []string{
			"2006-01-02 15:04:05.000 -0700 MST",
			"2006-01-02 15:04:05.000 -0700",
			"2006-01-02 15:04:05.000 -0700 -0700", // fallback
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05.000",
			"2006-01-02 15:04:05",
		}
		for _, l := range layouts {
			if parsed, err := time.Parse(l, t); err == nil {
				return parsed, true
			}
		}
	default:
		s := fmt.Sprintf("%v", v)
		layouts := []string{
			"2006-01-02 15:04:05.000 -0700 MST",
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05.000",
			"2006-01-02 15:04:05",
		}
		for _, l := range layouts {
			if parsed, err := time.Parse(l, s); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}
