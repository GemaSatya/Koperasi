package service

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
)

type fakeLaporanRepo struct {
	simpanan  []models.BarisSimpanan
	riwayat   []models.BarisPembayaran
	tunggakan []models.Tunggakan
	jasa      map[string]int64
	denda     map[string]int64
	err       error
}

func (f *fakeLaporanRepo) LaporanSimpanan(models.FilterLaporan) ([]models.BarisSimpanan, error) {
	return f.simpanan, f.err
}

func (f *fakeLaporanRepo) RiwayatPembayaran(models.FilterLaporan) ([]models.BarisPembayaran, error) {
	return f.riwayat, f.err
}

func (f *fakeLaporanRepo) DaftarTunggakan(time.Time) ([]models.Tunggakan, error) {
	return f.tunggakan, f.err
}

func (f *fakeLaporanRepo) JasaPerBulan(time.Time, time.Time) (map[string]int64, error) {
	return f.jasa, f.err
}

func (f *fakeLaporanRepo) DendaPerBulan(time.Time, time.Time) (map[string]int64, error) {
	return f.denda, f.err
}

func newLaporanServiceUntukTes(repo *fakeLaporanRepo) *LaporanService {
	return &LaporanService{repo: repo, persenDenda: 0.1}
}

func TestValidasiLaporan(t *testing.T) {
	dari := time.Date(2026, time.December, 2, 0, 0, 0, 0, time.UTC)
	sampai := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeLaporanRepo{}
	svc := newLaporanServiceUntukTes(repo)

	if _, err := svc.LaporanSimpanan(models.FilterLaporan{Dari: dari, Sampai: sampai}); !errors.Is(err, ErrPeriodeTidakValid) {
		t.Errorf("LaporanSimpanan error = %v, ingin %v", err, ErrPeriodeTidakValid)
	}
	if _, err := svc.RiwayatPembayaran(models.FilterLaporan{IDAnggota: -1}); !errors.Is(err, ErrInputAnggota) {
		t.Errorf("RiwayatPembayaran error = %v, ingin %v", err, ErrInputAnggota)
	}
	if _, err := svc.PendapatanJasa(time.Time{}, sampai); !errors.Is(err, ErrPeriodeTidakValid) {
		t.Errorf("PendapatanJasa error = %v, ingin %v", err, ErrPeriodeTidakValid)
	}
	if _, err := svc.PendapatanDenda(dari, sampai); !errors.Is(err, ErrPeriodeTidakValid) {
		t.Errorf("PendapatanDenda error = %v, ingin %v", err, ErrPeriodeTidakValid)
	}
}

func TestDaftarTunggakanMenghitungDendaBerjalan(t *testing.T) {
	jatuhTempo := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	acuan := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeLaporanRepo{tunggakan: []models.Tunggakan{
		{IDJadwal: 1, JatuhTempo: jatuhTempo, SisaPokok: 500000, SisaJasa: 10000, DendaDibayar: 1000},
		{IDJadwal: 2, JatuhTempo: jatuhTempo, SisaPokok: 100000, DendaDibayar: 20000},
	}}
	svc := newLaporanServiceUntukTes(repo)

	got, err := svc.DaftarTunggakan(acuan)
	if err != nil {
		t.Fatalf("DaftarTunggakan error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("jumlah baris = %d, ingin 2", len(got))
	}
	if got[0].HariTerlambat != 30 || got[0].DendaBerjalan != 14300 {
		t.Errorf("baris pertama = %+v, ingin hari 30 denda 14300", got[0])
	}
	if got[1].DendaBerjalan != 0 {
		t.Errorf("denda baris kedua = %d, ingin 0 (tidak negatif)", got[1].DendaBerjalan)
	}
}

func TestPendapatanRincianPerBulan(t *testing.T) {
	repo := &fakeLaporanRepo{
		jasa:  map[string]int64{"2026-11": 12000, "2026-12": 5000},
		denda: map[string]int64{"2026-12": 3000},
	}
	svc := newLaporanServiceUntukTes(repo)
	dari := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	sampai := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)

	jasa, err := svc.PendapatanJasa(dari, sampai)
	if err != nil {
		t.Fatalf("PendapatanJasa error = %v", err)
	}
	if jasa.Total != 17000 || !reflect.DeepEqual(jasa.Rincian, repo.jasa) {
		t.Errorf("pendapatan jasa = %+v", jasa)
	}

	denda, err := svc.PendapatanDenda(dari, sampai)
	if err != nil {
		t.Fatalf("PendapatanDenda error = %v", err)
	}
	if denda.Total != 3000 || !reflect.DeepEqual(denda.Rincian, repo.denda) {
		t.Errorf("pendapatan denda = %+v", denda)
	}
}

func TestKonstruktorLaporanMenyimpanPersenDenda(t *testing.T) {
	repo := &fakeLaporanRepo{}
	svc := NewLaporanService(repo, config.Config{DendaPersenPerHari: 0.25})
	if svc.persenDenda != 0.25 {
		t.Errorf("persen denda = %v, ingin 0.25", svc.persenDenda)
	}
}
