package util

import (
	"math"
	"time"
)

func AddMonths(t time.Time, months int) time.Time {
	year, month, day := t.Date()
	target := time.Date(year, month+time.Month(months), 1, 0, 0, 0, 0, t.Location())
	lastDay := time.Date(
		target.Year(),
		target.Month()+1,
		0,
		0,
		0,
		0,
		0,
		t.Location(),
	).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(
		target.Year(),
		target.Month(),
		day,
		0,
		0,
		0,
		0,
		t.Location(),
	)
}

func HariTerlambat(jatuhTempo, tanggalBayar time.Time) int {
	dueDate := dateAtUTC(jatuhTempo)
	paymentDate := dateAtUTC(tanggalBayar)
	days := int(paymentDate.Sub(dueDate).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func PersenDari(nominal int64, persen float64) int64 {
	return int64(math.Round(float64(nominal) * persen / 100))
}

func dateAtUTC(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
