package service

import (
	"errors"
	"testing"
)

func TestHitungDenda(t *testing.T) {
	const persen = 0.1
	tests := []struct {
		name        string
		sisaTagihan int64
		hari        int
		expected    int64
	}{
		{name: "tepat jatuh tempo", sisaTagihan: 295000, hari: 0, expected: 0},
		{name: "terlambat satu hari", sisaTagihan: 295000, hari: 1, expected: 295},
		{name: "terlambat tiga puluh hari", sisaTagihan: 295000, hari: 30, expected: 8850},
		{name: "tidak ada tagihan", sisaTagihan: 0, hari: 10, expected: 0},
		{name: "dibayar lebih awal", sisaTagihan: 295000, hari: -1, expected: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := HitungDenda(test.sisaTagihan, test.hari, persen)
			if actual != test.expected {
				t.Fatalf("HitungDenda() = %d, want %d", actual, test.expected)
			}
		})
	}
}

func TestAlokasiPembayaran(t *testing.T) {
	tests := []struct {
		name      string
		nominal   int64
		dendaSisa int64
		jasaSisa  int64
		pokokSisa int64
		wantDenda int64
		wantJasa  int64
		wantPokok int64
		wantErr   error
	}{
		{
			name: "bayar kurang dari denda", nominal: 3000,
			dendaSisa: 5000, jasaSisa: 60000, pokokSisa: 235000,
			wantDenda: 3000,
		},
		{
			name: "pas denda ditambah jasa", nominal: 65000,
			dendaSisa: 5000, jasaSisa: 60000, pokokSisa: 235000,
			wantDenda: 5000, wantJasa: 60000,
		},
		{
			name: "pas lunas", nominal: 300000,
			dendaSisa: 5000, jasaSisa: 60000, pokokSisa: 235000,
			wantDenda: 5000, wantJasa: 60000, wantPokok: 235000,
		},
		{
			name: "nominal kosong", nominal: 0,
			dendaSisa: 5000, jasaSisa: 60000, pokokSisa: 235000,
			wantErr: ErrNominalTidakValid,
		},
		{
			name: "kelebihan pembayaran", nominal: 300001,
			dendaSisa: 5000, jasaSisa: 60000, pokokSisa: 235000,
			wantErr: ErrNominalMelebihiTagihan,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			denda, jasa, pokok, err := AlokasiPembayaran(
				test.nominal, test.dendaSisa, test.jasaSisa, test.pokokSisa,
			)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if denda != test.wantDenda || jasa != test.wantJasa || pokok != test.wantPokok {
				t.Fatalf("alokasi = (%d, %d, %d), want (%d, %d, %d)",
					denda, jasa, pokok, test.wantDenda, test.wantJasa, test.wantPokok)
			}
		})
	}
}
