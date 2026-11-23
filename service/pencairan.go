package service

import (
	"database/sql"
	"errors"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type PencairanPinjamanRepository interface {
	GetByIDForUpdate(db repository.DBTX, id int64) (models.Pinjaman, error)
	UpdateCair(db repository.DBTX, id int64, tanggal time.Time) error
}

type JadwalInsertRepository interface {
	InsertBatch(db repository.DBTX, jadwal []models.JadwalAngsuran) error
}

type pencairanTxRunner interface {
	RunInTx(fn func(tx repository.DBTX) error) error
}

type sqlPencairanTxRunner struct {
	db *sql.DB
}

func (r sqlPencairanTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	return models.RunInTx(r.db, func(tx *sql.Tx) error {
		return fn(tx)
	})
}

type PencairanService struct {
	pinjamanRepository PencairanPinjamanRepository
	jadwalRepository   JadwalInsertRepository
	txRunner           pencairanTxRunner
}

func NewPencairanService(
	db *sql.DB,
	pinjamanRepository PencairanPinjamanRepository,
	jadwalRepository JadwalInsertRepository,
) *PencairanService {
	return &PencairanService{
		pinjamanRepository: pinjamanRepository,
		jadwalRepository:   jadwalRepository,
		txRunner:           sqlPencairanTxRunner{db: db},
	}
}

func (s *PencairanService) CairkanPinjaman(idPinjaman int64, tanggalCair time.Time) (int, error) {
	if idPinjaman <= 0 || tanggalCair.IsZero() {
		return 0, ErrInputPinjaman
	}

	var jumlahJadwal int
	err := s.txRunner.RunInTx(func(tx repository.DBTX) error {
		pinjaman, err := s.pinjamanRepository.GetByIDForUpdate(tx, idPinjaman)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrPinjamanTidakDitemukan
			}
			return err
		}
		if pinjaman.Status != "Aktif" || pinjaman.TanggalCair != nil {
			return ErrStatusPinjaman
		}
		if pinjaman.JumlahPinjaman <= 0 || pinjaman.Tenor < 1 || pinjaman.Tenor > 60 {
			return ErrJadwalTidakValid
		}

		if err := s.pinjamanRepository.UpdateCair(tx, idPinjaman, tanggalCair); err != nil {
			return err
		}

		jadwal, err := s.BuatJadwalAngsuran(tx, pinjaman, tanggalCair)
		if err != nil {
			return err
		}
		jumlahJadwal = len(jadwal)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return jumlahJadwal, nil
}

// BuatJadwalAngsuran memakai jasa_per_bulan yang tersimpan saat pengajuan pinjaman.
func (s *PencairanService) BuatJadwalAngsuran(
	tx repository.DBTX,
	pinjaman models.Pinjaman,
	tanggalCair time.Time,
) ([]models.JadwalAngsuran, error) {
	jadwal := HitungJadwal(
		pinjaman.JumlahPinjaman,
		pinjaman.Tenor,
		pinjaman.JasaPerBulan,
		tanggalCair,
	)
	if len(jadwal) == 0 {
		return nil, ErrJadwalTidakValid
	}
	for i := range jadwal {
		jadwal[i].IDPinjaman = pinjaman.IDPinjaman
	}
	if err := s.jadwalRepository.InsertBatch(tx, jadwal); err != nil {
		return nil, err
	}
	return jadwal, nil
}

func HitungJadwal(
	jumlah int64,
	tenor int,
	jasaPerBulan int64,
	tanggalCair time.Time,
) []models.JadwalAngsuran {
	if tenor < 1 {
		return nil
	}

	pokokDasar := jumlah / int64(tenor)
	sisa := jumlah - pokokDasar*int64(tenor)
	jadwal := make([]models.JadwalAngsuran, 0, tenor)

	for i := 1; i <= tenor; i++ {
		pokok := pokokDasar
		if i == tenor {
			pokok += sisa
		}

		jadwal = append(jadwal, models.JadwalAngsuran{
			AngsuranKe: i,
			JatuhTempo: util.AddMonths(tanggalCair, i),
			Pokok:      pokok,
			Jasa:       jasaPerBulan,
			Status:     "Belum",
		})
	}

	return jadwal
}
