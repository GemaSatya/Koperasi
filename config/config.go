package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                     string
	DBDSN                       string
	BatasPinjamanKali           int64
	JasaPersenPerBulan          float64
	DendaPersenPerHari          float64
	AdministrasiPelunasanPersen float64
}

func Load() (Config, error) {
	_ = godotenv.Load()

	batasPinjaman, err := envInt64("BATAS_PINJAMAN_KALI", 3)
	if err != nil {
		return Config{}, err
	}
	jasa, err := envFloat64("JASA_PERSEN_PER_BULAN", 1)
	if err != nil {
		return Config{}, err
	}
	denda, err := envFloat64("DENDA_PERSEN_PER_HARI", 0.1)
	if err != nil {
		return Config{}, err
	}
	administrasi, err := envFloat64("ADMINISTRASI_PELUNASAN_PERSEN", 2)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppPort:                     envString("APP_PORT", "8080"),
		DBDSN:                       databaseDSN(),
		BatasPinjamanKali:           batasPinjaman,
		JasaPersenPerBulan:          jasa,
		DendaPersenPerHari:          denda,
		AdministrasiPelunasanPersen: administrasi,
	}, nil
}

func databaseDSN() string {
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		return dsn
	}

	user := envString("DB_USER", "root")
	password := os.Getenv("DB_PASSWORD")
	host := envString("DB_HOST", "127.0.0.1")
	port := envString("DB_PORT", "3306")
	name := envString("DB_NAME", "pemweb")
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		user, password, host, port, name,
	)
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt64(key string, fallback int64) (int64, error) {
	value := envString(key, strconv.FormatInt(fallback, 10))
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa bilangan bulat: %w", key, err)
	}
	return parsed, nil
}

func envFloat64(key string, fallback float64) (float64, error) {
	value := envString(key, strconv.FormatFloat(fallback, 'f', -1, 64))
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka: %w", key, err)
	}
	return parsed, nil
}
