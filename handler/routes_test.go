package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GemaSatya/ProjekPemweb/config"
	"github.com/GemaSatya/ProjekPemweb/service"
)

func muxModulBaru() *http.ServeMux {
	mux := http.NewServeMux()
	cfg := config.Config{DendaPersenPerHari: 0.1}

	pinjamanSvc := service.NewPinjamanService(nil, nil, nil, nil, cfg)
	pembayaranSvc := service.NewPembayaranService(nil, nil, nil, nil, nil, cfg)
	pelunasanSvc := service.NewPelunasanService(nil, nil, nil, nil, cfg)
	laporanSvc := service.NewLaporanService(nil, nil, nil, nil, nil, nil, cfg.DendaPersenPerHari)

	NewPinjamanHandler(pinjamanSvc).RegisterRoutes(mux)
	NewPembayaranHandler(pembayaranSvc).RegisterRoutes(mux)
	NewPelunasanHandler(pelunasanSvc).RegisterRoutes(mux)
	NewLaporanHandler(laporanSvc).RegisterRoutes(mux)
	return mux
}

func TestSemuaRouteTerdaftar(t *testing.T) {
	mux := muxModulBaru()
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/anggota/1/batas-pinjaman"},
		{"POST", "/api/pinjaman"},
		{"GET", "/api/pinjaman"},
		{"GET", "/api/pinjaman/1"},
		{"POST", "/api/pinjaman/1/cairkan"},
		{"GET", "/api/pinjaman/1/jadwal"},
		{"GET", "/api/jadwal/1/tagihan"},
		{"POST", "/api/jadwal/1/bayar"},
		{"GET", "/api/pinjaman/1/pembayaran"},
		{"GET", "/api/pinjaman/1/pelunasan/hitung"},
		{"POST", "/api/pinjaman/1/pelunasan"},
		{"GET", "/api/laporan/simpanan"},
		{"GET", "/api/laporan/kartu-pinjaman/1"},
		{"GET", "/api/laporan/riwayat-pembayaran"},
		{"GET", "/api/laporan/tunggakan"},
		{"GET", "/api/laporan/pendapatan-jasa"},
		{"GET", "/api/laporan/pendapatan-denda"},
	}

	for _, route := range routes {
		request := httptest.NewRequest(route.method, route.path, nil)
		_, pattern := mux.Handler(request)
		if pattern == "" {
			t.Errorf("route %s %s tidak terdaftar", route.method, route.path)
		}
	}
}

func TestSemuaHandlerBisaDidaftarkan(t *testing.T) {
	mux := http.NewServeMux()
	cfg := config.Config{DendaPersenPerHari: 0.1}

	NewAuthHandler(service.NewAuthService(nil)).RegisterRoutes(mux)
	NewAnggotaHandler(service.NewAnggotaService(nil)).RegisterRoutes(mux)
	NewSimpananHandler(service.NewSimpananService(nil, nil, nil)).RegisterRoutes(mux)
	NewPinjamanHandler(service.NewPinjamanService(nil, nil, nil, nil, cfg)).RegisterRoutes(mux)
	NewPembayaranHandler(service.NewPembayaranService(nil, nil, nil, nil, nil, cfg)).RegisterRoutes(mux)
	NewPelunasanHandler(service.NewPelunasanService(nil, nil, nil, nil, cfg)).RegisterRoutes(mux)
	NewLaporanHandler(service.NewLaporanService(nil, nil, nil, nil, nil, nil, cfg.DendaPersenPerHari)).RegisterRoutes(mux)

	if mux == nil {
		t.Fatal("mux tidak valid")
	}
}

func TestValidasiTanpaDatabase(t *testing.T) {
	mux := muxModulBaru()
	tests := []struct {
		name string
		path string
	}{
		{name: "id anggota bukan angka", path: "/api/anggota/abc/batas-pinjaman"},
		{name: "id pinjaman bukan angka", path: "/api/laporan/kartu-pinjaman/abc"},
		{name: "tanggal tunggakan salah", path: "/api/laporan/tunggakan?tanggal=bad"},
		{name: "periode pendapatan jasa kosong", path: "/api/laporan/pendapatan-jasa"},
		{name: "periode pendapatan denda kosong", path: "/api/laporan/pendapatan-denda"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body: %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), `"success":false`) {
				t.Fatalf("response bukan error: %s", recorder.Body.String())
			}
		})
	}
}
