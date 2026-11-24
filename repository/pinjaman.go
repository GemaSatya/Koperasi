package repository

import (
	"database/sql"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

// rowScanner dipenuhi oleh *sql.Row maupun *sql.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

type PinjamanRepository struct {
	db *sql.DB
}

func NewPinjamanRepository(db *sql.DB) *PinjamanRepository {
	return &PinjamanRepository{db: db}
}

const pinjamanKolom = `p.id_pinjaman, p.id_anggota, p.tanggal_pengajuan, p.tanggal_cair,
	p.jumlah_pinjaman, p.tenor, p.jasa_per_bulan, p.status`

func scanPinjaman(row rowScanner) (models.Pinjaman, error) {
	var pinjaman models.Pinjaman
	var tanggalCair sql.NullTime
	if err := row.Scan(
		&pinjaman.IDPinjaman,
		&pinjaman.IDAnggota,
		&pinjaman.TanggalPengajuan,
		&tanggalCair,
		&pinjaman.JumlahPinjaman,
		&pinjaman.Tenor,
		&pinjaman.JasaPerBulan,
		&pinjaman.Status,
	); err != nil {
		return models.Pinjaman{}, err
	}
	if tanggalCair.Valid {
		pinjaman.TanggalCair = &tanggalCair.Time
	}
	return pinjaman, nil
}

func scanPinjamanDetail(row rowScanner) (models.PinjamanDetail, error) {
	var detail models.PinjamanDetail
	var tanggalCair sql.NullTime
	if err := row.Scan(
		&detail.IDPinjaman,
		&detail.IDAnggota,
		&detail.TanggalPengajuan,
		&tanggalCair,
		&detail.JumlahPinjaman,
		&detail.Tenor,
		&detail.JasaPerBulan,
		&detail.Status,
		&detail.NamaAnggota,
		&detail.NoAnggota,
	); err != nil {
		return models.PinjamanDetail{}, err
	}
	if tanggalCair.Valid {
		detail.TanggalCair = &tanggalCair.Time
	}
	return detail, nil
}

func (r *PinjamanRepository) Insert(db DBTX, pinjaman models.Pinjaman) (models.Pinjaman, error) {
	result, err := db.Exec(
		`INSERT INTO pinjaman
			(id_anggota, tanggal_pengajuan, jumlah_pinjaman, tenor, jasa_per_bulan, status)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		pinjaman.IDAnggota,
		pinjaman.TanggalPengajuan,
		pinjaman.JumlahPinjaman,
		pinjaman.Tenor,
		pinjaman.JasaPerBulan,
		"Aktif",
	)
	if err != nil {
		return models.Pinjaman{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Pinjaman{}, err
	}
	pinjaman.IDPinjaman = id
	pinjaman.Status = "Aktif"
	return pinjaman, nil
}

func (r *PinjamanRepository) GetByID(db DBTX, id int64) (models.Pinjaman, error) {
	return scanPinjaman(db.QueryRow(
		`SELECT `+pinjamanKolom+` FROM pinjaman p WHERE p.id_pinjaman = ?`, id,
	))
}

func (r *PinjamanRepository) GetByIDForUpdate(db DBTX, id int64) (models.Pinjaman, error) {
	return scanPinjaman(db.QueryRow(
		`SELECT `+pinjamanKolom+` FROM pinjaman p WHERE p.id_pinjaman = ? FOR UPDATE`, id,
	))
}

func (r *PinjamanRepository) GetDetail(db DBTX, id int64) (models.PinjamanDetail, error) {
	return scanPinjamanDetail(db.QueryRow(
		`SELECT `+pinjamanKolom+`, a.nama, a.no_anggota
		 FROM pinjaman p
		 JOIN anggota a ON a.id_anggota = p.id_anggota
		 WHERE p.id_pinjaman = ?`, id,
	))
}

func (r *PinjamanRepository) ExistsAktifByAnggota(db DBTX, idAnggota int64) (bool, error) {
	var exists int
	err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM pinjaman WHERE id_anggota = ? AND status = 'Aktif')`,
		idAnggota,
	).Scan(&exists)
	return exists == 1, err
}

func (r *PinjamanRepository) List(status string, idAnggota int64, offset, limit int) ([]models.PinjamanDetail, error) {
	query := `SELECT ` + pinjamanKolom + `, a.nama, a.no_anggota
		FROM pinjaman p
		JOIN anggota a ON a.id_anggota = p.id_anggota
		WHERE 1 = 1`
	args := make([]interface{}, 0, 4)

	if status != "" {
		query += " AND p.status = ?"
		args = append(args, status)
	}
	if idAnggota > 0 {
		query += " AND p.id_anggota = ?"
		args = append(args, idAnggota)
	}
	query += " ORDER BY p.id_pinjaman DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.PinjamanDetail, 0)
	for rows.Next() {
		detail, err := scanPinjamanDetail(rows)
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

func (r *PinjamanRepository) Count(status string, idAnggota int64) (int, error) {
	query := `SELECT COUNT(*) FROM pinjaman p WHERE 1 = 1`
	args := make([]interface{}, 0, 2)

	if status != "" {
		query += " AND p.status = ?"
		args = append(args, status)
	}
	if idAnggota > 0 {
		query += " AND p.id_anggota = ?"
		args = append(args, idAnggota)
	}

	var total int
	err := r.db.QueryRow(query, args...).Scan(&total)
	return total, err
}

func (r *PinjamanRepository) UpdateStatus(db DBTX, id int64, status string) error {
	_, err := db.Exec(`UPDATE pinjaman SET status = ? WHERE id_pinjaman = ?`, status, id)
	return err
}

func (r *PinjamanRepository) UpdateCair(db DBTX, id int64, tanggalCair time.Time) error {
	_, err := db.Exec(`UPDATE pinjaman SET tanggal_cair = ? WHERE id_pinjaman = ?`, tanggalCair, id)
	return err
}
