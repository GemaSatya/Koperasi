package service

import (
	"database/sql"
	"errors"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

type LaporanService struct {
	db             *sql.DB
	laporanRepo    *repository.LaporanRepository
	pinjamanRepo   *repository.PinjamanRepository
	jadwalRepo     *repository.JadwalRepository
	pembayaranRepo *repository.PembayaranRepository
	pelunasanRepo  *repository.PelunasanRepository
	dendaPersen    float64
}

func NewLaporanService(
	db *sql.DB,
	laporanRepo *repository.LaporanRepository,
	pinjamanRepo *repository.PinjamanRepository,
	jadwalRepo *repository.JadwalRepository,
	pembayaranRepo *repository.PembayaranRepository,
	pelunasanRepo *repository.PelunasanRepository,
	dendaPersen float64,
) *LaporanService {
	return &LaporanService{
		db:             db,
		laporanRepo:    laporanRepo,
		pinjamanRepo:   pinjamanRepo,
		jadwalRepo:     jadwalRepo,
		pembayaranRepo: pembayaranRepo,
		pelunasanRepo:  pelunasanRepo,
		dendaPersen:    dendaPersen,
	}
}

func validasiFilter(f models.FilterLaporan, periodeWajib bool) error {
	if f.IDAnggota < 0 {
		return ErrInputAnggota
	}
	dariKosong := f.Dari.IsZero()
	sampaiKosong := f.Sampai.IsZero()

	if dariKosong && sampaiKosong {
		if periodeWajib {
			return ErrPeriodeTidakValid
		}
		return nil
	}
	if dariKosong || sampaiKosong || f.Dari.After(f.Sampai) {
		return ErrPeriodeTidakValid
	}
	return nil
}

func (s *LaporanService) LaporanSimpanan(f models.FilterLaporan) ([]models.BarisSimpanan, error) {
	if err := validasiFilter(f, false); err != nil {
		return nil, err
	}
	return s.laporanRepo.LaporanSimpanan(f)
}

// KartuPinjaman menyajikan pinjaman, jadwal, pembayaran, pelunasan, dan sisa pokok.
func (s *LaporanService) KartuPinjaman(idPinjaman int64) (models.KartuPinjaman, error) {
	if idPinjaman <= 0 {
		return models.KartuPinjaman{}, ErrInputAnggota
	}

	detail, err := s.pinjamanRepo.GetDetailByID(idPinjaman)
	if errors.Is(err, sql.ErrNoRows) {
		return models.KartuPinjaman{}, ErrPinjamanTidakDitemukan
	}
	if err != nil {
		return models.KartuPinjaman{}, err
	}

	jadwal, err := s.jadwalRepo.ListDetailByPinjaman(s.db, idPinjaman)
	if err != nil {
		return models.KartuPinjaman{}, err
	}
	pembayaran, err := s.pembayaranRepo.ListByPinjaman(s.db, idPinjaman)
	if err != nil {
		return models.KartuPinjaman{}, err
	}
	pelunasan, err := s.pelunasanRepo.GetByPinjaman(s.db, idPinjaman)
	if err != nil {
		return models.KartuPinjaman{}, err
	}

	var sisaPokok int64
	for _, baris := range jadwal {
		sisaPokok += baris.SisaPokok
	}
	if pelunasan != nil {
		sisaPokok = clampMin0(sisaPokok - pelunasan.SisaPokok)
	}

	return models.KartuPinjaman{
		Pinjaman:   detail,
		Jadwal:     jadwal,
		Pembayaran: pembayaran,
		Pelunasan:  pelunasan,
		SisaPokok:  sisaPokok,
	}, nil
}

func (s *LaporanService) RiwayatPembayaran(f models.FilterLaporan) ([]models.BarisPembayaran, error) {
	if err := validasiFilter(f, false); err != nil {
		return nil, err
	}
	return s.laporanRepo.RiwayatPembayaran(f)
}

func (s *LaporanService) DaftarTunggakan(tanggalAcuan time.Time) ([]models.Tunggakan, error) {
	if tanggalAcuan.IsZero() {
		tanggalAcuan = time.Now()
	}
	daftar, err := s.laporanRepo.DaftarTunggakan(tanggalAcuan)
	if err != nil {
		return nil, err
	}
	for i := range daftar {
		daftar[i].DendaBerjalan = HitungDenda(
			daftar[i].SisaPokok+daftar[i].SisaJasa,
			daftar[i].HariTerlambat,
			s.dendaPersen,
		)
	}
	return daftar, nil
}

func (s *LaporanService) PendapatanJasa(dari, sampai time.Time) (models.PendapatanJasa, error) {
	if err := validasiFilter(models.FilterLaporan{Dari: dari, Sampai: sampai}, true); err != nil {
		return models.PendapatanJasa{}, err
	}
	rincian, err := s.laporanRepo.PendapatanJasa(dari, sampai)
	if err != nil {
		return models.PendapatanJasa{}, err
	}
	return models.PendapatanJasa{Total: jumlahRincian(rincian), Rincian: rincian}, nil
}

func (s *LaporanService) PendapatanDenda(dari, sampai time.Time) (models.PendapatanDenda, error) {
	if err := validasiFilter(models.FilterLaporan{Dari: dari, Sampai: sampai}, true); err != nil {
		return models.PendapatanDenda{}, err
	}
	rincian, err := s.laporanRepo.PendapatanDenda(dari, sampai)
	if err != nil {
		return models.PendapatanDenda{}, err
	}
	return models.PendapatanDenda{Total: jumlahRincian(rincian), Rincian: rincian}, nil
}

func jumlahRincian(rincian map[string]int64) int64 {
	var total int64
	for _, jumlah := range rincian {
		total += jumlah
	}
	return total
}
