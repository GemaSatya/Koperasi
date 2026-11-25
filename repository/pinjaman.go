package repository

import (
	"database/sql"
	"strings"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type PinjamanRepository struct {
	db *sql.DB
}

func NewPinjamanRepository(db *sql.DB) *PinjamanRepository {
	return &PinjamanRepository{db: db}
}

type pinjamanScanner interface {
	Scan(dest ...interface{}) error
}

func (r *PinjamanRepository) Insert(db DBTX, pinjaman *models.Pinjaman) error {
	result, err := db.Exec(
		`INSERT INTO pinjaman
			(id_anggota, tanggal_pengajuan, tanggal_cair, jumlah_pinjaman, tenor, jasa_per_bulan, status)
		 VALUES (?, ?, NULL, ?, ?, ?, ?)`,
		pinjaman.IDAnggota,
		pinjaman.TanggalPengajuan,
		pinjaman.JumlahPinjaman,
		pinjaman.Tenor,
		pinjaman.JasaPerBulan,
		pinjaman.Status,
	)
	if err != nil {
		return err
	}

	pinjaman.IDPinjaman, err = result.LastInsertId()
	pinjaman.TanggalCair = nil
	return err
}

func (r *PinjamanRepository) GetByID(id int64) (models.Pinjaman, error) {
	return scanPinjaman(r.db.QueryRow(`
		SELECT id_pinjaman, id_anggota, tanggal_pengajuan, tanggal_cair,
		       jumlah_pinjaman, tenor, jasa_per_bulan, status
		FROM pinjaman
		WHERE id_pinjaman = ?`, id))
}

func (r *PinjamanRepository) GetByIDForUpdate(db DBTX, id int64) (models.Pinjaman, error) {
	return scanPinjaman(db.QueryRow(`
		SELECT id_pinjaman, id_anggota, tanggal_pengajuan, tanggal_cair,
		       jumlah_pinjaman, tenor, jasa_per_bulan, status
		FROM pinjaman
		WHERE id_pinjaman = ? FOR UPDATE`, id))
}

func (r *PinjamanRepository) GetDetailByID(id int64) (models.PinjamanDetail, error) {
	return scanPinjamanDetail(r.db.QueryRow(`
		SELECT p.id_pinjaman, p.id_anggota, p.tanggal_pengajuan, p.tanggal_cair,
		       p.jumlah_pinjaman, p.tenor, p.jasa_per_bulan, p.status,
		       a.nama, a.no_anggota
		FROM pinjaman p
		JOIN anggota a ON a.id_anggota = p.id_anggota
		WHERE p.id_pinjaman = ?`, id))
}

func (r *PinjamanRepository) List(status string, idAnggota int64, offset, limit int) ([]models.PinjamanDetail, error) {
	query := []string{`
		SELECT p.id_pinjaman, p.id_anggota, p.tanggal_pengajuan, p.tanggal_cair,
		       p.jumlah_pinjaman, p.tenor, p.jasa_per_bulan, p.status,
		       a.nama, a.no_anggota
		FROM pinjaman p
		JOIN anggota a ON a.id_anggota = p.id_anggota
		WHERE 1 = 1`}
	args := make([]interface{}, 0, 4)

	if status != "" {
		query = append(query, " AND p.status = ?")
		args = append(args, status)
	}
	if idAnggota > 0 {
		query = append(query, " AND p.id_anggota = ?")
		args = append(args, idAnggota)
	}

	query = append(query, " ORDER BY p.id_pinjaman DESC LIMIT ? OFFSET ?")
	args = append(args, limit, offset)

	rows, err := r.db.Query(strings.Join(query, ""), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.PinjamanDetail, 0)
	for rows.Next() {
		pinjaman, err := scanPinjamanDetail(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, pinjaman)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PinjamanRepository) Count(status string, idAnggota int64) (int, error) {
	query := []string{"SELECT COUNT(*) FROM pinjaman p WHERE 1 = 1"}
	args := make([]interface{}, 0, 2)

	if status != "" {
		query = append(query, " AND p.status = ?")
		args = append(args, status)
	}
	if idAnggota > 0 {
		query = append(query, " AND p.id_anggota = ?")
		args = append(args, idAnggota)
	}

	var total int
	err := r.db.QueryRow(strings.Join(query, ""), args...).Scan(&total)
	return total, err
}

func (r *PinjamanRepository) ExistsAktifByAnggota(db DBTX, idAnggota int64) (bool, error) {
	var exists int
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pinjaman WHERE id_anggota = ? AND status = 'Aktif')",
		idAnggota,
	).Scan(&exists)
	return exists == 1, err
}

func (r *PinjamanRepository) UpdateStatus(db DBTX, id int64, status string) error {
	_, err := db.Exec(
		"UPDATE pinjaman SET status = ? WHERE id_pinjaman = ?",
		status,
		id,
	)
	return err
}

func (r *PinjamanRepository) UpdateCair(db DBTX, id int64, tanggal time.Time) error {
	_, err := db.Exec(
		"UPDATE pinjaman SET tanggal_cair = ? WHERE id_pinjaman = ?",
		tanggal,
		id,
	)
	return err
}

// LockAnggota mengunci baris anggota agar satu anggota tidak bisa dua kali mengajukan secara bersamaan.
func (r *PinjamanRepository) LockAnggota(db DBTX, idAnggota int64) error {
	var id int64
	return db.QueryRow(
		"SELECT id_anggota FROM anggota WHERE id_anggota = ? FOR UPDATE",
		idAnggota,
	).Scan(&id)
}

func scanPinjaman(row pinjamanScanner) (models.Pinjaman, error) {
	var pinjaman models.Pinjaman
	err := row.Scan(
		&pinjaman.IDPinjaman,
		&pinjaman.IDAnggota,
		&pinjaman.TanggalPengajuan,
		&pinjaman.TanggalCair,
		&pinjaman.JumlahPinjaman,
		&pinjaman.Tenor,
		&pinjaman.JasaPerBulan,
		&pinjaman.Status,
	)
	return pinjaman, err
}

func scanPinjamanDetail(row pinjamanScanner) (models.PinjamanDetail, error) {
	var pinjaman models.PinjamanDetail
	err := row.Scan(
		&pinjaman.IDPinjaman,
		&pinjaman.IDAnggota,
		&pinjaman.TanggalPengajuan,
		&pinjaman.TanggalCair,
		&pinjaman.JumlahPinjaman,
		&pinjaman.Tenor,
		&pinjaman.JasaPerBulan,
		&pinjaman.Status,
		&pinjaman.NamaAnggota,
		&pinjaman.NoAnggota,
	)
	return pinjaman, err
}
