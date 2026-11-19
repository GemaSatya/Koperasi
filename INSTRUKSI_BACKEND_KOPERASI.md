# Instruksi Backend: Sistem Informasi Koperasi Simpan Pinjam

Dokumen ini adalah panduan tim backend (Kelompok 2 PemWeb) untuk membangun API dengan **Golang** dan **MySQL**. Isinya mengikuti dokumen persiapan (PDF): 7 tabel, ERD, dan use case admin. Tidak ada tabel, status, atau fitur tambahan di luar PDF, kecuali tabel `admin` untuk login.

Frontend hanya mengonsumsi API ini (JSON over HTTP).

---

## 1. Ringkasan Sistem

- Aktor tunggal: **Admin** (pengurus koperasi). Anggota hanya objek data, **tidak login**.
- Modul sesuai PDF: Login, Anggota, Simpanan (Pokok, Wajib, Sukarela + penarikan sukarela), Pengajuan Pinjaman, Pencairan Pinjaman, Pembayaran Angsuran (bisa sebagian, ada denda), Pelunasan Dipercepat (administrasi 2%), Laporan.
- Database: `projekpemweb` (MySQL 8, InnoDB, utf8mb4).

---

## 2. Tech Stack

| Kebutuhan | Pilihan |
|---|---|
| Bahasa | Go 1.22 atau lebih baru |
| HTTP | `net/http` standar (ServeMux dengan pola `GET /path/{id}`) |
| Driver MySQL | `github.com/go-sql-driver/mysql` |
| Akses DB | `database/sql` (tanpa ORM, query ditulis manual) |
| Konfigurasi | environment variable (opsional `github.com/joho/godotenv`) |

Alasan tanpa framework dan tanpa ORM: dependensi sedikit, query transaksi keuangan eksplisit, dan mudah dijelaskan saat presentasi.

Setup awal:

```bash
mkdir koperasi-backend && cd koperasi-backend
go mod init koperasi
go get github.com/go-sql-driver/mysql
go get github.com/joho/godotenv
```

File `.env` (jangan di-commit, buat `.env.example`):

```env
APP_PORT=8080
DB_DSN=root:password@tcp(127.0.0.1:3306)/projekpemweb?parseTime=true&charset=utf8mb4&loc=Local

# Aturan bisnis (lihat bagian 4, SESUAIKAN dengan kesepakatan kelompok/dosen)
BATAS_PINJAMAN_KALI=3
JASA_PERSEN_PER_BULAN=1.0
DENDA_PERSEN_PER_HARI=0.1
ADMINISTRASI_PELUNASAN_PERSEN=2.0
```

---

## 3. Database

### 3.1 Tabel sesuai PDF

Tujuh tabel mengikuti PDF dan ERD tanpa perubahan:

| Tabel | Kolom |
|---|---|
| `anggota` | `id_anggota` PK, `no_anggota`, `nama`, `alamat`, `no_hp`, `tanggal_daftar`, `status` (Aktif/Keluar) |
| `simpanan` | `id_simpanan` PK, `id_anggota` FK, `jenis` (Pokok/Wajib/Sukarela), `tanggal`, `nominal` |
| `penarikan_sukarela` | `id_penarikan` PK, `id_anggota` FK, `tanggal`, `nominal` |
| `pinjaman` | `id_pinjaman` PK, `id_anggota` FK, `tanggal_pengajuan`, `tanggal_cair`, `jumlah_pinjaman`, `tenor`, `jasa_per_bulan`, `status` (Aktif/Lunas) |
| `jadwal_angsuran` | `id_jadwal` PK, `id_pinjaman` FK, `angsuran_ke`, `jatuh_tempo`, `pokok`, `jasa`, `status` (Lunas/Belum) |
| `pembayaran` | `id_pembayaran` PK, `id_jadwal` FK, `tanggal_bayar`, `total_bayar`, `denda_dibayar`, `jasa_dibayar`, `pokok_dibayar` |
| `pelunasan` | `id_pelunasan` PK, `id_pinjaman` FK, `tanggal`, `sisa_pokok`, `jasa`, `administrasi`, `total` |

Relasi (ERD): `anggota` 1-N `simpanan`, `penarikan_sukarela`, `pinjaman`; `pinjaman` 1-N `jadwal_angsuran`; `pinjaman` 1-0..1 `pelunasan`; `jadwal_angsuran` 1-N `pembayaran`.

### 3.2 Tabel admin (untuk login)

