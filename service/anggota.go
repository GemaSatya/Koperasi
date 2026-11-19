package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

type AnggotaService struct {
	anggotaRepository *repository.AnggotaRepository
}

func NewAnggotaService(anggotaRepository *repository.AnggotaRepository) *AnggotaService {
	return &AnggotaService{anggotaRepository: anggotaRepository}
}

func (s *AnggotaService) Tambah(input models.AnggotaInput) (models.Anggota, error) {
	if err := validateAnggotaInput(input); err != nil {
		return models.Anggota{}, err
	}

	tanggal := time.Now()
	year := tanggal.Year()
	count, err := s.anggotaRepository.CountByYear(year)
	if err != nil {
		return models.Anggota{}, err
	}

	anggota := models.Anggota{
		NoAnggota:     fmt.Sprintf("AGT-%d-%04d", year, count+1),
		Nama:          strings.TrimSpace(input.Nama),
		Alamat:        strings.TrimSpace(input.Alamat),
		NoHP:          strings.TrimSpace(input.NoHP),
		TanggalDaftar: tanggal,
		Status:        "Aktif",
	}
	if err := s.anggotaRepository.Insert(&anggota); err != nil {
		return models.Anggota{}, err
	}
	return anggota, nil
}

func (s *AnggotaService) GetByID(id int64) (models.Anggota, error) {
	if id <= 0 {
		return models.Anggota{}, ErrInputAnggota
	}

	anggota, err := s.anggotaRepository.GetByID(id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Anggota{}, ErrAnggotaTidakDitemukan
	}
	return anggota, err
}

func (s *AnggotaService) Edit(id int64, input models.AnggotaInput) error {
	if id <= 0 {
		return ErrInputAnggota
	}
	if err := validateAnggotaInput(input); err != nil {
		return err
	}

	if _, err := s.GetByID(id); err != nil {
		return err
	}
	return s.anggotaRepository.Update(id, models.AnggotaInput{
		Nama:   strings.TrimSpace(input.Nama),
		Alamat: strings.TrimSpace(input.Alamat),
		NoHP:   strings.TrimSpace(input.NoHP),
	})
}

func (s *AnggotaService) Nonaktifkan(id int64) error {
	if id <= 0 {
		return ErrInputAnggota
	}

	anggota, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if anggota.Status == "Keluar" {
		return ErrStatusAnggota
	}

	hasActiveLoan, err := s.anggotaRepository.HasActiveLoan(id)
	if err != nil {
		return err
	}
	if hasActiveLoan {
		return ErrAnggotaMasihMeminjam
	}

	return s.anggotaRepository.UpdateStatus(id, "Keluar")
}

func (s *AnggotaService) List(keyword, status string, page, limit int) ([]models.Anggota, int, error) {
	if page < 1 || limit < 1 || limit > 100 {
		return nil, 0, ErrInputAnggota
	}
	if status != "" && status != "Aktif" && status != "Keluar" {
		return nil, 0, ErrStatusAnggota
	}

	total, err := s.anggotaRepository.Count(strings.TrimSpace(keyword), status)
	if err != nil {
		return nil, 0, err
	}
	anggotaList, err := s.anggotaRepository.List(
		strings.TrimSpace(keyword),
		status,
		(page-1)*limit,
		limit,
	)
	if err != nil {
		return nil, 0, err
	}
	return anggotaList, total, nil
}

func validateAnggotaInput(input models.AnggotaInput) error {
	name := strings.TrimSpace(input.Nama)
	if name == "" || len([]rune(name)) > 100 {
		return ErrInputAnggota
	}

	phone := strings.TrimSpace(input.NoHP)
	if phone != "" && (len(phone) < 8 || len(phone) > 15) {
		return ErrInputAnggota
	}
	for _, char := range phone {
		if !unicode.IsDigit(char) {
			return ErrInputAnggota
		}
	}
	return nil
}

func ParsePositiveID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInputAnggota
	}
	return id, nil
}
