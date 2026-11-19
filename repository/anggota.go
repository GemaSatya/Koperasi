package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type AnggotaRepository struct {
	db *sql.DB
}

func NewAnggotaRepository(db *sql.DB) *AnggotaRepository {
	return &AnggotaRepository{db: db}
}

func (r *AnggotaRepository) CountByYear(year int) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM anggota WHERE YEAR(tanggal_daftar) = ?",
		year,
	).Scan(&count)
	return count, err
}

func (r *AnggotaRepository) Insert(anggota *models.Anggota) error {
	result, err := r.db.Exec(
		`INSERT INTO anggota
			(no_anggota, nama, alamat, no_hp, tanggal_daftar, status)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		anggota.NoAnggota,
		anggota.Nama,
		anggota.Alamat,
		anggota.NoHP,
		anggota.TanggalDaftar,
		anggota.Status,
	)
	if err != nil {
		return err
	}

	anggota.IDAnggota, err = result.LastInsertId()
	return err
}

func (r *AnggotaRepository) GetByID(id int64) (models.Anggota, error) {
	var anggota models.Anggota
	err := r.db.QueryRow(
		`SELECT id_anggota, no_anggota, nama, COALESCE(alamat, ''), COALESCE(no_hp, ''), tanggal_daftar, status
		 FROM anggota
		 WHERE id_anggota = ?`,
		id,
	).Scan(
		&anggota.IDAnggota,
		&anggota.NoAnggota,
		&anggota.Nama,
		&anggota.Alamat,
		&anggota.NoHP,
		&anggota.TanggalDaftar,
		&anggota.Status,
	)
	return anggota, err
}

func (r *AnggotaRepository) Update(id int64, input models.AnggotaInput) error {
	_, err := r.db.Exec(
		"UPDATE anggota SET nama = ?, alamat = ?, no_hp = ? WHERE id_anggota = ?",
		input.Nama,
		input.Alamat,
		input.NoHP,
		id,
	)
	return err
}

func (r *AnggotaRepository) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec(
		"UPDATE anggota SET status = ? WHERE id_anggota = ?",
		status,
		id,
	)
	return err
}

func (r *AnggotaRepository) HasActiveLoan(id int64) (bool, error) {
	var exists int
	err := r.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pinjaman WHERE id_anggota = ? AND status = 'Aktif')",
		id,
	).Scan(&exists)
	return exists == 1, err
}

func (r *AnggotaRepository) List(keyword, status string, offset, limit int) ([]models.Anggota, error) {
	query := `SELECT id_anggota, no_anggota, nama, COALESCE(alamat, ''), COALESCE(no_hp, ''), tanggal_daftar, status
		FROM anggota
		WHERE (nama LIKE ? OR no_anggota LIKE ? OR no_hp LIKE ?)`
	args := []interface{}{
		"%" + keyword + "%",
		"%" + keyword + "%",
		"%" + keyword + "%",
	}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY id_anggota DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	anggotaList := make([]models.Anggota, 0)
	for rows.Next() {
		var anggota models.Anggota
		if err := rows.Scan(
			&anggota.IDAnggota,
			&anggota.NoAnggota,
			&anggota.Nama,
			&anggota.Alamat,
			&anggota.NoHP,
			&anggota.TanggalDaftar,
			&anggota.Status,
		); err != nil {
			return nil, err
		}
		anggotaList = append(anggotaList, anggota)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return anggotaList, nil
}

func (r *AnggotaRepository) Count(keyword, status string) (int, error) {
	query := `SELECT COUNT(*) FROM anggota
		WHERE (nama LIKE ? OR no_anggota LIKE ? OR no_hp LIKE ?)`
	args := []interface{}{
		"%" + keyword + "%",
		"%" + keyword + "%",
		"%" + keyword + "%",
	}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	var total int
	err := r.db.QueryRow(query, args...).Scan(&total)
	return total, err
}
