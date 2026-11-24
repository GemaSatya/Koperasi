package repository

import (
	"database/sql"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type DBTX interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type SimpananRepository struct {
	db *sql.DB
}

func NewSimpananRepository(db *sql.DB) *SimpananRepository {
	return &SimpananRepository{db: db}
}

func (r *SimpananRepository) GetAnggotaStatus(db DBTX, idAnggota int64) (string, error) {
	var status string
	err := db.QueryRow(
		"SELECT status FROM anggota WHERE id_anggota = ? FOR UPDATE",
		idAnggota,
	).Scan(&status)
	return status, err
}

func (r *SimpananRepository) ExistsPokok(db DBTX, idAnggota int64) (bool, error) {
	var exists int
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM simpanan WHERE id_anggota = ? AND jenis = 'Pokok')",
		idAnggota,
	).Scan(&exists)
	return exists == 1, err
}

func (r *SimpananRepository) Insert(db DBTX, input models.SimpananInput) (models.Simpanan, error) {
	result, err := db.Exec(
		"INSERT INTO simpanan (id_anggota, jenis, tanggal, nominal) VALUES (?, ?, ?, ?)",
		input.IDAnggota,
		input.Jenis,
		input.Tanggal,
		input.Nominal,
	)
	if err != nil {
		return models.Simpanan{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Simpanan{}, err
	}
	return models.Simpanan{
		IDSimpanan: id,
		IDAnggota:  input.IDAnggota,
		Jenis:      input.Jenis,
		Tanggal:    input.Tanggal,
		Nominal:    input.Nominal,
	}, nil
}

func (r *SimpananRepository) ListByAnggota(idAnggota int64, jenis string) ([]models.Simpanan, error) {
	query := `SELECT id_simpanan, id_anggota, jenis, tanggal, nominal
		FROM simpanan
		WHERE id_anggota = ?`
	args := []interface{}{idAnggota}

	if jenis != "" {
		query += " AND jenis = ?"
		args = append(args, jenis)
	}
	query += " ORDER BY tanggal DESC, id_simpanan DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Simpanan, 0)
	for rows.Next() {
		var simpanan models.Simpanan
		if err := rows.Scan(
			&simpanan.IDSimpanan,
			&simpanan.IDAnggota,
			&simpanan.Jenis,
			&simpanan.Tanggal,
			&simpanan.Nominal,
		); err != nil {
			return nil, err
		}
		result = append(result, simpanan)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *SimpananRepository) Summary(idAnggota int64) (models.RingkasanSimpanan, error) {
	return r.SummaryWith(r.db, idAnggota)
}

// SummaryWith menghitung ringkasan menggunakan koneksi atau transaksi tertentu.
func (r *SimpananRepository) SummaryWith(db DBTX, idAnggota int64) (models.RingkasanSimpanan, error) {
	var summary models.RingkasanSimpanan
	err := db.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN jenis = 'Pokok' THEN nominal ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN jenis = 'Wajib' THEN nominal ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN jenis = 'Sukarela' THEN nominal ELSE 0 END), 0)
		 FROM simpanan
		 WHERE id_anggota = ?`,
		idAnggota,
	).Scan(
		&summary.TotalPokok,
		&summary.TotalWajib,
		&summary.TotalSukarela,
	)
	if err != nil {
		return models.RingkasanSimpanan{}, err
	}

	err = db.QueryRow(
		"SELECT COALESCE(SUM(nominal), 0) FROM penarikan_sukarela WHERE id_anggota = ?",
		idAnggota,
	).Scan(&summary.TotalPenarikan)
	if err != nil {
		return models.RingkasanSimpanan{}, err
	}

	summary.SaldoSukarela = summary.TotalSukarela - summary.TotalPenarikan
	summary.GrandTotal = summary.TotalPokok + summary.TotalWajib + summary.SaldoSukarela
	return summary, nil
}

func (r *SimpananRepository) VoluntaryBalance(db DBTX, idAnggota int64) (int64, error) {
	var balance int64
	err := db.QueryRow(
		`SELECT
			COALESCE((SELECT SUM(nominal) FROM simpanan
				WHERE id_anggota = ? AND jenis = 'Sukarela'), 0)
			- COALESCE((SELECT SUM(nominal) FROM penarikan_sukarela
				WHERE id_anggota = ?), 0)`,
		idAnggota,
		idAnggota,
	).Scan(&balance)
	return balance, err
}
