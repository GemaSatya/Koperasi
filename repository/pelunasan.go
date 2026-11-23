package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type PelunasanRepository struct {
	db *sql.DB
}

func NewPelunasanRepository(db *sql.DB) *PelunasanRepository {
	return &PelunasanRepository{db: db}
}

type pelunasanScanner interface {
	Scan(dest ...interface{}) error
}

func (r *PelunasanRepository) Insert(db DBTX, pelunasan *models.Pelunasan) error {
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
		return err
	}

	pelunasan.IDPelunasan, err = result.LastInsertId()
	return err
}

func (r *PelunasanRepository) GetByIDPinjaman(db DBTX, idPinjaman int64) (models.Pelunasan, error) {
	return scanPelunasan(db.QueryRow(`
		SELECT id_pelunasan, id_pinjaman, tanggal, sisa_pokok, jasa, administrasi, total
		FROM pelunasan
		WHERE id_pinjaman = ?`, idPinjaman))
}

// TotalSisaPokok menjumlahkan pokok jadwal tanpa menggandakan baris saat sebuah jadwal punya beberapa pembayaran.
func (r *PelunasanRepository) TotalSisaPokok(db DBTX, idPinjaman int64) (int64, error) {
	var sisa int64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(j.pokok), 0) - COALESCE((
			SELECT SUM(p.pokok_dibayar)
			FROM pembayaran p
			JOIN jadwal_angsuran jp ON jp.id_jadwal = p.id_jadwal
			WHERE jp.id_pinjaman = ?
		), 0)
		FROM jadwal_angsuran j
		WHERE j.id_pinjaman = ?`, idPinjaman, idPinjaman).Scan(&sisa)
	return sisa, err
}

// JasaBulanBerjalan mengambil sisa jasa dari jadwal Belum dengan angsuran_ke terkecil.
func (r *PelunasanRepository) JasaBulanBerjalan(db DBTX, idPinjaman int64) (int64, error) {
	var jasa int64
	err := db.QueryRow(`
		SELECT j.jasa - COALESCE(SUM(p.jasa_dibayar), 0)
		FROM jadwal_angsuran j
		LEFT JOIN pembayaran p ON p.id_jadwal = j.id_jadwal
		WHERE j.id_pinjaman = ? AND j.status = 'Belum'
		GROUP BY j.id_jadwal, j.jasa
		ORDER BY j.angsuran_ke ASC
		LIMIT 1`, idPinjaman).Scan(&jasa)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return jasa, err
}

func (r *PelunasanRepository) TandaiBelumLunas(db DBTX, idPinjaman int64) error {
	_, err := db.Exec(
		"UPDATE jadwal_angsuran SET status = 'Lunas' WHERE id_pinjaman = ? AND status = 'Belum'",
		idPinjaman,
	)
	return err
}

func scanPelunasan(row pelunasanScanner) (models.Pelunasan, error) {
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
