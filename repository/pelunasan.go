package repository

import (
	"database/sql"
	"errors"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type PelunasanRepository struct {
	db *sql.DB
}

func NewPelunasanRepository(db *sql.DB) *PelunasanRepository {
	return &PelunasanRepository{db: db}
}

const pelunasanKolom = `id_pelunasan, id_pinjaman, tanggal, sisa_pokok, jasa, administrasi, total`

func scanPelunasan(row rowScanner) (models.Pelunasan, error) {
	var pelunasan models.Pelunasan
	err := row.Scan(
		&pelunasan.IDPelunasan,
		&pelunasan.IDPinjaman,
		&pelunasan.Tanggal,
		&pelunasan.SisaPokok,
		&pelunasan.Jasa,
		&pelunasan.Administrasi,
		&pelunasan.Total,
	)
	return pelunasan, err
}

func (r *PelunasanRepository) Insert(db DBTX, pelunasan models.Pelunasan) (models.Pelunasan, error) {
	result, err := db.Exec(
		`INSERT INTO pelunasan
			(id_pinjaman, tanggal, sisa_pokok, jasa, administrasi, total)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		pelunasan.IDPinjaman,
		pelunasan.Tanggal,
		pelunasan.SisaPokok,
		pelunasan.Jasa,
		pelunasan.Administrasi,
		pelunasan.Total,
	)
	if err != nil {
		return models.Pelunasan{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Pelunasan{}, err
	}
	pelunasan.IDPelunasan = id
	return pelunasan, nil
}

// GetByPinjaman mengembalikan nil jika pinjaman belum pernah dilunasi.
func (r *PelunasanRepository) GetByPinjaman(db DBTX, idPinjaman int64) (*models.Pelunasan, error) {
	pelunasan, err := scanPelunasan(db.QueryRow(
		`SELECT `+pelunasanKolom+` FROM pelunasan WHERE id_pinjaman = ?`, idPinjaman,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pelunasan, nil
}

func (r *PelunasanRepository) ListBetween(dari, sampai time.Time) ([]models.Pelunasan, error) {
	rows, err := r.db.Query(
		`SELECT `+pelunasanKolom+` FROM pelunasan WHERE tanggal BETWEEN ? AND ? ORDER BY tanggal`,
		dari, sampai,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Pelunasan, 0)
	for rows.Next() {
		pelunasan, err := scanPelunasan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, pelunasan)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
