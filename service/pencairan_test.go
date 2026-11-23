package service

import (
	"testing"
	"time"
)

func TestHitungJadwal(t *testing.T) {
	cair := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		jumlah      int64
		tenor       int
		jasa        int64
		tanggalCair time.Time
		wantLen     int
		wantPokok   []int64
		wantJasa    int64
		wantTempo   []time.Time
		wantStatus  string
	}{
		{
			name:        "jumlah habis dibagi tenor",
			jumlah:      6000000,
			tenor:       12,
			jasa:        60000,
			tanggalCair: cair,
			wantLen:     12,
			wantPokok: []int64{
				500000, 500000, 500000, 500000, 500000, 500000,
				500000, 500000, 500000, 500000, 500000, 500000,
			},
			wantJasa: 60000,
			wantTempo: []time.Time{
				time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				time.Date(2027, time.January, 31, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
		{
			name:        "sisa pembulatan masuk angsuran terakhir",
			jumlah:      1000000,
			tenor:       3,
			jasa:        10000,
			tanggalCair: time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC),
			wantLen:     3,
			wantPokok:   []int64{333333, 333333, 333334},
			wantJasa:    10000,
			wantTempo: []time.Time{
				time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
		{
			name:        "tenor satu",
			jumlah:      750000,
			tenor:       1,
			jasa:        7500,
			tanggalCair: time.Date(2026, time.November, 30, 0, 0, 0, 0, time.UTC),
			wantLen:     1,
			wantPokok:   []int64{750000},
			wantJasa:    7500,
			wantTempo: []time.Time{
				time.Date(2026, time.December, 30, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
		{
			name:        "tanggal cair 31 Januari tahun kabisat",
			jumlah:      120000,
			tenor:       1,
			jasa:        1200,
			tanggalCair: time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC),
			wantLen:     1,
			wantPokok:   []int64{120000},
			wantJasa:    1200,
			wantTempo: []time.Time{
				time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := HitungJadwal(test.jumlah, test.tenor, test.jasa, test.tanggalCair)
			if len(got) != test.wantLen {
				t.Fatalf("jumlah baris = %d, ingin %d", len(got), test.wantLen)
			}

			var totalPokok int64
			for i, row := range got {
				totalPokok += row.Pokok

				if row.AngsuranKe != i+1 {
					t.Errorf("angsuran ke-%d = %d, ingin %d", i, row.AngsuranKe, i+1)
				}
				if row.Pokok != test.wantPokok[i] {
					t.Errorf("pokok baris %d = %d, ingin %d", i+1, row.Pokok, test.wantPokok[i])
				}
				if row.Jasa != test.wantJasa {
					t.Errorf("jasa baris %d = %d, ingin %d", i+1, row.Jasa, test.wantJasa)
				}
				if row.Status != test.wantStatus {
					t.Errorf("status baris %d = %q, ingin %q", i+1, row.Status, test.wantStatus)
				}
				if row.IDPinjaman != 0 {
					t.Errorf("IDPinjaman baris %d = %d, ingin 0", i+1, row.IDPinjaman)
				}
			}

			if totalPokok != test.jumlah {
				t.Errorf("total pokok = %d, ingin %d", totalPokok, test.jumlah)
			}

			for i, want := range test.wantTempo {
				index := i
				if len(test.wantTempo) == 2 && i == 1 {
					index = len(got) - 1
				}
				if !got[index].JatuhTempo.Equal(want) {
					t.Errorf("jatuh tempo baris %d = %v, ingin %v", index+1, got[index].JatuhTempo, want)
				}
			}
		})
	}
}

func TestHitungJadwalTenorTidakValid(t *testing.T) {
	for _, tenor := range []int{0, -1} {
		if got := HitungJadwal(100000, tenor, 1000, time.Now()); got != nil {
			t.Errorf("tenor %d menghasilkan %v, ingin nil", tenor, got)
		}
	}
}
