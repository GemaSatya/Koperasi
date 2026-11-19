package util

import (
	"testing"
	"time"
)

func TestAddMonths(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		months   int
		expected time.Time
	}{
		{
			name:     "31 Januari ke Februari tahun biasa",
			input:    time.Date(2023, time.January, 31, 15, 30, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, time.February, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "31 Januari ke Februari tahun kabisat",
			input:    time.Date(2024, time.January, 31, 15, 30, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "lintas tahun",
			input:    time.Date(2024, time.November, 30, 10, 0, 0, 0, time.UTC),
			months:   2,
			expected: time.Date(2025, time.January, 30, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := AddMonths(test.input, test.months)
			if !actual.Equal(test.expected) {
				t.Fatalf("AddMonths() = %v, want %v", actual, test.expected)
			}
		})
	}
}

func TestHariTerlambat(t *testing.T) {
	dueDate := time.Date(2026, time.November, 19, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		paymentDate time.Time
		expected    int
	}{
		{
			name:        "tepat jatuh tempo",
			paymentDate: dueDate,
			expected:    0,
		},
		{
			name:        "terlambat satu hari",
			paymentDate: dueDate.AddDate(0, 0, 1),
			expected:    1,
		},
		{
			name:        "dibayar lebih awal",
			paymentDate: dueDate.AddDate(0, 0, -1),
			expected:    0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := HariTerlambat(dueDate, test.paymentDate)
			if actual != test.expected {
				t.Fatalf("HariTerlambat() = %d, want %d", actual, test.expected)
			}
		})
	}
}

func TestPersenDari(t *testing.T) {
	tests := []struct {
		name     string
		nominal  int64
		persen   float64
		expected int64
	}{
		{
			name:     "1 persen dari enam juta",
			nominal:  6000000,
			persen:   1,
			expected: 60000,
		},
		{
			name:     "dibulatkan ke bawah",
			nominal:  124,
			persen:   2,
			expected: 2,
		},
		{
			name:     "dibulatkan ke atas",
			nominal:  126,
			persen:   2,
			expected: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := PersenDari(test.nominal, test.persen)
			if actual != test.expected {
				t.Fatalf("PersenDari() = %d, want %d", actual, test.expected)
			}
		})
	}
}
