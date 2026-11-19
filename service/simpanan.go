package service

import (
	"database/sql"
	"errors"
	"strings"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

type SimpananService struct {
	simpananRepository  *repository.SimpananRepository
	penarikanRepository *repository.PenarikanRepository
	db                  *sql.DB
}

func NewSimpananService(
	db *sql.DB,
	simpananRepository *repository.SimpananRepository,
	penarikanRepository *repository.PenarikanRepository,
) *SimpananService {
	return &SimpananService{
		db:                  db,
		simpananRepository:  simpananRepository,
		penarikanRepository: penarikanRepository,
	}
}

func (s *SimpananService) TambahSimpanan(input models.SimpananInput) (models.Simpanan, error) {
	if err := validateSimpanan(input); err != nil {
		return models.Simpanan{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return models.Simpanan{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	status, err := s.simpananRepository.GetAnggotaStatus(tx, input.IDAnggota)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Simpanan{}, ErrAnggotaTidakDitemukan
	}
	if err != nil {
		return models.Simpanan{}, err
	}
	if status != "Aktif" {
		return models.Simpanan{}, ErrAnggotaTidakAktif
	}

	if input.Jenis == "Pokok" {
		exists, err := s.simpananRepository.ExistsPokok(tx, input.IDAnggota)
		if err != nil {
			return models.Simpanan{}, err
		}
		if exists {
			return models.Simpanan{}, ErrSimpananPokokAda
		}
	}

	simpanan, err := s.simpananRepository.Insert(tx, input)
	if err != nil {
		return models.Simpanan{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.Simpanan{}, err
	}
	committed = true
	return simpanan, nil
}

func (s *SimpananService) RiwayatSimpanan(idAnggota int64, jenis string) ([]models.Simpanan, error) {
	if idAnggota <= 0 {
		return nil, ErrInputAnggota
	}
	if jenis != "" && !validJenisSimpanan(jenis) {
		return nil, ErrJenisSimpanan
	}
	if _, err := s.simpananRepository.GetAnggotaStatus(s.db, idAnggota); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAnggotaTidakDitemukan
	} else if err != nil {
		return nil, err
	}
	return s.simpananRepository.ListByAnggota(idAnggota, jenis)
}

func (s *SimpananService) TotalSimpanan(idAnggota int64) (models.RingkasanSimpanan, error) {
	if idAnggota <= 0 {
		return models.RingkasanSimpanan{}, ErrInputAnggota
	}
	if _, err := s.simpananRepository.GetAnggotaStatus(s.db, idAnggota); errors.Is(err, sql.ErrNoRows) {
		return models.RingkasanSimpanan{}, ErrAnggotaTidakDitemukan
	} else if err != nil {
		return models.RingkasanSimpanan{}, err
	}
	return s.simpananRepository.Summary(idAnggota)
}

func (s *SimpananService) TarikSukarela(input models.PenarikanInput) (models.Penarikan, error) {
	if input.IDAnggota <= 0 {
		return models.Penarikan{}, ErrInputAnggota
	}
	if input.Nominal <= 0 {
		return models.Penarikan{}, ErrNominalTidakValid
	}
	if input.Tanggal.IsZero() {
		return models.Penarikan{}, ErrTanggalTidakValid
	}

	tx, err := s.db.Begin()
	if err != nil {
		return models.Penarikan{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	status, err := s.simpananRepository.GetAnggotaStatus(tx, input.IDAnggota)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Penarikan{}, ErrAnggotaTidakDitemukan
	}
	if err != nil {
		return models.Penarikan{}, err
	}
	if status != "Aktif" {
		return models.Penarikan{}, ErrAnggotaTidakAktif
	}

	balance, err := s.simpananRepository.VoluntaryBalance(tx, input.IDAnggota)
	if err != nil {
		return models.Penarikan{}, err
	}
	if input.Nominal > balance {
		return models.Penarikan{}, ErrSaldoTidakCukup
	}

	penarikan, err := s.penarikanRepository.Insert(tx, input)
	if err != nil {
		return models.Penarikan{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.Penarikan{}, err
	}
	committed = true
	return penarikan, nil
}

func (s *SimpananService) RiwayatPenarikan(idAnggota int64) ([]models.Penarikan, error) {
	if idAnggota <= 0 {
		return nil, ErrInputAnggota
	}
	if _, err := s.simpananRepository.GetAnggotaStatus(s.db, idAnggota); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAnggotaTidakDitemukan
	} else if err != nil {
		return nil, err
	}
	return s.penarikanRepository.ListByAnggota(idAnggota)
}

func validateSimpanan(input models.SimpananInput) error {
	if input.IDAnggota <= 0 {
		return ErrInputAnggota
	}
	if !validJenisSimpanan(input.Jenis) {
		return ErrJenisSimpanan
	}
	if input.Nominal <= 0 {
		return ErrNominalTidakValid
	}
	if input.Tanggal.IsZero() {
		return ErrTanggalTidakValid
	}
	return nil
}

func validJenisSimpanan(jenis string) bool {
	switch strings.TrimSpace(jenis) {
	case "Pokok", "Wajib", "Sukarela":
		return true
	default:
		return false
	}
}
