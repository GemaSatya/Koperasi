package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSemuaRouteTerdaftar(t *testing.T) {
	mux := http.NewServeMux()
	NewAuthHandler(nil).RegisterRoutes(mux)
	NewAnggotaHandler(nil).RegisterRoutes(mux)
	NewSimpananHandler(nil).RegisterRoutes(mux)
	NewPinjamanHandler(nil, nil).RegisterRoutes(mux)
	NewPelunasanHandler(nil).RegisterRoutes(mux)
	NewLaporanKartuHandler(nil).RegisterRoutes(mux)
	NewPembayaranHandler(nil).RegisterRoutes(mux)
	NewLaporanHandler(nil).RegisterRoutes(mux)

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/login"},
		{"GET", "/api/anggota"},
		{"POST", "/api/simpanan"},
		{"GET", "/api/pinjaman"},
		{"POST", "/api/pinjaman/1/cairkan"},
		{"GET", "/api/pinjaman/1/pelunasan/hitung"},
		{"GET", "/api/laporan/kartu-pinjaman/1"},
		{"GET", "/api/pinjaman/1/jadwal"},
		{"GET", "/api/jadwal/1/tagihan"},
		{"POST", "/api/jadwal/1/bayar"},
		{"GET", "/api/pinjaman/1/pembayaran"},
		{"GET", "/api/laporan/simpanan"},
		{"GET", "/api/laporan/riwayat-pembayaran"},
		{"GET", "/api/laporan/tunggakan"},
		{"GET", "/api/laporan/pendapatan-jasa"},
		{"GET", "/api/laporan/pendapatan-denda"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			_, pattern := mux.Handler(req)
			if pattern == "" {
				t.Errorf("route %s %s tidak terdaftar", route.method, route.path)
			}
		})
	}
}

func TestValidasiHandlerTanpaDatabase(t *testing.T) {
	mux := http.NewServeMux()
	NewPembayaranHandler(nil).RegisterRoutes(mux)
	NewLaporanHandler(nil).RegisterRoutes(mux)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   string
	}{
		{
			name:   "id jadwal bukan angka",
			method: "POST",
			path:   "/api/jadwal/abc/bayar",
			body:   `{"tanggal_bayar":"2026-11-20","nominal":100000}`,
			want:   "id jadwal tidak valid",
		},
		{
			name:   "tanggal bayar salah format",
			method: "POST",
			path:   "/api/jadwal/5/bayar",
			body:   `{"tanggal_bayar":"20-11-2026","nominal":100000}`,
			want:   "tanggal_bayar harus berformat YYYY-MM-DD",
		},
		{
			name:   "tagihan tanggal salah format",
			method: "GET",
			path:   "/api/jadwal/5/tagihan?tanggal=20-11-2026",
			want:   "tanggal tidak valid",
		},
		{
			name:   "pendapatan jasa tanpa periode",
			method: "GET",
			path:   "/api/laporan/pendapatan-jasa",
			want:   "periode laporan tidak valid",
		},
		{
			name:   "pendapatan denda periode terbalik",
			method: "GET",
			path:   "/api/laporan/pendapatan-denda?dari=2026-12-02&sampai=2026-12-01",
			want:   "periode laporan tidak valid",
		},
		{
			name:   "id anggota bukan angka",
			method: "GET",
			path:   "/api/laporan/simpanan?id_anggota=abc",
			want:   "data anggota tidak valid",
		},
		{
			name:   "tanggal tunggakan salah format",
			method: "GET",
			path:   "/api/laporan/tunggakan?tanggal=20-11-2026",
			want:   "tanggal tidak valid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status = %d, ingin 400; body = %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), test.want) {
				t.Errorf("body = %s, ingin mengandung %q", recorder.Body.String(), test.want)
			}
			if !strings.Contains(recorder.Body.String(), `"success":false`) {
				t.Errorf("body tidak bertanda success:false: %s", recorder.Body.String())
			}
		})
	}
}
