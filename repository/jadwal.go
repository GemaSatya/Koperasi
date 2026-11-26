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

const jadwalKolom = `id_jadwal, id_pinjaman, angsuran_ke, jatuh_tempo, pokok, jasa, status`

// jadwalDenganPembayaran menggabungkan jadwal dengan total pembayaran per angsuran.
const jadwalDenganPembayaran = `
SELECT j.id_jadwal, j.id_pinjaman, j.angsuran_ke, j.jatuh_tempo, j.pokok, j.jasa, j.status,
       COALESCE(b.denda, 0) AS denda_terbayar,
       COALESCE(b.jasa, 0) AS jasa_terbayar,
       COALESCE(b.pokok, 0) AS pokok_terbayar,
       GREATEST(j.pokok - COALESCE(b.pokok, 0), 0) AS sisa_pokok,
       GREATEST(j.jasa - COALESCE(b.jasa, 0), 0) AS sisa_jasa
FROM jadwal_angsuran j
LEFT JOIN (
    SELECT id_jadwal,
           SUM(denda_dibayar) AS denda,
           SUM(jasa_dibayar)  AS jasa,
           SUM(pokok_dibayar) AS pokok
    FROM pembayaran
    GROUP BY id_jadwal
) b ON b.id_jadwal = j.id_jadwal`

func scanJadwal(row rowScanner) (models.JadwalAngsuran, error) {
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

func scanJadwalDetail(row rowScanner) (models.JadwalDetail, error) {
	var detail models.JadwalDetail
	err := row.Scan(
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
		&detail.SisaPokok,
		&detail.SisaJasa,
	)
	// Denda dihitung ulang dari tanggal acuan, bukan disimpan sebagai saldo.
	detail.SisaDenda = 0
	return detail, err
}

func (r *JadwalRepository) InsertBatch(db DBTX, daftar []models.JadwalAngsuran) error {
	for _, jadwal := range daftar {
		_, err := db.Exec(
			`INSERT INTO jadwal_angsuran
				(id_pinjaman, angsuran_ke, jatuh_tempo, pokok, jasa, status)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			jadwal.IDPinjaman,
			jadwal.AngsuranKe,
			jadwal.JatuhTempo,
			jadwal.Pokok,
			jadwal.Jasa,
			jadwal.Status,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *JadwalRepository) GetByID(db DBTX, id int64) (models.JadwalAngsuran, error) {
	return scanJadwal(db.QueryRow(
		`SELECT `+jadwalKolom+` FROM jadwal_angsuran WHERE id_jadwal = ?`, id,
	))
}

func (r *JadwalRepository) GetByIDForUpdate(db DBTX, id int64) (models.JadwalAngsuran, error) {
	return scanJadwal(db.QueryRow(
		`SELECT `+jadwalKolom+` FROM jadwal_angsuran WHERE id_jadwal = ? FOR UPDATE`, id,
	))
}

func (r *JadwalRepository) ListByPinjaman(db DBTX, idPinjaman int64) ([]models.JadwalAngsuran, error) {
	rows, err := db.Query(
		`SELECT `+jadwalKolom+` FROM jadwal_angsuran WHERE id_pinjaman = ? ORDER BY angsuran_ke`,
		idPinjaman,
	)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *JadwalRepository) ListDetailByPinjaman(db DBTX, idPinjaman int64) ([]models.JadwalDetail, error) {
	rows, err := db.Query(
		jadwalDenganPembayaran+` WHERE j.id_pinjaman = ? ORDER BY j.angsuran_ke`, idPinjaman,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.JadwalDetail, 0)
	for rows.Next() {
		detail, err := scanJadwalDetail(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, detail)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *JadwalRepository) CountUnpaid(db DBTX, idPinjaman int64) (int, error) {
	var total int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM jadwal_angsuran WHERE id_pinjaman = ? AND status = 'Belum'`,
		idPinjaman,
	).Scan(&total)
	return total, err
}

func (r *JadwalRepository) UpdateStatus(db DBTX, id int64, status string) error {
	_, err := db.Exec(`UPDATE jadwal_angsuran SET status = ? WHERE id_jadwal = ?`, status, id)
	return err
}

func (r *JadwalRepository) MarkAllLunas(db DBTX, idPinjaman int64) error {
	_, err := db.Exec(
		`UPDATE jadwal_angsuran SET status = 'Lunas' WHERE id_pinjaman = ? AND status = 'Belum'`,
		idPinjaman,
	)
	return err
}
