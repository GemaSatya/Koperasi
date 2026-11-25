package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type PembayaranRepository struct {
	db *sql.DB
}

func NewPembayaranRepository(db *sql.DB) *PembayaranRepository {
	return &PembayaranRepository{db: db}
}

func (r *PembayaranRepository) Insert(db DBTX, pembayaran *models.Pembayaran) error {
	result, err := db.Exec(`
		INSERT INTO pembayaran
			(id_jadwal, tanggal_bayar, total_bayar, denda_dibayar, jasa_dibayar, pokok_dibayar)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		pembayaran.IDJadwal,
		pembayaran.TanggalBayar,
		pembayaran.TotalBayar,
		pembayaran.DendaDibayar,
		pembayaran.JasaDibayar,
		pembayaran.PokokDibayar,
	)
	if err != nil {
		return err
	}
	pembayaran.IDPembayaran, err = result.LastInsertId()
	return err
}

func (r *PembayaranRepository) SumByJadwal(db DBTX, idJadwal int64) (denda, jasa, pokok int64, err error) {
	err = db.QueryRow(`
		SELECT COALESCE(SUM(denda_dibayar), 0),
		       COALESCE(SUM(jasa_dibayar), 0),
		       COALESCE(SUM(pokok_dibayar), 0)
		FROM pembayaran
		WHERE id_jadwal = ?`, idJadwal).Scan(&denda, &jasa, &pokok)
	return denda, jasa, pokok, err
}

func (r *PembayaranRepository) ListByPinjaman(idPinjaman int64) ([]models.Pembayaran, error) {
	rows, err := r.db.Query(`
		SELECT b.id_pembayaran, b.id_jadwal, b.tanggal_bayar, b.total_bayar,
		       b.denda_dibayar, b.jasa_dibayar, b.pokok_dibayar
		FROM pembayaran b
		JOIN jadwal_angsuran j ON j.id_jadwal = b.id_jadwal
		WHERE j.id_pinjaman = ?
		ORDER BY j.angsuran_ke ASC, b.tanggal_bayar ASC, b.id_pembayaran ASC`, idPinjaman)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Pembayaran, 0)
	for rows.Next() {
		var pembayaran models.Pembayaran
		if err := rows.Scan(
			&pembayaran.IDPembayaran,
			&pembayaran.IDJadwal,
			&pembayaran.TanggalBayar,
			&pembayaran.TotalBayar,
			&pembayaran.DendaDibayar,
			&pembayaran.JasaDibayar,
			&pembayaran.PokokDibayar,
		); err != nil {
			return nil, err
		}
		result = append(result, pembayaran)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
