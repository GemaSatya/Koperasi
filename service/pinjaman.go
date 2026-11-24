package service

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type PinjamanService struct {
	db           *sql.DB
	pinjamanRepo *repository.PinjamanRepository
	simpananRepo *repository.SimpananRepository
	jadwalRepo   *repository.JadwalRepository
	batasKali    int64
	jasaPersen   float64
}

func NewPinjamanService(
	db *sql.DB,
	pinjamanRepo *repository.PinjamanRepository,
	simpananRepo *repository.SimpananRepository,
	jadwalRepo *repository.JadwalRepository,
	cfg config.Config,
) *PinjamanService {
	return &PinjamanService{
		db:           db,
		pinjamanRepo: pinjamanRepo,
		simpananRepo: simpananRepo,
		jadwalRepo:   jadwalRepo,
		batasKali:    cfg.BatasPinjamanKali,
		jasaPersen:   cfg.JasaPersenPerBulan,
	}
}

func (s *PinjamanService) CekPinjamanAktif(idAnggota int64) (bool, error) {
	if idAnggota <= 0 {
		return false, ErrInputAnggota
	}
	return s.pinjamanRepo.ExistsAktifByAnggota(s.db, idAnggota)
}

func (s *PinjamanService) HitungBatasMaksimal(idAnggota int64) (int64, error) {
	if idAnggota <= 0 {
		return 0, ErrInputAnggota
	}
	return s.hitungBatas(s.db, idAnggota)
}

func (s *PinjamanService) hitungBatas(db repository.DBTX, idAnggota int64) (int64, error) {
	ringkasan, err := s.simpananRepo.SummaryWith(db, idAnggota)
	if err != nil {
		return 0, err
	}
	return s.batasKali * ringkasan.GrandTotal, nil
}

func (s *PinjamanService) HitungJasaPerBulan(jumlah int64) int64 {
	return util.PersenDari(jumlah, s.jasaPersen)
}

func (s *PinjamanService) AjukanPinjaman(input models.PinjamanInput) (models.Pinjaman, error) {
	if input.IDAnggota <= 0 || input.JumlahPinjaman <= 0 {
		return models.Pinjaman{}, ErrInputAnggota
	}
	if input.Tenor < 1 || input.Tenor > 60 {
		return models.Pinjaman{}, ErrTenorTidakValid
	}
	tanggalPengajuan := input.TanggalPengajuan
	if tanggalPengajuan.IsZero() {
		tanggalPengajuan = time.Now()
	}

	var hasil models.Pinjaman
	err := models.RunInTx(s.db, func(tx *sql.Tx) error {
		status, err := s.simpananRepo.GetAnggotaStatus(tx, input.IDAnggota)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAnggotaTidakDitemukan
		}
		if err != nil {
			return err
		}
		if status != "Aktif" {
			return ErrAnggotaTidakAktif
		}

		aktif, err := s.pinjamanRepo.ExistsAktifByAnggota(tx, input.IDAnggota)
		if err != nil {
			return err
		}
		if aktif {
			return ErrPinjamanAktifAda
		}

		batas, err := s.hitungBatas(tx, input.IDAnggota)
		if err != nil {
			return err
		}
		if input.JumlahPinjaman > batas {
			return ErrMelebihiBatas
		}

		hasil, err = s.pinjamanRepo.Insert(tx, models.Pinjaman{
			IDAnggota:        input.IDAnggota,
			TanggalPengajuan: tanggalPengajuan,
			JumlahPinjaman:   input.JumlahPinjaman,
			Tenor:            input.Tenor,
			JasaPerBulan:     s.HitungJasaPerBulan(input.JumlahPinjaman),
			Status:           "Aktif",
		})
		return err
	})
	if err != nil {
		return models.Pinjaman{}, err
	}
	return hasil, nil
}

func (s *PinjamanService) GetPinjaman(id int64) (models.PinjamanDetail, error) {
	if id <= 0 {
		return models.PinjamanDetail{}, ErrInputAnggota
	}
	detail, err := s.pinjamanRepo.GetDetail(s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.PinjamanDetail{}, ErrPinjamanTidakDitemukan
	}
	return detail, err
}

func (s *PinjamanService) ListPinjaman(status string, idAnggota int64, page, limit int) ([]models.PinjamanDetail, int, error) {
	if page < 1 || limit < 1 || limit > 100 || idAnggota < 0 {
		return nil, 0, ErrInputAnggota
	}
	status = strings.TrimSpace(status)
	if status != "" && status != "Aktif" && status != "Lunas" {
		return nil, 0, ErrStatusPinjaman
	}

	total, err := s.pinjamanRepo.Count(status, idAnggota)
	if err != nil {
		return nil, 0, err
	}
	daftar, err := s.pinjamanRepo.List(status, idAnggota, (page-1)*limit, limit)
	if err != nil {
		return nil, 0, err
	}
	return daftar, total, nil
}
