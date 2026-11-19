package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type PenarikanRepository struct {
	db *sql.DB
}

func NewPenarikanRepository(db *sql.DB) *PenarikanRepository {
	return &PenarikanRepository{db: db}
}

func (r *PenarikanRepository) Insert(db DBTX, input models.PenarikanInput) (models.Penarikan, error) {
	result, err := db.Exec(
		"INSERT INTO penarikan_sukarela (id_anggota, tanggal, nominal) VALUES (?, ?, ?)",
		input.IDAnggota,
		input.Tanggal,
		input.Nominal,
	)
	if err != nil {
		return models.Penarikan{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Penarikan{}, err
	}
	return models.Penarikan{
		IDPenarikan: id,
		IDAnggota:   input.IDAnggota,
		Tanggal:     input.Tanggal,
		Nominal:     input.Nominal,
	}, nil
}

func (r *PenarikanRepository) ListByAnggota(idAnggota int64) ([]models.Penarikan, error) {
	rows, err := r.db.Query(
		`SELECT id_penarikan, id_anggota, tanggal, nominal
		 FROM penarikan_sukarela
		 WHERE id_anggota = ?
		 ORDER BY tanggal DESC, id_penarikan DESC`,
		idAnggota,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Penarikan, 0)
	for rows.Next() {
		var penarikan models.Penarikan
		if err := rows.Scan(
			&penarikan.IDPenarikan,
			&penarikan.IDAnggota,
			&penarikan.Tanggal,
			&penarikan.Nominal,
		); err != nil {
			return nil, err
		}
		result = append(result, penarikan)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
