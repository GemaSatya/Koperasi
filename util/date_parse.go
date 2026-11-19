package util

import "time"

func ParseTanggal(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, time.Local)
}
