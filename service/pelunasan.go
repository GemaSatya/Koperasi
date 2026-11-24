package service

import (
	"database/sql"
	"errors"
	"time"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type PelunasanService struct {
	db            *sql.DB
	pelunasanRepo *repository.PelunasanRepository
	jadwalRepo    *repository.JadwalRepository
	pinjamanRepo  *repository.PinjamanRepository
	persenAdmin   float64
}

func NewPelunasanService(
	db *sql.DB,
	pelunasanRepo *repository.PelunasanRepository,
	jadwalRepo *repository.JadwalRepository,
	pinjamanRepo *repository.PinjamanRepository,
	cfg config.Config,
) *PelunasanService {
	return &PelunasanService{
		db:            db,
		pelunasanRepo: pelunasanRepo,
		jadwalRepo:    jadwalRepo,
		pinjamanRepo:  pinjamanRepo,
		persenAdmin:   cfg.AdministrasiPelunasanPersen,
	}
}

// HitungRincianPelunasan adalah fungsi murni perhitungan pelunasan dipercepat.
func HitungRincianPelunasan(sisaPokok, jasaBulanIni int64, persenAdmin float64) models.RincianPelunasan {
	administrasi := util.PersenDari(sisaPokok, persenAdmin)
	return models.RincianPelunasan{
		SisaPokok:    sisaPokok,
		Jasa:         jasaBulanIni,
		Administrasi: administrasi,
		Total:        sisaPokok + jasaBulanIni + administrasi,
	}
}

func (s *PelunasanService) hitungRincian(tx repository.DBTX, pinjaman models.Pinjaman) (models.RincianPelunasan, error) {
	jadwal, err := s.jadwalRepo.ListDetailByPinjaman(tx, pinjaman.IDPinjaman)
	if err != nil {
		return models.RincianPelunasan{}, err
	}

	var sisaPokok int64
	var jasaBulanIni int64
	for _, baris := range jadwal {
		sisaPokok += baris.SisaPokok
		if baris.Status == "Belum" && jasaBulanIni == 0 {
			jasaBulanIni = baris.SisaJasa
		}
	}

	rincian := HitungRincianPelunasan(sisaPokok, jasaBulanIni, s.persenAdmin)
	rincian.IDPinjaman = pinjaman.IDPinjaman
	return rincian, nil
}

func (s *PelunasanService) validasiPinjaman(tx repository.DBTX, idPinjaman int64) (models.Pinjaman, error) {
	pinjaman, err := s.pinjamanRepo.GetByID(tx, idPinjaman)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Pinjaman{}, ErrPinjamanTidakDitemukan
	}
	if err != nil {
		return models.Pinjaman{}, err
	}
	if pinjaman.Status != "Aktif" || pinjaman.TanggalCair == nil {
		return models.Pinjaman{}, ErrStatusPinjaman
	}
	return pinjaman, nil
}

func (s *PelunasanService) HitungPelunasan(idPinjaman int64, tanggal time.Time) (models.RincianPelunasan, error) {
	if idPinjaman <= 0 || tanggal.IsZero() {
		return models.RincianPelunasan{}, ErrInputAnggota
	}
	pinjaman, err := s.validasiPinjaman(s.db, idPinjaman)
	if err != nil {
		return models.RincianPelunasan{}, err
	}
	return s.hitungRincian(s.db, pinjaman)
}

// ProsesPelunasan menyimpan pelunasan, menutup semua jadwal, dan menutup pinjaman
// dalam satu transaksi. Kunci FOR UPDATE mencegah pelunasan ganda.
func (s *PelunasanService) ProsesPelunasan(idPinjaman int64, tanggal time.Time) (models.Pelunasan, error) {
	if idPinjaman <= 0 || tanggal.IsZero() {
		return models.Pelunasan{}, ErrInputAnggota
	}

	var hasil models.Pelunasan
	err := models.RunInTx(s.db, func(tx *sql.Tx) error {
		pinjaman, err := s.pinjamanRepo.GetByIDForUpdate(tx, idPinjaman)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPinjamanTidakDitemukan
		}
		if err != nil {
			return err
		}
		if pinjaman.Status != "Aktif" || pinjaman.TanggalCair == nil {
			return ErrStatusPinjaman
		}

		sudahAda, err := s.pelunasanRepo.GetByPinjaman(tx, idPinjaman)
		if err != nil {
			return err
		}
		if sudahAda != nil {
			return ErrStatusPinjaman
		}

		rincian, err := s.hitungRincian(tx, pinjaman)
		if err != nil {
			return err
		}

		hasil, err = s.pelunasanRepo.Insert(tx, models.Pelunasan{
			IDPinjaman:   idPinjaman,
			Tanggal:      tanggal,
			SisaPokok:    rincian.SisaPokok,
			Jasa:         rincian.Jasa,
			Administrasi: rincian.Administrasi,
			Total:        rincian.Total,
		})
		if err != nil {
			return err
		}
		if err := s.jadwalRepo.MarkAllLunas(tx, idPinjaman); err != nil {
			return err
		}
		return s.pinjamanRepo.UpdateStatus(tx, idPinjaman, "Lunas")
	})
	if err != nil {
		return models.Pelunasan{}, err
	}
	return hasil, nil
}

func (s *PelunasanService) GetPelunasan(idPinjaman int64) (*models.Pelunasan, error) {
	if idPinjaman <= 0 {
		return nil, ErrInputAnggota
	}
	return s.pelunasanRepo.GetByPinjaman(s.db, idPinjaman)
}
