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

type PelunasanPinjamanRepository interface {
	GetByID(id int64) (models.Pinjaman, error)
	GetByIDForUpdate(db repository.DBTX, id int64) (models.Pinjaman, error)
	UpdateStatus(db repository.DBTX, id int64, status string) error
}

type PelunasanDataRepository interface {
	Insert(db repository.DBTX, pelunasan *models.Pelunasan) error
	GetByIDPinjaman(db repository.DBTX, idPinjaman int64) (models.Pelunasan, error)
	TotalSisaPokok(db repository.DBTX, idPinjaman int64) (int64, error)
	JasaBulanBerjalan(db repository.DBTX, idPinjaman int64) (int64, error)
	TandaiBelumLunas(db repository.DBTX, idPinjaman int64) error
}

type pelunasanTxRunner interface {
	RunInTx(fn func(tx repository.DBTX) error) error
}

type sqlPelunasanTxRunner struct {
	db *sql.DB
}

func (r sqlPelunasanTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	return models.RunInTx(r.db, func(tx *sql.Tx) error {
		return fn(tx)
	})
}

type PelunasanService struct {
	db                  *sql.DB
	pinjamanRepository  PelunasanPinjamanRepository
	pelunasanRepository PelunasanDataRepository
	persenAdministrasi  float64
	txRunner            pelunasanTxRunner
}

func NewPelunasanService(
	db *sql.DB,
	pinjamanRepository PelunasanPinjamanRepository,
	pelunasanRepository PelunasanDataRepository,
	cfg config.Config,
) *PelunasanService {
	return &PelunasanService{
		db:                  db,
		pinjamanRepository:  pinjamanRepository,
		pelunasanRepository: pelunasanRepository,
		persenAdministrasi:  cfg.AdministrasiPelunasanPersen,
		txRunner:            sqlPelunasanTxRunner{db: db},
	}
}

// HitungRincianPelunasan adalah rumus murni; denda sengaja tidak masuk.
func HitungRincianPelunasan(sisaPokok, jasaBulanIni int64, persenAdmin float64) models.RincianPelunasan {
	administrasi := util.PersenDari(sisaPokok, persenAdmin)
	return models.RincianPelunasan{
		SisaPokok:    sisaPokok,
		Jasa:         jasaBulanIni,
		Administrasi: administrasi,
		Total:        sisaPokok + jasaBulanIni + administrasi,
	}
}

func (s *PelunasanService) HitungPelunasan(idPinjaman int64, tanggal time.Time) (models.RincianPelunasan, error) {
	if err := validasiInputPelunasan(idPinjaman, tanggal); err != nil {
		return models.RincianPelunasan{}, err
	}

	pinjaman, err := s.pinjamanRepository.GetByID(idPinjaman)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.RincianPelunasan{}, ErrPinjamanTidakDitemukan
		}
		return models.RincianPelunasan{}, err
	}
	if err := validasiPinjamanPelunasan(pinjaman); err != nil {
		return models.RincianPelunasan{}, err
	}

	rincian, err := s.hitungRincian(s.db, idPinjaman)
	if err != nil {
		return models.RincianPelunasan{}, err
	}
	rincian.IDPinjaman = idPinjaman
	return rincian, nil
}

func (s *PelunasanService) ProsesPelunasan(idPinjaman int64, tanggal time.Time) (models.Pelunasan, error) {
	if err := validasiInputPelunasan(idPinjaman, tanggal); err != nil {
		return models.Pelunasan{}, err
	}

	var pelunasan models.Pelunasan
	err := s.txRunner.RunInTx(func(tx repository.DBTX) error {
		pinjaman, err := s.pinjamanRepository.GetByIDForUpdate(tx, idPinjaman)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrPinjamanTidakDitemukan
			}
			return err
		}
		if err := validasiPinjamanPelunasan(pinjaman); err != nil {
			return err
		}

		_, err = s.pelunasanRepository.GetByIDPinjaman(tx, idPinjaman)
		if err == nil {
			return ErrStatusPinjaman
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		rincian, err := s.hitungRincian(tx, idPinjaman)
		if err != nil {
			return err
		}

		pelunasan = models.Pelunasan{
			IDPinjaman:   idPinjaman,
			Tanggal:      tanggal,
			SisaPokok:    rincian.SisaPokok,
			Jasa:         rincian.Jasa,
			Administrasi: rincian.Administrasi,
			Total:        rincian.Total,
		}
		if err := s.pelunasanRepository.Insert(tx, &pelunasan); err != nil {
			return err
		}
		if err := s.pelunasanRepository.TandaiBelumLunas(tx, idPinjaman); err != nil {
			return err
		}
		return s.pinjamanRepository.UpdateStatus(tx, idPinjaman, "Lunas")
	})
	if err != nil {
		return models.Pelunasan{}, err
	}
	return pelunasan, nil
}

func (s *PelunasanService) GetPelunasan(idPinjaman int64) (*models.Pelunasan, error) {
	if idPinjaman <= 0 {
		return nil, ErrInputPinjaman
	}

	pelunasan, err := s.pelunasanRepository.GetByIDPinjaman(s.db, idPinjaman)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pelunasan, nil
}

func (s *PelunasanService) hitungRincian(db repository.DBTX, idPinjaman int64) (models.RincianPelunasan, error) {
	sisaPokok, err := s.pelunasanRepository.TotalSisaPokok(db, idPinjaman)
	if err != nil {
		return models.RincianPelunasan{}, err
	}
	jasa, err := s.pelunasanRepository.JasaBulanBerjalan(db, idPinjaman)
	if err != nil {
		return models.RincianPelunasan{}, err
	}
	return HitungRincianPelunasan(sisaPokok, jasa, s.persenAdministrasi), nil
}

func validasiInputPelunasan(idPinjaman int64, tanggal time.Time) error {
	if idPinjaman <= 0 {
		return ErrInputPinjaman
	}
	if tanggal.IsZero() {
		return ErrTanggalTidakValid
	}
	return nil
}

func validasiPinjamanPelunasan(pinjaman models.Pinjaman) error {
	if pinjaman.Status != "Aktif" || pinjaman.TanggalCair == nil {
		return ErrStatusPinjaman
	}
	return nil
}
