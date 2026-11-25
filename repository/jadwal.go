package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type JadwalRepository struct {
	db *sql.DB
}

func NewJadwalRepository(db *sql.DB) *JadwalRepository {
	return &JadwalRepository{db: db}
}

func (r *JadwalRepository) InsertBatch(db DBTX, jadwal []models.JadwalAngsuran) error {
	for _, angsuran := range jadwal {
		_, err := db.Exec(
			`INSERT INTO jadwal_angsuran
				(id_pinjaman, angsuran_ke, jatuh_tempo, pokok, jasa, status)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			angsuran.IDPinjaman,
			angsuran.AngsuranKe,
			angsuran.JatuhTempo,
			angsuran.Pokok,
			angsuran.Jasa,
			angsuran.Status,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
