package service

import (
	"time"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type LaporanRepository interface {
	LaporanSimpanan(f models.FilterLaporan) ([]models.BarisSimpanan, error)
	RiwayatPembayaran(f models.FilterLaporan) ([]models.BarisPembayaran, error)
	DaftarTunggakan(tanggalAcuan time.Time) ([]models.Tunggakan, error)
	JasaPerBulan(dari, sampai time.Time) (map[string]int64, error)
	DendaPerBulan(dari, sampai time.Time) (map[string]int64, error)
}

type LaporanService struct {
	repo        LaporanRepository
	persenDenda float64
}

func NewLaporanService(repo LaporanRepository, cfg config.Config) *LaporanService {
	return &LaporanService{repo: repo, persenDenda: cfg.DendaPersenPerHari}
}

func (s *LaporanService) LaporanSimpanan(f models.FilterLaporan) ([]models.BarisSimpanan, error) {
	if err := validasiFilterLaporan(f); err != nil {
		return nil, err
	}
	return s.repo.LaporanSimpanan(f)
}

func (s *LaporanService) RiwayatPembayaran(f models.FilterLaporan) ([]models.BarisPembayaran, error) {
	if err := validasiFilterLaporan(f); err != nil {
		return nil, err
	}
	return s.repo.RiwayatPembayaran(f)
}

func (s *LaporanService) DaftarTunggakan(tanggalAcuan time.Time) ([]models.Tunggakan, error) {
	if tanggalAcuan.IsZero() {
		tanggalAcuan = time.Now()
	}
	items, err := s.repo.DaftarTunggakan(tanggalAcuan)
	if err != nil {
		return nil, err
	}
	for i := range items {
		hari := util.HariTerlambat(items[i].JatuhTempo, tanggalAcuan)
		denda := HitungDenda(items[i].SisaPokok+items[i].SisaJasa, hari, s.persenDenda)
		// Denda yang sudah dibayar tidak boleh dihitung dua kali.
		items[i].HariTerlambat = hari
		items[i].DendaBerjalan = max64(0, denda-items[i].DendaDibayar)
	}
	return items, nil
}

func (s *LaporanService) PendapatanJasa(dari, sampai time.Time) (models.PendapatanJasa, error) {
	if err := validasiPeriodeLaporan(dari, sampai); err != nil {
		return models.PendapatanJasa{}, err
	}
	rincian, err := s.repo.JasaPerBulan(dari, sampai)
	if err != nil {
		return models.PendapatanJasa{}, err
	}
	return models.PendapatanJasa{Total: jumlahRincian(rincian), Rincian: rincian}, nil
}

func (s *LaporanService) PendapatanDenda(dari, sampai time.Time) (models.PendapatanDenda, error) {
	if err := validasiPeriodeLaporan(dari, sampai); err != nil {
		return models.PendapatanDenda{}, err
	}
	rincian, err := s.repo.DendaPerBulan(dari, sampai)
	if err != nil {
		return models.PendapatanDenda{}, err
	}
	return models.PendapatanDenda{Total: jumlahRincian(rincian), Rincian: rincian}, nil
}

func validasiFilterLaporan(f models.FilterLaporan) error {
	if f.IDAnggota < 0 {
		return ErrInputAnggota
	}
	if !f.Dari.IsZero() && !f.Sampai.IsZero() && f.Dari.After(f.Sampai) {
		return ErrPeriodeTidakValid
	}
	return nil
}

func validasiPeriodeLaporan(dari, sampai time.Time) error {
	if dari.IsZero() || sampai.IsZero() {
		return ErrPeriodeTidakValid
	}
	if dari.After(sampai) {
		return ErrPeriodeTidakValid
	}
	return nil
}

func jumlahRincian(rincian map[string]int64) int64 {
	var total int64
	for _, nominal := range rincian {
		total += nominal
	}
	return total
}