```sql
CREATE TABLE admin (
    id_admin INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50)  NOT NULL UNIQUE,
    password VARCHAR(100) NOT NULL
) ENGINE=InnoDB;
```

Data admin diisi langsung di database. Tidak ada seeder. Jika tabel `admin` Anda sudah ada dengan kolom lain, cukup pastikan kolom `username` dan `password` tersedia.

### 3.3 Catatan skema

- Status `pinjaman` hanya `Aktif` dan `Lunas` (sesuai PDF). Tahap pengajuan dan pencairan dibedakan lewat `tanggal_cair`:
  - `status = 'Aktif'` dan `tanggal_cair IS NULL`: sudah diajukan, belum dicairkan, belum ada jadwal angsuran.
  - `status = 'Aktif'` dan `tanggal_cair` terisi: sudah dicairkan, jadwal angsuran sudah dibuat.
  - `status = 'Lunas'`: seluruh angsuran lunas atau sudah dilunasi dipercepat.
- Pengecekan "masih punya pinjaman aktif" = ada pinjaman anggota dengan `status = 'Aktif'` (termasuk yang belum dicairkan).
- Nominal uang memakai tipe sesuai ERD. Di Go gunakan `int64` untuk semua nominal (hindari `float`). Persentase dihitung dengan integer lalu dibulatkan.
- Anggota **tidak dihapus fisik**. "Hapus/nonaktifkan" = ubah `status` menjadi `Keluar`.
- Tidak ada kolom `sisa_pinjaman`. Sisa selalu **dihitung** dari `jadwal_angsuran` dan `pembayaran`.

---

## 4. Aturan Bisnis

PDF belum menentukan angka pada bagian ini. Nilai di bawah adalah asumsi awal, putuskan bersama kelompok lalu ubah di `.env`.

| Aturan | Asumsi awal |
|---|---|
| Batas maksimal pinjaman | 3 x total simpanan anggota (Pokok + Wajib + saldo Sukarela) |
| Pinjaman aktif | Maksimal 1 per anggota |
| Jasa pinjaman | Flat 1% per bulan dari jumlah pinjaman (`jasa_per_bulan`), sama setiap bulan |
| Pokok per angsuran | `jumlah / tenor`, sisa pembulatan masuk angsuran terakhir |
| Jatuh tempo | `tanggal_cair` + n bulan (pakai `AddMonths`) |
| Denda keterlambatan | 0,1% per hari dari sisa tagihan angsuran (pokok + jasa), mulai hari setelah jatuh tempo |
| Alokasi pembayaran | Denda, lalu jasa, lalu pokok |
| Pembayaran sebagian | Boleh. Jadwal `Lunas` jika pokok dan jasa terbayar penuh |
| Pelunasan dipercepat | Sisa pokok + jasa bulan berjalan + administrasi 2% dari sisa pokok |
| Simpanan Pokok | Hanya satu kali per anggota |
| Penarikan | Hanya Sukarela, maksimal sebesar saldo sukarela |
| Anggota `Keluar` | Tidak bisa transaksi baru |

Rumus:

```
saldo_sukarela   = SUM(simpanan sukarela) - SUM(penarikan_sukarela)
batas_maksimal   = BATAS_PINJAMAN_KALI x (total_pokok + total_wajib + saldo_sukarela)
jasa_per_bulan   = jumlah_pinjaman x JASA_PERSEN / 100
denda            = sisa_tagihan_jadwal x DENDA_PERSEN / 100 x hari_terlambat
sisa_pokok       = SUM(jadwal.pokok) - SUM(pembayaran.pokok_dibayar)   (per pinjaman)
jasa_bulan_ini   = sisa jasa pada jadwal Belum dengan angsuran_ke terkecil
administrasi     = sisa_pokok x 2 / 100
total_pelunasan  = sisa_pokok + jasa_bulan_ini + administrasi
```

Denda tidak disimpan sebagai tagihan tersendiri, tetapi dihitung ulang saat pembayaran:
`denda_terutang = HitungDenda(tanggal_bayar) - SUM(denda_dibayar untuk jadwal itu)`.

---

## 5. Struktur Proyek

```
koperasi-backend/
├── cmd/
│   └── server/main.go          # entry point API + routing
├── internal/
│   ├── config/config.go        # baca env
│   ├── db/db.go                # koneksi + helper transaksi
│   ├── model/                  # struct (Anggota, Simpanan, Pinjaman, ...)
│   ├── repository/             # query SQL saja
│   ├── service/                # aturan bisnis
│   ├── handler/                # HTTP handler (parse request, panggil service)
│   ├── middleware/cors.go      # CORS
│   └── util/                   # tanggal, uang, response JSON
├── migrations/
│   └── 001_schema.sql          # 7 tabel PDF + tabel admin
├── .env.example
└── go.mod
```

