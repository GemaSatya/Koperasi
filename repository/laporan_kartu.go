package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type LaporanKartuRepository struct {
	db *sql.DB
}

func NewLaporanKartuRepository(db *sql.DB) *LaporanKartuRepository {
	return &LaporanKartuRepository{db: db}
}

// ListDetailByPinjaman menggabungkan jadwal dengan total pembayaran per angsuran.
func (r *LaporanKartuRepository) ListDetailByPinjaman(idPinjaman int64) ([]models.JadwalDetail, error) {
	rows, err := r.db.Query(`
		SELECT j.id_jadwal, j.id_pinjaman, j.angsuran_ke, j.jatuh_tempo,
		       j.pokok, j.jasa, j.status,
		       COALESCE(b.denda, 0), COALESCE(b.jasa, 0), COALESCE(b.pokok, 0),
		       j.jasa - COALESCE(b.jasa, 0), j.pokok - COALESCE(b.pokok, 0), 0
		FROM jadwal_angsuran j
		LEFT JOIN (
			SELECT id_jadwal,
			       SUM(denda_dibayar) AS denda,
			       SUM(jasa_dibayar)  AS jasa,
			       SUM(pokok_dibayar) AS pokok
			FROM pembayaran
			GROUP BY id_jadwal
		) b ON b.id_jadwal = j.id_jadwal
		WHERE j.id_pinjaman = ?
		ORDER BY j.angsuran_ke ASC`, idPinjaman)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.JadwalDetail, 0)
	for rows.Next() {
		var detail models.JadwalDetail
		if err := rows.Scan(
			&detail.IDJadwal,
			&detail.IDPinjaman,
			&detail.AngsuranKe,
			&detail.JatuhTempo,
			&detail.Pokok,
			&detail.Jasa,
			&detail.Status,
			&detail.DendaTerbayar,
			&detail.JasaTerbayar,
			&detail.PokokTerbayar,
			&detail.SisaJasa,
			&detail.SisaPokok,
			&detail.SisaDenda,
		); err != nil {
			return nil, err
		}
		result = append(result, detail)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *LaporanKartuRepository) ListPembayaranByPinjaman(idPinjaman int64) ([]models.Pembayaran, error) {
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
