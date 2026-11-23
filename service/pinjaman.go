package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type AnggotaLookup interface {
	GetByID(id int64) (models.Anggota, error)
}

type SimpananLookup interface {
	TotalSimpanan(idAnggota int64) (models.RingkasanSimpanan, error)
}

type PinjamanRepository interface {
	Insert(db repository.DBTX, pinjaman *models.Pinjaman) error
	GetDetailByID(id int64) (models.PinjamanDetail, error)
	List(status string, idAnggota int64, offset, limit int) ([]models.PinjamanDetail, error)
	Count(status string, idAnggota int64) (int, error)
	ExistsAktifByAnggota(db repository.DBTX, idAnggota int64) (bool, error)
	LockAnggota(db repository.DBTX, idAnggota int64) error
}

type pinjamanTxRunner interface {
	RunInTx(fn func(tx repository.DBTX) error) error
}

type sqlPinjamanTxRunner struct {
	db *sql.DB
}

func (r sqlPinjamanTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	return models.RunInTx(r.db, func(tx *sql.Tx) error {
		return fn(tx)
	})
}

type PinjamanService struct {
	db                 *sql.DB
	pinjamanRepository PinjamanRepository
	anggota            AnggotaLookup
	simpanan           SimpananLookup
	cfg                config.Config
	txRunner           pinjamanTxRunner
}

func NewPinjamanService(
	db *sql.DB,
	pinjamanRepository PinjamanRepository,
	anggota AnggotaLookup,
	simpanan SimpananLookup,
	cfg config.Config,
) *PinjamanService {
	return &PinjamanService{
		db:                 db,
		pinjamanRepository: pinjamanRepository,
		anggota:            anggota,
		simpanan:           simpanan,
		cfg:                cfg,
		txRunner:           sqlPinjamanTxRunner{db: db},
	}
}

func (s *PinjamanService) CekPinjamanAktif(idAnggota int64) (bool, error) {
	if idAnggota <= 0 {
		return false, ErrInputPinjaman
	}
	return s.pinjamanRepository.ExistsAktifByAnggota(s.db, idAnggota)
}

func (s *PinjamanService) HitungBatasMaksimal(idAnggota int64) (int64, error) {
	if idAnggota <= 0 {
		return 0, ErrInputPinjaman
	}

	ringkasan, err := s.simpanan.TotalSimpanan(idAnggota)
	if err != nil {
		return 0, err
	}
	return s.cfg.BatasPinjamanKali * ringkasan.GrandTotal, nil
}

func (s *PinjamanService) HitungJasaPerBulan(jumlah int64) int64 {
	return util.PersenDari(jumlah, s.cfg.JasaPersenPerBulan)
}

func (s *PinjamanService) BatasPinjaman(idAnggota int64) (int64, bool, error) {
	batas, err := s.HitungBatasMaksimal(idAnggota)
	if err != nil {
		return 0, false, err
	}

	aktif, err := s.CekPinjamanAktif(idAnggota)
	if err != nil {
		return 0, false, err
	}
	return batas, aktif, nil
}

func (s *PinjamanService) AjukanPinjaman(in models.PinjamanInput) (models.Pinjaman, error) {
	if err := validatePinjamanInput(in); err != nil {
		return models.Pinjaman{}, err
	}

	// Batas dihitung sebelum transaksi karena TotalSimpanan mengunci baris anggota dengan FOR UPDATE.
	batas, err := s.HitungBatasMaksimal(in.IDAnggota)
	if err != nil {
		return models.Pinjaman{}, err
	}

	var pinjaman models.Pinjaman
	err = s.txRunner.RunInTx(func(tx repository.DBTX) error {
		if err := s.pinjamanRepository.LockAnggota(tx, in.IDAnggota); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAnggotaTidakDitemukan
			}
			return err
		}

		anggota, err := s.anggota.GetByID(in.IDAnggota)
		if err != nil {
			return err
		}
		if anggota.Status != "Aktif" {
			return ErrAnggotaTidakAktif
		}

		if in.Tenor < 1 || in.Tenor > 60 {
			return ErrTenorTidakValid
		}

		aktif, err := s.pinjamanRepository.ExistsAktifByAnggota(tx, in.IDAnggota)
		if err != nil {
			return err
		}
		if aktif {
			return ErrPinjamanAktifAda
		}

		if in.JumlahPinjaman > batas {
			return ErrMelebihiBatas
		}

		pinjaman = models.Pinjaman{
			IDAnggota:        in.IDAnggota,
			TanggalPengajuan: in.TanggalPengajuan,
			JumlahPinjaman:   in.JumlahPinjaman,
			Tenor:            in.Tenor,
			JasaPerBulan:     s.HitungJasaPerBulan(in.JumlahPinjaman),
			Status:           "Aktif",
		}
		return s.pinjamanRepository.Insert(tx, &pinjaman)
	})
	if err != nil {
		return models.Pinjaman{}, err
	}
	return pinjaman, nil
}

func (s *PinjamanService) GetPinjaman(id int64) (models.PinjamanDetail, error) {
	if id <= 0 {
		return models.PinjamanDetail{}, ErrInputPinjaman
	}

	pinjaman, err := s.pinjamanRepository.GetDetailByID(id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.PinjamanDetail{}, ErrPinjamanTidakDitemukan
	}
	return pinjaman, err
}

func (s *PinjamanService) ListPinjaman(status string, idAnggota int64, page, limit int) ([]models.PinjamanDetail, int, error) {
	if page < 1 || limit < 1 || limit > 100 {
		return nil, 0, ErrInputPinjaman
	}
	status = strings.TrimSpace(status)
	if status != "" && status != "Aktif" && status != "Lunas" {
		return nil, 0, ErrInputPinjaman
	}
	if idAnggota < 0 {
		return nil, 0, ErrInputPinjaman
	}

	total, err := s.pinjamanRepository.Count(status, idAnggota)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.pinjamanRepository.List(status, idAnggota, (page-1)*limit, limit)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func validatePinjamanInput(in models.PinjamanInput) error {
	if in.IDAnggota <= 0 {
		return ErrInputPinjaman
	}
	if in.JumlahPinjaman <= 0 {
		return ErrJumlahTidakValid
	}
	if in.TanggalPengajuan.IsZero() {
		return ErrTanggalTidakValid
	}
	return nil
}
