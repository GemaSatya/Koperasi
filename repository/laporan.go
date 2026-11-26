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

// LaporanSimpanan merangkum simpanan dan penarikan per anggota.
// Periode dan id anggota bersifat opsional.
func (r *LaporanRepository) LaporanSimpanan(f models.FilterLaporan) ([]models.BarisSimpanan, error) {
	args := make([]interface{}, 0, 5)
	filterSimpanan := ""
	filterPenarikan := ""

	if !f.Dari.IsZero() && !f.Sampai.IsZero() {
		filterSimpanan = " AND s.tanggal BETWEEN ? AND ?"
		args = append(args, f.Dari, f.Sampai)
		filterPenarikan = " WHERE tanggal BETWEEN ? AND ?"
		args = append(args, f.Dari, f.Sampai)
	}

	filterAnggota := ""
	if f.IDAnggota > 0 {
		filterAnggota = " WHERE a.id_anggota = ?"
		args = append(args, f.IDAnggota)
	}

	query := `
SELECT a.id_anggota, a.no_anggota, a.nama,
       COALESCE(SUM(CASE WHEN s.jenis = 'Pokok' THEN s.nominal ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN s.jenis = 'Wajib' THEN s.nominal ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN s.jenis = 'Sukarela' THEN s.nominal ELSE 0 END), 0),
       COALESCE(t.total, 0),
       COALESCE(SUM(CASE WHEN s.jenis = 'Sukarela' THEN s.nominal ELSE 0 END), 0) - COALESCE(t.total, 0)
FROM anggota a
LEFT JOIN simpanan s ON s.id_anggota = a.id_anggota` + filterSimpanan + `
LEFT JOIN (
    SELECT id_anggota, SUM(nominal) AS total
    FROM penarikan_sukarela` + filterPenarikan + `
    GROUP BY id_anggota
) t ON t.id_anggota = a.id_anggota` + filterAnggota + `
GROUP BY a.id_anggota, a.no_anggota, a.nama, t.total
ORDER BY a.no_anggota`

	rows, err := r.db.Query(query, args...)
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
			&baris.SaldoSukarela,
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

// RiwayatPembayaran mengembalikan seluruh pembayaran beserta identitas anggota.
func (r *LaporanRepository) RiwayatPembayaran(f models.FilterLaporan) ([]models.BarisPembayaran, error) {
	conditions := []string{"1 = 1"}
	args := make([]interface{}, 0, 3)

	if !f.Dari.IsZero() && !f.Sampai.IsZero() {
		conditions = append(conditions, "pb.tanggal_bayar BETWEEN ? AND ?")
		args = append(args, f.Dari, f.Sampai)
	}
	if f.IDAnggota > 0 {
		conditions = append(conditions, "a.id_anggota = ?")
		args = append(args, f.IDAnggota)
	}

	query := `
SELECT pb.id_pembayaran, pb.tanggal_bayar, a.no_anggota, a.nama,
       p.id_pinjaman, pb.id_jadwal, pb.total_bayar,
       pb.denda_dibayar, pb.jasa_dibayar, pb.pokok_dibayar
FROM pembayaran pb
JOIN jadwal_angsuran j ON j.id_jadwal = pb.id_jadwal
JOIN pinjaman p ON p.id_pinjaman = j.id_pinjaman
JOIN anggota a ON a.id_anggota = p.id_anggota
WHERE ` + strings.Join(conditions, " AND ") + `
ORDER BY pb.tanggal_bayar, pb.id_pembayaran`

	rows, err := r.db.Query(query, args...)
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

// DaftarTunggakan mengambil angsuran belum lunas yang sudah melewati jatuh tempo.
// Denda berjalan dihitung di service agar rumusnya hanya ada satu tempat.
func (r *LaporanRepository) DaftarTunggakan(tanggalAcuan time.Time) ([]models.Tunggakan, error) {
	query := `
SELECT a.id_anggota, a.no_anggota, a.nama,
       p.id_pinjaman, j.id_jadwal, j.angsuran_ke, j.jatuh_tempo,
       GREATEST(j.pokok - COALESCE(b.pokok, 0), 0),
       GREATEST(j.jasa - COALESCE(b.jasa, 0), 0),
       DATEDIFF(?, j.jatuh_tempo)
FROM jadwal_angsuran j
JOIN pinjaman p ON p.id_pinjaman = j.id_pinjaman
JOIN anggota a ON a.id_anggota = p.id_anggota
LEFT JOIN (
    SELECT id_jadwal,
           SUM(pokok_dibayar) AS pokok,
           SUM(jasa_dibayar)  AS jasa
    FROM pembayaran
    GROUP BY id_jadwal
) b ON b.id_jadwal = j.id_jadwal
WHERE j.status = 'Belum'
  AND p.status = 'Aktif'
  AND j.jatuh_tempo < ?
ORDER BY j.jatuh_tempo, a.no_anggota`

	rows, err := r.db.Query(query, tanggalAcuan, tanggalAcuan)
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
			&tunggakan.HariTerlambat,
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

// jumlahPerBulan menjalankan query agregasi yang menghasilkan (periode, jumlah).
func (r *LaporanRepository) jumlahPerBulan(query string, dari, sampai time.Time) (map[string]int64, error) {
	rows, err := r.db.Query(query, dari, sampai)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var periode string
		var jumlah int64
		if err := rows.Scan(&periode, &jumlah); err != nil {
			return nil, err
		}
		result[periode] += jumlah
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// PendapatanJasa mengembalikan rincian jasa per bulan dari pembayaran dan pelunasan.
func (r *LaporanRepository) PendapatanJasa(dari, sampai time.Time) (map[string]int64, error) {
	pembayaran, err := r.jumlahPerBulan(`
SELECT DATE_FORMAT(tanggal_bayar, '%Y-%m'), COALESCE(SUM(jasa_dibayar), 0)
FROM pembayaran
WHERE tanggal_bayar BETWEEN ? AND ?
GROUP BY DATE_FORMAT(tanggal_bayar, '%Y-%m')`, dari, sampai)
	if err != nil {
		return nil, err
	}

	pelunasan, err := r.jumlahPerBulan(`
SELECT DATE_FORMAT(tanggal, '%Y-%m'), COALESCE(SUM(jasa), 0)
FROM pelunasan
WHERE tanggal BETWEEN ? AND ?
GROUP BY DATE_FORMAT(tanggal, '%Y-%m')`, dari, sampai)
	if err != nil {
		return nil, err
	}

	for bulan, jumlah := range pelunasan {
		pembayaran[bulan] += jumlah
	}
	return pembayaran, nil
}

// PendapatanDenda mengembalikan rincian denda yang benar-benar dibayar per bulan.
func (r *LaporanRepository) PendapatanDenda(dari, sampai time.Time) (map[string]int64, error) {
	return r.jumlahPerBulan(`
SELECT DATE_FORMAT(tanggal_bayar, '%Y-%m'), COALESCE(SUM(denda_dibayar), 0)
FROM pembayaran
WHERE tanggal_bayar BETWEEN ? AND ?
GROUP BY DATE_FORMAT(tanggal_bayar, '%Y-%m')`, dari, sampai)
}