Aturan layering (agar tiga orang tidak bentrok):
- **handler**: hanya decode JSON, panggil service, kirim response. Tidak ada SQL.
- **service**: semua aturan bisnis dan perhitungan. Tidak ada `http`.
- **repository**: hanya SQL. Menerima `DBTX` agar bisa dipakai dengan atau tanpa transaksi.

### 5.1 Koneksi dan transaksi

```go
// internal/db/db.go
package db

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

// DBTX dipenuhi oleh *sql.DB dan *sql.Tx
type DBTX interface {
    Exec(q string, args ...any) (sql.Result, error)
    Query(q string, args ...any) (*sql.Rows, error)
    QueryRow(q string, args ...any) *sql.Row
}

func Open(dsn string) (*sql.DB, error) {
    d, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    return d, d.Ping()
}

// RunInTx menjalankan fn dalam satu transaksi. Error atau panic berarti rollback.
func RunInTx(d *sql.DB, fn func(tx *sql.Tx) error) (err error) {
    tx, err := d.Begin()
    if err != nil {
        return err
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p)
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
```

Gunakan `RunInTx` untuk: `CairkanPinjaman`, `BayarAngsuran`, `ProsesPelunasan`, `TarikSukarela`. Pada baca-ubah-tulis, gunakan `SELECT ... FOR UPDATE` pada baris pinjaman atau jadwal agar dua request bersamaan tidak membayar dua kali.

### 5.2 Format response

```json
{ "success": true,  "data": { }, "message": "OK" }
{ "success": false, "error": "pinjaman melebihi batas maksimal" }
```

Kode HTTP: 200/201 sukses, 400 validasi gagal, 401 username atau password salah, 404 tidak ditemukan, 409 konflik aturan bisnis (misalnya masih ada pinjaman aktif), 500 error server.

Error bisnis didefinisikan sebagai variabel agar mudah dipetakan ke 400/409:

```go
var (
    ErrAnggotaTidakAktif      = errors.New("anggota tidak aktif")
    ErrPinjamanAktifAda       = errors.New("anggota masih memiliki pinjaman aktif")
    ErrMelebihiBatas          = errors.New("jumlah pinjaman melebihi batas maksimal")
    ErrSaldoTidakCukup        = errors.New("saldo simpanan sukarela tidak cukup")
    ErrSimpananPokokAda       = errors.New("simpanan pokok sudah pernah dibayar")
    ErrStatusPinjaman         = errors.New("status pinjaman tidak sesuai untuk proses ini")
    ErrNominalMelebihiTagihan = errors.New("nominal melebihi total tagihan")
    ErrLoginGagal             = errors.New("username atau password salah")
)
```

---

## 6. Daftar Function per Modul

Notasi: parameter `db.DBTX` di repository tidak ditulis di setiap baris agar ringkas. Semua function tidak memakai `context`.

### 6.0 Utilitas (`internal/util`)

| Function | Fungsi |
|---|---|
| `AddMonths(t time.Time, n int) time.Time` | Tambah bulan dengan penjepitan akhir bulan (31 Jan + 1 = 28/29 Feb) |
| `HariTerlambat(jatuhTempo, tanggalBayar time.Time) int` | Selisih hari, minimal 0 |
| `PersenDari(nominal int64, persen float64) int64` | Hitung persen lalu bulatkan |
| `WriteJSON(w, status, payload)` / `WriteError(w, err)` | Response seragam dan pemetaan error ke kode HTTP |
| `ParseTanggal(s string) (time.Time, error)` | Format `YYYY-MM-DD` |

```go
func AddMonths(t time.Time, n int) time.Time {
    y, m, d := t.Date()
    target := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, t.Location())
    last := time.Date(target.Year(), target.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
    if d > last {
        d = last
    }
    return time.Date(target.Year(), target.Month(), d, 0, 0, 0, 0, t.Location())
}
```

### 6.1 Auth (`service/auth.go`)

Login hanya mencocokkan `username` dan `password` pada tabel `admin`. Tanpa token, tanpa hash, tanpa middleware. Jika cocok, kembalikan data admin. Jika tidak, kembalikan `ErrLoginGagal` (HTTP 401). Status login dijaga frontend (misalnya menyimpan data admin di `localStorage`, dan kembali ke halaman login jika kosong). Endpoint lain tidak memeriksa login. Ini cukup untuk tugas kuliah, tetapi tidak aman untuk dipakai sungguhan.

