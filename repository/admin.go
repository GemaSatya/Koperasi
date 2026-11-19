package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) FindByCredentials(username, password string) (models.Admin, error) {
	var admin models.Admin

	err := r.db.QueryRow(
		`SELECT id_admin, username, password_hash, nama
		 FROM admin
		 WHERE username = ? AND password_hash = ?`,
		username,
		password,
	).Scan(
		&admin.IDAdmin,
		&admin.Username,
		&admin.PasswordHash,
		&admin.Nama,
	)
	if err != nil {
		return models.Admin{}, err
	}

	return admin, nil
}
