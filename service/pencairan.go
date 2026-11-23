package service

import (
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/util"
)

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