| Function | Fungsi |
|---|---|
| `Login(username, password string) (Admin, error)` | Cocokkan username dan password ke tabel `admin` |

```go
func (r *AdminRepo) Login(username, password string) (Admin, error) {
    var a Admin
    err := r.db.QueryRow(
        "SELECT id_admin, username FROM admin WHERE username = ? AND password = ?",
        username, password).Scan(&a.IDAdmin, &a.Username)
    if errors.Is(err, sql.ErrNoRows) {
        return Admin{}, ErrLoginGagal
    }
    return a, err
}
```

Kolom `password` tidak ikut di-`SELECT` ke response. Logout dilakukan frontend (hapus data dari `localStorage`), tidak perlu endpoint.

### 6.2 Anggota (`service/anggota.go`)

| Function | Fungsi |
|---|---|
| `TambahAnggota(in AnggotaInput) (Anggota, error)` | Validasi, generate `no_anggota`, status `Aktif`, `tanggal_daftar` default hari ini |
| `GenerateNoAnggota() (string, error)` | Contoh `AGT-2026-0001` (ambil nomor urut terakhir tahun berjalan) |
| `EditAnggota(id int64, in AnggotaInput) error` | Ubah nama, alamat, no_hp |
| `NonaktifkanAnggota(id int64) error` | Status `Keluar`. Tolak jika punya pinjaman `Aktif` |
| `GetAnggota(id int64) (Anggota, error)` | |
| `ListAnggota(keyword, status string, page, limit int) ([]Anggota, total int, err error)` | Cari nama, no_anggota, no_hp |

Repository: `Insert`, `Update`, `UpdateStatus`, `GetByID`, `List`, `CountByYear`.

### 6.3 Simpanan (`service/simpanan.go`)

| Function | Fungsi |
|---|---|
| `TambahSimpanan(in SimpananInput) (Simpanan, error)` | Validasi anggota aktif, nominal > 0, jenis valid. Jenis `Pokok` ditolak jika sudah pernah ada |
| `RiwayatSimpanan(idAnggota int64, jenis string) ([]Simpanan, error)` | `jenis` kosong berarti semua |
| `TotalSimpanan(idAnggota int64) (RingkasanSimpanan, error)` | Total Pokok, Wajib, Sukarela, penarikan, saldo sukarela, grand total |
| `SaldoSukarela(idAnggota int64) (int64, error)` | Setoran sukarela dikurangi penarikan |
| `TarikSukarela(idAnggota int64, tanggal time.Time, nominal int64) (Penarikan, error)` | Tolak jika `nominal > saldo` (`ErrSaldoTidakCukup`). Jalankan di transaksi dengan penguncian agar saldo tidak minus saat request bersamaan |
| `RiwayatPenarikan(idAnggota int64) ([]Penarikan, error)` | |

Repository: `SimpananRepo.Insert`, `ListByAnggota`, `SumByJenis`, `ExistsPokok`; `PenarikanRepo.Insert`, `SumByAnggota`, `ListByAnggota`.

### 6.4 Pengajuan pinjaman (`service/pinjaman.go`)

| Function | Fungsi |
|---|---|
| `CekPinjamanAktif(idAnggota int64) (bool, error)` | Ada pinjaman anggota dengan status `Aktif` |
| `HitungBatasMaksimal(idAnggota int64) (int64, error)` | Rumus bagian 4 |
| `HitungJasaPerBulan(jumlah int64) int64` | Flat persen dari jumlah |
| `AjukanPinjaman(in PinjamanInput) (Pinjaman, error)` | Urutan validasi: anggota aktif, tenor 1 sampai 60, tidak ada pinjaman aktif, jumlah <= batas. Insert status `Aktif`, `tanggal_cair` NULL, simpan `jasa_per_bulan` |
| `GetPinjaman(id int64) (PinjamanDetail, error)` | Termasuk nama anggota |
| `ListPinjaman(status string, idAnggota int64, page, limit int)` | |

Repository: `PinjamanRepo.Insert`, `GetByID`, `GetByIDForUpdate`, `List`, `ExistsAktifByAnggota`, `UpdateStatus`, `UpdateCair`.

### 6.5 Pencairan pinjaman (`service/pencairan.go`)

