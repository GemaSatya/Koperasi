package repository

import (
	"database/sql"
	"strings"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
)

type LaporanRepository struct {
	db *sql.DB
}

func NewLaporanRepository(db *sql.DB) *LaporanRepository {
	return &LaporanRepository{db: db}
}

// LaporanSimpanan merangkum simpanan dan penarikan per anggota. Filter periode bersifat opsional.
func (r *LaporanRepository) LaporanSimpanan(f models.FilterLaporan) ([]models.BarisSimpanan, error) {
	query := []string{`
		SELECT a.id_anggota, a.no_anggota, a.nama,
		       COALESCE(SUM(CASE WHEN s.jenis = 'Pokok' THEN s.nominal ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN s.jenis = 'Wajib' THEN s.nominal ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN s.jenis = 'Sukarela' THEN s.nominal ELSE 0 END), 0),
		       COALESCE(t.penarikan, 0)
		FROM anggota a
		LEFT JOIN simpanan s ON s.id_anggota = a.id_anggota`}
	args := make([]interface{}, 0, 8)
	if !f.Dari.IsZero() {
		query = append(query, " AND s.tanggal >= ?")
		args = append(args, f.Dari)
	}
	if !f.Sampai.IsZero() {
		query = append(query, " AND s.tanggal <= ?")
		args = append(args, f.Sampai)
	}

	query = append(query, ` LEFT JOIN (
		SELECT id_anggota, SUM(nominal) AS penarikan
		FROM penarikan_sukarela
		WHERE 1 = 1`)
	if !f.Dari.IsZero() {
		query = append(query, " AND tanggal >= ?")
		args = append(args, f.Dari)
	}
	if !f.Sampai.IsZero() {
		query = append(query, " AND tanggal <= ?")
		args = append(args, f.Sampai)
	}
	query = append(query, " GROUP BY id_anggota) t ON t.id_anggota = a.id_anggota WHERE 1 = 1")
	if f.IDAnggota > 0 {
		query = append(query, " AND a.id_anggota = ?")
		args = append(args, f.IDAnggota)
	}
	query = append(query, " GROUP BY a.id_anggota, a.no_anggota, a.nama ORDER BY a.no_anggota")

	rows, err := r.db.Query(strings.Join(query, ""), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.BarisSimpanan, 0)
	for rows.Next() {
		var baris models.BarisSimpanan
		if err := rows.Scan(
			&baris.IDAnggota,
			&baris.NoAnggota,
			&baris.Nama,
			&baris.TotalPokok,
			&baris.TotalWajib,
			&baris.TotalSukarela,
			&baris.TotalPenarikan,
		); err != nil {
			return nil, err
		}
		baris.SaldoSukarela = baris.TotalSukarela - baris.TotalPenarikan
		result = append(result, baris)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *LaporanRepository) RiwayatPembayaran(f models.FilterLaporan) ([]models.BarisPembayaran, error) {
	query := []string{`
		SELECT b.id_pembayaran, b.tanggal_bayar, a.no_anggota, a.nama,
		       p.id_pinjaman, j.angsuran_ke, j.id_jadwal,
		       b.total_bayar, b.denda_dibayar, b.jasa_dibayar, b.pokok_dibayar
		FROM pembayaran b
		JOIN jadwal_angsuran j ON j.id_jadwal = b.id_jadwal
		JOIN pinjaman p ON p.id_pinjaman = j.id_pinjaman
		JOIN anggota a ON a.id_anggota = p.id_anggota
		WHERE 1 = 1`}
	args := make([]interface{}, 0, 3)
	if !f.Dari.IsZero() {
		query = append(query, " AND b.tanggal_bayar >= ?")
		args = append(args, f.Dari)
	}
	if !f.Sampai.IsZero() {
		query = append(query, " AND b.tanggal_bayar <= ?")
		args = append(args, f.Sampai)
	}
	if f.IDAnggota > 0 {
		query = append(query, " AND a.id_anggota = ?")
		args = append(args, f.IDAnggota)
	}
	query = append(query, " ORDER BY b.tanggal_bayar DESC, b.id_pembayaran DESC")

	rows, err := r.db.Query(strings.Join(query, ""), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.BarisPembayaran, 0)
	for rows.Next() {
		var baris models.BarisPembayaran
		if err := rows.Scan(
			&baris.IDPembayaran,
			&baris.TanggalBayar,
			&baris.NoAnggota,
			&baris.Nama,
			&baris.IDPinjaman,
			&baris.AngsuranKe,
			&baris.IDJadwal,
			&baris.TotalBayar,
			&baris.DendaDibayar,
			&baris.JasaDibayar,
			&baris.PokokDibayar,
		); err != nil {
			return nil, err
		}
		result = append(result, baris)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// DaftarTunggakan mengembalikan data mentah; hari terlambat dan denda dihitung di service.
func (r *LaporanRepository) DaftarTunggakan(tanggalAcuan time.Time) ([]models.Tunggakan, error) {
	rows, err := r.db.Query(`
		SELECT a.id_anggota, a.no_anggota, a.nama,
		       p.id_pinjaman, j.id_jadwal, j.angsuran_ke, j.jatuh_tempo,
		       j.pokok - COALESCE(b.pokok, 0),
		       j.jasa  - COALESCE(b.jasa, 0),
		       COALESCE(b.denda, 0)
		FROM jadwal_angsuran j
		JOIN pinjaman p ON p.id_pinjaman = j.id_pinjaman
		JOIN anggota a ON a.id_anggota = p.id_anggota
		LEFT JOIN (
			SELECT id_jadwal,
			       SUM(denda_dibayar) AS denda,
			       SUM(jasa_dibayar)  AS jasa,
			       SUM(pokok_dibayar) AS pokok
			FROM pembayaran
			GROUP BY id_jadwal
		) b ON b.id_jadwal = j.id_jadwal
		WHERE j.status = 'Belum'
		  AND p.status = 'Aktif'
		  AND j.jatuh_tempo < ?
		ORDER BY j.jatuh_tempo ASC, j.id_jadwal ASC`, tanggalAcuan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.Tunggakan, 0)
	for rows.Next() {
		var tunggakan models.Tunggakan
		if err := rows.Scan(
			&tunggakan.IDAnggota,
			&tunggakan.NoAnggota,
			&tunggakan.Nama,
			&tunggakan.IDPinjaman,
			&tunggakan.IDJadwal,
			&tunggakan.AngsuranKe,
			&tunggakan.JatuhTempo,
			&tunggakan.SisaPokok,
			&tunggakan.SisaJasa,
			&tunggakan.DendaDibayar,
		); err != nil {
			return nil, err
		}
		result = append(result, tunggakan)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// JasaPerBulan menggabungkan jasa angsuran yang dibayar dan jasa pelunasan.
func (r *LaporanRepository) JasaPerBulan(dari, sampai time.Time) (map[string]int64, error) {
	result := make(map[string]int64)
	rows, err := r.db.Query(`
		SELECT DATE_FORMAT(b.tanggal_bayar, '%Y-%m'), SUM(b.jasa_dibayar)
		FROM pembayaran b
		WHERE b.tanggal_bayar BETWEEN ? AND ?
		GROUP BY DATE_FORMAT(b.tanggal_bayar, '%Y-%m')
		ORDER BY DATE_FORMAT(b.tanggal_bayar, '%Y-%m')`, dari, sampai)
	if err != nil {
		return nil, err
	}
	if err := scanPerBulan(rows, result); err != nil {
		return nil, err
	}

	rows, err = r.db.Query(`
		SELECT DATE_FORMAT(l.tanggal, '%Y-%m'), SUM(l.jasa)
		FROM pelunasan l
		WHERE l.tanggal BETWEEN ? AND ?
		GROUP BY DATE_FORMAT(l.tanggal, '%Y-%m')
		ORDER BY DATE_FORMAT(l.tanggal, '%Y-%m')`, dari, sampai)
	if err != nil {
		return nil, err
	}
	if err := scanPerBulan(rows, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *LaporanRepository) DendaPerBulan(dari, sampai time.Time) (map[string]int64, error) {
	rows, err := r.db.Query(`
		SELECT DATE_FORMAT(b.tanggal_bayar, '%Y-%m'), SUM(b.denda_dibayar)
		FROM pembayaran b
		WHERE b.tanggal_bayar BETWEEN ? AND ?
		GROUP BY DATE_FORMAT(b.tanggal_bayar, '%Y-%m')
		ORDER BY DATE_FORMAT(b.tanggal_bayar, '%Y-%m')`, dari, sampai)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	if err := scanPerBulan(rows, result); err != nil {
		return nil, err
	}
	return result, nil
}

func scanPerBulan(rows *sql.Rows, result map[string]int64) error {
	defer rows.Close()
	for rows.Next() {
		var bulan string
		var nominal int64
		if err := rows.Scan(&bulan, &nominal); err != nil {
			return err
		}
		result[bulan] += nominal
	}
	return rows.Err()
}
