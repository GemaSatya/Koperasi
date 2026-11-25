package service

import (
	"database/sql"
	"errors"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

type KartuPinjamanRepository interface {
	GetDetailByID(id int64) (models.PinjamanDetail, error)
}

type KartuJadwalRepository interface {
	ListDetailByPinjaman(idPinjaman int64) ([]models.JadwalDetail, error)
	ListPembayaranByPinjaman(idPinjaman int64) ([]models.Pembayaran, error)
}

type KartuPelunasanRepository interface {
	GetByIDPinjaman(db repository.DBTX, idPinjaman int64) (models.Pelunasan, error)
}

type LaporanKartuService struct {
	db                  *sql.DB
	pinjamanRepository  KartuPinjamanRepository
	jadwalRepository    KartuJadwalRepository
	pelunasanRepository KartuPelunasanRepository
}

func NewLaporanKartuService(
	db *sql.DB,
	pinjamanRepository KartuPinjamanRepository,
	jadwalRepository KartuJadwalRepository,
	pelunasanRepository KartuPelunasanRepository,
) *LaporanKartuService {
	return &LaporanKartuService{
		db:                  db,
		pinjamanRepository:  pinjamanRepository,
		jadwalRepository:    jadwalRepository,
		pelunasanRepository: pelunasanRepository,
	}
}

func (s *LaporanKartuService) KartuPinjaman(idPinjaman int64) (models.KartuPinjaman, error) {
	if idPinjaman <= 0 {
		return models.KartuPinjaman{}, ErrInputPinjaman
	}

	pinjaman, err := s.pinjamanRepository.GetDetailByID(idPinjaman)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.KartuPinjaman{}, ErrPinjamanTidakDitemukan
		}
		return models.KartuPinjaman{}, err
	}
	jadwal, err := s.jadwalRepository.ListDetailByPinjaman(idPinjaman)
	if err != nil {
		return models.KartuPinjaman{}, err
	}
	pembayaran, err := s.jadwalRepository.ListPembayaranByPinjaman(idPinjaman)
	if err != nil {
		return models.KartuPinjaman{}, err
	}

	var sisaPokok int64
	for _, row := range jadwal {
		sisaPokok += row.SisaPokok
	}

	pelunasan, err := s.pelunasanRepository.GetByIDPinjaman(s.db, idPinjaman)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return models.KartuPinjaman{}, err
	}

	var dataPelunasan *models.Pelunasan
	if err == nil {
		dataPelunasan = &pelunasan
		// Pelunasan menutup pinjaman tanpa membuat baris pembayaran, jadi sisa jadwal bukan utang lagi.
		sisaPokok = 0
	}

	return models.KartuPinjaman{
		Pinjaman:   pinjaman,
		Jadwal:     jadwal,
		Pembayaran: pembayaran,
		Pelunasan:  dataPelunasan,
		SisaPokok:  sisaPokok,
	}, nil
}