| Function | Fungsi |
|---|---|
| `CairkanPinjaman(idPinjaman int64, tanggalCair time.Time) error` | Dalam **satu transaksi**: kunci baris pinjaman, pastikan status `Aktif` dan `tanggal_cair` masih NULL (jika tidak, `ErrStatusPinjaman`), simpan `tanggal_cair`, panggil `BuatJadwalAngsuran` |
| `BuatJadwalAngsuran(tx, p Pinjaman) ([]Jadwal, error)` | Insert sebanyak `tenor` baris (pinjaman 12 bulan menghasilkan 12 data) |
| `HitungJadwal(jumlah int64, tenor int, jasaPerBulan int64, tanggalCair time.Time) []Jadwal` | Fungsi murni (tanpa DB), mudah diuji |

```go
func HitungJadwal(jumlah int64, tenor int, jasa int64, cair time.Time) []Jadwal {
    pokokDasar := jumlah / int64(tenor)
    sisa := jumlah - pokokDasar*int64(tenor)
    out := make([]Jadwal, 0, tenor)
    for i := 1; i <= tenor; i++ {
        p := pokokDasar
        if i == tenor {
            p += sisa // sisa pembulatan masuk angsuran terakhir
        }
        out = append(out, Jadwal{
            AngsuranKe: i,
            JatuhTempo: util.AddMonths(cair, i),
            Pokok:      p,
            Jasa:       jasa,
            Status:     "Belum",
        })
    }
    return out
}
```

Repository: `JadwalRepo.InsertBatch(tx, []Jadwal)`.

### 6.6 Pembayaran angsuran (`service/pembayaran.go`)

| Function | Fungsi |
|---|---|
| `GetJadwal(idPinjaman int64) ([]JadwalDetail, error)` | Jadwal + terbayar + sisa tagihan per baris |
| `HitungTagihan(idJadwal int64, tanggalBayar time.Time) (Tagihan, error)` | Sisa pokok, sisa jasa, denda terutang, total harus dibayar |
| `HitungDenda(sisaTagihan int64, hariTerlambat int) int64` | Fungsi murni |
| `BayarAngsuran(idJadwal int64, tanggalBayar time.Time, nominal int64) (Pembayaran, error)` | Lihat algoritma di bawah |
| `UpdateStatusJadwal(tx, idJadwal int64) error` | Set `Lunas` jika pokok dan jasa terbayar penuh |
| `CekDanTutupPinjaman(tx, idPinjaman int64) error` | Jika semua jadwal `Lunas`, status pinjaman menjadi `Lunas` |
| `HitungSisaPinjaman(idPinjaman int64) (int64, error)` | Sisa pokok belum dibayar |
| `RiwayatPembayaranPinjaman(idPinjaman int64) ([]Pembayaran, error)` | |

Algoritma `BayarAngsuran` (dalam satu transaksi):

```
1. Kunci jadwal dan pinjaman (FOR UPDATE). Tolak jika pinjaman bukan Aktif atau jadwal sudah Lunas.
2. Hitung:
     pokokSisa = jadwal.pokok - SUM(pokok_dibayar jadwal ini)
     jasaSisa  = jadwal.jasa  - SUM(jasa_dibayar jadwal ini)
     dendaSisa = HitungDenda(pokokSisa + jasaSisa, hariTerlambat) - SUM(denda_dibayar jadwal ini)  (minimal 0)
     totalTagihan = dendaSisa + jasaSisa + pokokSisa
3. Jika nominal <= 0 atau nominal > totalTagihan, tolak (ErrNominalMelebihiTagihan).
4. Alokasi berurutan:
     denda = min(nominal, dendaSisa);            sisaUang = nominal - denda
     jasa  = min(sisaUang, jasaSisa);            sisaUang -= jasa
     pokok = min(sisaUang, pokokSisa)
5. INSERT pembayaran (total_bayar = nominal, denda_dibayar, jasa_dibayar, pokok_dibayar).
6. UpdateStatusJadwal, lalu CekDanTutupPinjaman.
```

`hariTerlambat` dihitung dari `tanggalBayar` terhadap `jatuh_tempo`. Jika `tanggalBayar <= jatuh_tempo` maka denda 0.

Repository: `PembayaranRepo.Insert`, `SumByJadwal` (mengembalikan total denda, jasa, pokok), `ListByPinjaman`; `JadwalRepo.GetByID`, `GetByIDForUpdate`, `ListByPinjaman`, `UpdateStatus`, `CountUnpaid`.

