package service

import (
	"database/sql"
	"errors"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/util"
)

// HitungJadwal membuat jadwal angsuran murni tanpa database.
// Sisa pembulatan pokok masuk ke angsuran terakhir.
func HitungJadwal(jumlah int64, tenor int, jasaPerBulan int64, tanggalCair time.Time) []models.JadwalAngsuran {
	if tenor <= 0 {
		return []models.JadwalAngsuran{}
	}

	pokokDasar := jumlah / int64(tenor)
	sisa := jumlah - pokokDasar*int64(tenor)
	daftar := make([]models.JadwalAngsuran, 0, tenor)

	for i := 1; i <= tenor; i++ {
		pokok := pokokDasar
		if i == tenor {
			pokok += sisa
		}
		daftar = append(daftar, models.JadwalAngsuran{
			AngsuranKe: i,
			JatuhTempo: util.AddMonths(tanggalCair, i),
			Pokok:      pokok,
			Jasa:       jasaPerBulan,
			Status:     "Belum",
		})
	}
	return daftar
}

// CairkanPinjaman menyimpan tanggal cair dan membuat seluruh jadwal angsuran
// dalam satu transaksi. Baris pinjaman dikunci agar tidak dicairkan dua kali.
func (s *PinjamanService) CairkanPinjaman(idPinjaman int64, tanggalCair time.Time) error {
	if idPinjaman <= 0 || tanggalCair.IsZero() {
		return ErrInputAnggota
	}

	return models.RunInTx(s.db, func(tx *sql.Tx) error {
		pinjaman, err := s.pinjamanRepo.GetByIDForUpdate(tx, idPinjaman)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPinjamanTidakDitemukan
		}
		if err != nil {
			return err
		}
		if pinjaman.Status != "Aktif" || pinjaman.TanggalCair != nil {
			return ErrStatusPinjaman
		}

		if err := s.pinjamanRepo.UpdateCair(tx, idPinjaman, tanggalCair); err != nil {
			return err
		}

		jadwal := HitungJadwal(
			pinjaman.JumlahPinjaman,
			pinjaman.Tenor,
			pinjaman.JasaPerBulan,
			tanggalCair,
		)
		for i := range jadwal {
			jadwal[i].IDPinjaman = idPinjaman
		}
		return s.jadwalRepo.InsertBatch(tx, jadwal)
	})
}
