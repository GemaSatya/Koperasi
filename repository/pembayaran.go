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

func scanPembayaran(row rowScanner) (models.Pembayaran, error) {
	var pembayaran models.Pembayaran
	err := row.Scan(
		&pembayaran.IDPembayaran,
		&pembayaran.IDJadwal,
		&pembayaran.TanggalBayar,
		&pembayaran.TotalBayar,
		&pembayaran.DendaDibayar,
		&pembayaran.JasaDibayar,
		&pembayaran.PokokDibayar,
	)
	return pembayaran, err
}

func (r *PembayaranRepository) Insert(db DBTX, pembayaran models.Pembayaran) (models.Pembayaran, error) {
	result, err := db.Exec(
		`INSERT INTO pembayaran
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
		return models.Pembayaran{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Pembayaran{}, err
	}
	pembayaran.IDPembayaran = id
	return pembayaran, nil
}

// SumByJadwal mengembalikan total denda, jasa, dan pokok yang sudah dibayar
// untuk satu jadwal angsuran.
func (r *PembayaranRepository) SumByJadwal(db DBTX, idJadwal int64) (denda, jasa, pokok int64, err error) {
	err = db.QueryRow(
		`SELECT COALESCE(SUM(denda_dibayar), 0),
		        COALESCE(SUM(jasa_dibayar), 0),
		        COALESCE(SUM(pokok_dibayar), 0)
		 FROM pembayaran
		 WHERE id_jadwal = ?`, idJadwal,
	).Scan(&denda, &jasa, &pokok)
	return denda, jasa, pokok, err
}

func (r *PembayaranRepository) ListByPinjaman(db DBTX, idPinjaman int64) ([]models.Pembayaran, error) {
	rows, err := db.Query(
		`SELECT p.id_pembayaran, p.id_jadwal, p.tanggal_bayar, p.total_bayar,
		        p.denda_dibayar, p.jasa_dibayar, p.pokok_dibayar
		 FROM pembayaran p
		 JOIN jadwal_angsuran j ON j.id_jadwal = p.id_jadwal
		 WHERE j.id_pinjaman = ?
		 ORDER BY p.tanggal_bayar, p.id_pembayaran`, idPinjaman,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Pembayaran, 0)
	for rows.Next() {
		pembayaran, err := scanPembayaran(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, pembayaran)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