### 6.7 Pelunasan dipercepat (`service/pelunasan.go`)

| Function | Fungsi |
|---|---|
| `HitungPelunasan(idPinjaman int64, tanggal time.Time) (RincianPelunasan, error)` | Sisa pokok, jasa bulan berjalan, administrasi 2%, total. Hanya untuk pinjaman `Aktif` yang sudah cair |
| `ProsesPelunasan(idPinjaman int64, tanggal time.Time) (Pelunasan, error)` | Transaksi: kunci pinjaman, hitung ulang di dalam transaksi, insert `pelunasan`, tandai semua jadwal `Belum` menjadi `Lunas`, status pinjaman `Lunas` |
| `GetPelunasan(idPinjaman int64) (*Pelunasan, error)` | `nil` jika belum dilunasi |

Rincian hitungan:
- `sisa_pokok` = total pokok semua jadwal dikurangi total `pokok_dibayar`
- `jasa` (bulan berjalan) = sisa jasa pada jadwal `Belum` dengan `angsuran_ke` terkecil (satu bulan jasa)
- `administrasi` = 2% x `sisa_pokok`
- `total` = `sisa_pokok + jasa + administrasi`

Pelunasan ganda dicegah lewat status: setelah `ProsesPelunasan`, pinjaman berstatus `Lunas` sehingga proses berikutnya ditolak dengan `ErrStatusPinjaman`. Karena pinjaman dikunci `FOR UPDATE`, dua request bersamaan tidak bisa lolos keduanya.

### 6.8 Laporan (`service/laporan.go`)

| Function | Fungsi | Sumber data |
|---|---|---|
| `LaporanSimpanan(f FilterLaporan) ([]BarisSimpanan, error)` | Per anggota: total Pokok, Wajib, Sukarela, penarikan, saldo | `simpanan`, `penarikan_sukarela` |
| `KartuPinjaman(idPinjaman int64) (KartuPinjaman, error)` | Data pinjaman, jadwal, pembayaran per angsuran, sisa, status, pelunasan jika ada | `pinjaman`, `jadwal_angsuran`, `pembayaran`, `pelunasan` |
| `RiwayatPembayaran(f FilterLaporan) ([]BarisPembayaran, error)` | Filter periode dan anggota | `pembayaran` JOIN `jadwal_angsuran` JOIN `pinjaman` JOIN `anggota` |
| `DaftarTunggakan(tanggalAcuan time.Time) ([]Tunggakan, error)` | Jadwal `Belum` dengan `jatuh_tempo < tanggalAcuan` beserta hari terlambat, sisa tagihan, denda berjalan | `jadwal_angsuran` + `pembayaran` |
| `PendapatanJasa(dari, sampai time.Time) (PendapatanJasa, error)` | `SUM(jasa_dibayar)` dari pembayaran ditambah `pelunasan.jasa`, rinci per bulan | `pembayaran`, `pelunasan` |
| `PendapatanDenda(dari, sampai time.Time) (PendapatanDenda, error)` | `SUM(denda_dibayar)` rinci per bulan | `pembayaran` |

`FilterLaporan`: `Dari`, `Sampai` (tanggal), `IDAnggota` (opsional).

Contoh query tunggakan (sisa tagihan dihitung dengan subquery):

```sql
SELECT a.id_anggota, a.no_anggota, a.nama,
       p.id_pinjaman, j.id_jadwal, j.angsuran_ke, j.jatuh_tempo,
       (j.pokok - COALESCE(b.pokok,0)) AS sisa_pokok,
       (j.jasa  - COALESCE(b.jasa,0))  AS sisa_jasa,
       DATEDIFF(?, j.jatuh_tempo)      AS hari_terlambat
FROM jadwal_angsuran j
JOIN pinjaman p ON p.id_pinjaman = j.id_pinjaman
JOIN anggota  a ON a.id_anggota  = p.id_anggota
LEFT JOIN (
    SELECT id_jadwal,
           SUM(pokok_dibayar) AS pokok,
           SUM(jasa_dibayar)  AS jasa
    FROM pembayaran GROUP BY id_jadwal
) b ON b.id_jadwal = j.id_jadwal
WHERE j.status = 'Belum'
  AND p.status = 'Aktif'
  AND j.jatuh_tempo < ?
ORDER BY j.jatuh_tempo;
```

Denda berjalan tidak dihitung di SQL, melainkan di Go dengan `HitungDenda` agar rumusnya satu tempat saja.

---

## 7. Daftar Endpoint REST

