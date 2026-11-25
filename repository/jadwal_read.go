package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

// JadwalReadRepository berisi query baca/kunci jadwal untuk modul pembayaran.
type JadwalReadRepository struct {
	db *sql.DB
}

func NewJadwalReadRepository(db *sql.DB) *JadwalReadRepository {
	return &JadwalReadRepository{db: db}
}

func (r *JadwalReadRepository) GetByID(db DBTX, id int64) (models.JadwalAngsuran, error) {
	return scanJadwal(db.QueryRow(`
		SELECT id_jadwal, id_pinjaman, angsuran_ke, jatuh_tempo, pokok, jasa, status
		FROM jadwal_angsuran
		WHERE id_jadwal = ?`, id))
}

func (r *JadwalReadRepository) GetByIDForUpdate(db DBTX, id int64) (models.JadwalAngsuran, error) {
	return scanJadwal(db.QueryRow(`
		SELECT id_jadwal, id_pinjaman, angsuran_ke, jatuh_tempo, pokok, jasa, status
		FROM jadwal_angsuran
		WHERE id_jadwal = ? FOR UPDATE`, id))
}

func (r *JadwalReadRepository) ListByPinjaman(idPinjaman int64) ([]models.JadwalAngsuran, error) {
	rows, err := r.db.Query(`
		SELECT id_jadwal, id_pinjaman, angsuran_ke, jatuh_tempo, pokok, jasa, status
		FROM jadwal_angsuran
		WHERE id_pinjaman = ?
		ORDER BY angsuran_ke ASC`, idPinjaman)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.JadwalAngsuran, 0)
	for rows.Next() {
		jadwal, err := scanJadwal(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, jadwal)
	}
	return result, rows.Err()
}

// ListDetailByPinjaman menggabungkan jadwal dengan total pembayaran per angsuran.
func (r *JadwalReadRepository) ListDetailByPinjaman(idPinjaman int64) ([]models.JadwalDetail, error) {
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

func (r *JadwalReadRepository) UpdateStatus(db DBTX, id int64, status string) error {
	_, err := db.Exec(
		"UPDATE jadwal_angsuran SET status = ? WHERE id_jadwal = ?",
		status,
		id,
	)
	return err
}

func (r *JadwalReadRepository) CountUnpaid(db DBTX, idPinjaman int64) (int, error) {
	var total int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM jadwal_angsuran WHERE id_pinjaman = ? AND status = 'Belum'",
		idPinjaman,
	).Scan(&total)
	return total, err
}

type jadwalScanner interface {
	Scan(dest ...interface{}) error
}

func scanJadwal(row jadwalScanner) (models.JadwalAngsuran, error) {
	var jadwal models.JadwalAngsuran
	err := row.Scan(
		&jadwal.IDJadwal,
		&jadwal.IDPinjaman,
		&jadwal.AngsuranKe,
		&jadwal.JatuhTempo,
		&jadwal.Pokok,
		&jadwal.Jasa,
		&jadwal.Status,
	)
	return jadwal, err
}
