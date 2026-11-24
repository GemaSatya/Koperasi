package service

import "testing"

func TestHitungRincianPelunasan(t *testing.T) {
	tests := []struct {
		name             string
		sisaPokok        int64
		jasaBulanIni     int64
		persenAdmin      float64
		wantAdministrasi int64
		wantTotal        int64
	}{
		{
			name: "belum ada pembayaran", sisaPokok: 1000000, jasaBulanIni: 10000,
			persenAdmin: 2, wantAdministrasi: 20000, wantTotal: 1030000,
		},
		{
			name: "administrasi dibulatkan ke bawah", sisaPokok: 123, jasaBulanIni: 0,
			persenAdmin: 2, wantAdministrasi: 2, wantTotal: 125,
		},
		{
			name: "administrasi dibulatkan ke atas", sisaPokok: 126, jasaBulanIni: 0,
			persenAdmin: 2, wantAdministrasi: 3, wantTotal: 129,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rincian := HitungRincianPelunasan(test.sisaPokok, test.jasaBulanIni, test.persenAdmin)
			if rincian.Administrasi != test.wantAdministrasi {
				t.Errorf("administrasi = %d, want %d", rincian.Administrasi, test.wantAdministrasi)
			}
			if rincian.Total != test.wantTotal {
				t.Errorf("total = %d, want %d", rincian.Total, test.wantTotal)
			}
			if rincian.SisaPokok != test.sisaPokok || rincian.Jasa != test.jasaBulanIni {
				t.Errorf("rincian pokok/jasa tidak sesuai: %+v", rincian)
			}
		})
	}
}