Backend tidak memeriksa login pada endpoint selain `/api/login`. Perlindungan halaman dilakukan di frontend setelah login berhasil.

### Auth
| Method | Path | Fungsi |
|---|---|---|
| POST | `/api/login` | `Login` |

### Anggota
| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/anggota?keyword=&status=&page=&limit=` | `ListAnggota` |
| POST | `/api/anggota` | `TambahAnggota` |
| GET | `/api/anggota/{id}` | `GetAnggota` |
| PUT | `/api/anggota/{id}` | `EditAnggota` |
| PATCH | `/api/anggota/{id}/nonaktif` | `NonaktifkanAnggota` |

### Simpanan
| Method | Path | Fungsi |
|---|---|---|
| POST | `/api/simpanan` | `TambahSimpanan` |
| GET | `/api/anggota/{id}/simpanan?jenis=` | `RiwayatSimpanan` |
| GET | `/api/anggota/{id}/simpanan/ringkasan` | `TotalSimpanan` |
| POST | `/api/penarikan-sukarela` | `TarikSukarela` |
| GET | `/api/anggota/{id}/penarikan` | `RiwayatPenarikan` |

### Pinjaman
| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/anggota/{id}/batas-pinjaman` | `HitungBatasMaksimal` + `CekPinjamanAktif` |
| POST | `/api/pinjaman` | `AjukanPinjaman` |
| GET | `/api/pinjaman?status=&id_anggota=` | `ListPinjaman` |
| GET | `/api/pinjaman/{id}` | `GetPinjaman` |
| POST | `/api/pinjaman/{id}/cairkan` | `CairkanPinjaman` |

### Angsuran dan pembayaran
| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/pinjaman/{id}/jadwal` | `GetJadwal` |
| GET | `/api/jadwal/{id}/tagihan?tanggal=` | `HitungTagihan` |
| POST | `/api/jadwal/{id}/bayar` | `BayarAngsuran` |
| GET | `/api/pinjaman/{id}/pembayaran` | `RiwayatPembayaranPinjaman` |

### Pelunasan
| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/pinjaman/{id}/pelunasan/hitung?tanggal=` | `HitungPelunasan` |
| POST | `/api/pinjaman/{id}/pelunasan` | `ProsesPelunasan` |

