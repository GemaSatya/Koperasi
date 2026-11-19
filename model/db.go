package models

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

// ConnectDatabase membuka koneksi ke database MySQL menggunakan DSN konfigurasi.
func ConnectDatabase(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return fmt.Errorf("gagal terhubung ke database: %w", err)
	}

	DB = db
	return nil
}

type DBTX interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// RunInTx menjalankan operasi dalam satu transaksi dan rollback saat gagal.
func RunInTx(db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// CloseDatabase menutup koneksi database yang sedang digunakan.
func CloseDatabase() error {
	if DB == nil {
		return nil
	}

	err := DB.Close()
	DB = nil
	return err
}