### Laporan
| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/laporan/simpanan?dari=&sampai=&id_anggota=` | `LaporanSimpanan` |
| GET | `/api/laporan/kartu-pinjaman/{id_pinjaman}` | `KartuPinjaman` |
| GET | `/api/laporan/riwayat-pembayaran?dari=&sampai=&id_anggota=` | `RiwayatPembayaran` |
| GET | `/api/laporan/tunggakan?tanggal=` | `DaftarTunggakan` |
| GET | `/api/laporan/pendapatan-jasa?dari=&sampai=` | `PendapatanJasa` |
| GET | `/api/laporan/pendapatan-denda?dari=&sampai=` | `PendapatanDenda` |

Contoh routing dengan Go 1.22:

```go
mux := http.NewServeMux()
mux.HandleFunc("POST /api/login", authH.Login)
mux.HandleFunc("GET /api/anggota", anggotaH.List)
mux.HandleFunc("POST /api/jadwal/{id}/bayar", pembayaranH.Bayar)
// id diambil dengan r.PathValue("id")
```

Tambahkan middleware CORS agar frontend (domain atau port berbeda) bisa memanggil API.

### Contoh request dan response

`POST /api/login`
```json
{ "username": "admin", "password": "admin123" }
```
```json
{ "success": true, "data": { "id_admin": 1, "username": "admin" } }
```

`POST /api/pinjaman`
```json
{ "id_anggota": 3, "jumlah_pinjaman": 6000000, "tenor": 12, "tanggal_pengajuan": "2026-11-10" }
```

`POST /api/jadwal/15/bayar`
```json
{ "tanggal_bayar": "2026-12-20", "nominal": 300000 }
```
```json
{
  "success": true,
  "data": {
    "id_pembayaran": 42, "total_bayar": 300000,
    "denda_dibayar": 5000, "jasa_dibayar": 60000, "pokok_dibayar": 235000,
    "status_jadwal": "Belum"
  }
}
```

---

## 8. Validasi Input (ringkas)

| Data | Aturan |
|---|---|
| `username`, `password` | Wajib diisi |
| `nama` | Wajib, maksimal 100 karakter |
| `no_hp` | Opsional, angka, 8 sampai 15 digit |
| `nominal` | Bilangan bulat > 0 |
| `jenis` | Salah satu: `Pokok`, `Wajib`, `Sukarela` |
| `tenor` | Bilangan bulat, 1 sampai 60 (sesuaikan) |
| tanggal | Format `YYYY-MM-DD` |
| `id` path | Angka positif, jika tidak 400 |

Gunakan placeholder `?` pada semua query SQL (jangan menyambung string) untuk mencegah SQL injection. Jangan mengirim kolom `password` di response login.

---

## 9. Pengujian

Prioritaskan **unit test fungsi murni** karena inilah inti perhitungan:

| Fungsi | Kasus uji |
|---|---|
| `AddMonths` | 31 Jan + 1 bulan, 29 Feb tahun kabisat, lintas tahun |
| `HitungJadwal` | Jumlah tidak habis dibagi tenor (sisa masuk angsuran terakhir), jumlah baris = tenor |
| `HitungDenda` | Tepat jatuh tempo (0), terlambat 1 hari, terlambat 30 hari |
| Alokasi pembayaran | Bayar kurang dari denda, pas denda+jasa, pas lunas, kelebihan (ditolak) |
| `HitungPelunasan` | Belum ada pembayaran, sudah bayar sebagian, pinjaman sudah `Lunas` (ditolak) |

Lalu uji manual alur utama end-to-end (Postman atau curl):
`login -> tambah anggota -> setor pokok dan wajib -> ajukan pinjaman -> cairkan -> bayar angsuran (sebagian lalu lunas) -> pelunasan dipercepat -> buka semua laporan`.

---

## 10. Pembagian Tugas Backend (3 orang)

Agar tidak saling menimpa, bagi per paket (package/file), bukan per layer.

| Anggota | Tanggung jawab |
|---|---|
| **Orang 1** | Fondasi: `config`, `db` (koneksi + `RunInTx`), `util`, CORS, modul **Auth** (login), **Anggota**, **Simpanan** + penarikan, migrasi SQL, routing di `main.go` |
| **Orang 2** | **Pinjaman** (pengajuan, batas maksimal), **Pencairan** (`HitungJadwal`, `BuatJadwalAngsuran`), **Pembayaran angsuran** (tagihan, denda, alokasi, penutupan pinjaman) |
| **Orang 3** | **Pelunasan dipercepat**, seluruh **Laporan**, unit test perhitungan, dokumentasi API (kumpulan contoh request untuk frontend) |

Kesepakatan awal yang harus selesai **sebelum** mulai coding:
1. Aturan bisnis bagian 4 (batas pinjaman, jasa, denda, administrasi).
2. Bentuk struct di `internal/model` dan format response JSON (bagian 5.2).
3. Aturan Git: satu branch per modul, `main` hanya lewat pull request, jangan commit `.env`.

Urutan pengerjaan yang disarankan:

```
1. Fondasi (config, db, util, auth, model)          Orang 1, lainnya menyusun model dan repository
2. Anggota + Simpanan                               Orang 1
   Pinjaman + Pencairan                             Orang 2   (paralel)
   Struktur laporan + test util                     Orang 3   (paralel)
3. Pembayaran angsuran                              Orang 2
   Pelunasan                                        Orang 3
4. Laporan lengkap, integrasi, uji end-to-end       Semua
5. Rapikan error, validasi, dokumentasi API         Semua
```

---

## 11. Checklist Sebelum Diserahkan ke Frontend

- [ ] Semua endpoint pada bagian 7 berjalan dan memberi response seragam
- [ ] Login berfungsi, username atau password salah mengembalikan 401
- [ ] Pencairan membuat jadwal sebanyak tenor, jumlah pokok semua jadwal = jumlah pinjaman
- [ ] Pembayaran sebagian tersimpan benar, jadwal baru `Lunas` setelah pokok dan jasa penuh
- [ ] Denda hanya muncul saat terlambat, tidak dihitung dua kali
- [ ] Pelunasan menutup pinjaman dan semua jadwal, tidak bisa dilakukan dua kali
- [ ] Anggota `Keluar` tidak bisa bertransaksi, anggota dengan pinjaman aktif tidak bisa dinonaktifkan
- [ ] Setiap operasi multi-tabel memakai transaksi
- [ ] Semua laporan menghasilkan angka yang cocok dengan hitungan manual
- [ ] `.env.example` dan petunjuk menjalankan (`go run ./cmd/server`) tersedia di README
- [ ] Contoh request/response tiap endpoint dibagikan ke tim frontend
