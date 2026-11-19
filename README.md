# Backend Sistem Informasi Koperasi

Backend untuk Sistem Informasi Koperasi Simpan Pinjam menggunakan Go dan MySQL.

## Teknologi

- Go 1.22+
- MySQL 8+
- `net/http`
- `database/sql`
- `github.com/go-sql-driver/mysql`
- `github.com/joho/godotenv`

Tidak menggunakan framework atau ORM.

## Menjalankan Project

### 1. Siapkan database

Buat database dan tabel menggunakan salah satu file berikut:

- `database.sql` untuk setup database dari awal.
- `migrations/001_schema.sql` untuk schema migrasi.

Database yang digunakan adalah `pemweb`.

Contoh melalui MySQL:

```sql
SOURCE database.sql;
```

### 2. Buat konfigurasi environment

Salin `.env.example` menjadi `.env`, lalu sesuaikan nilainya:

```env
APP_PORT=8080
DB_USER=root
DB_PASSWORD=password_mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=pemweb

BATAS_PINJAMAN_KALI=3
JASA_PERSEN_PER_BULAN=1.0
DENDA_PERSEN_PER_HARI=0.1
ADMINISTRASI_PELUNASAN_PERSEN=2.0
```

Jangan commit file `.env` karena dapat berisi password database.

### 3. Jalankan server

```bash
go mod tidy
go run .
```

Server berjalan pada:

```text
http://localhost:8080
```

### 4. Jalankan validasi

```bash
go test ./...
go vet ./...
```

## Struktur Project

```text
.
├── config/                  # Pembacaan konfigurasi environment
├── handler/                 # HTTP handler dan routing
├── middleware/              # Middleware CORS
├── migrations/              # Schema database
├── model/                   # Struct database dan response
├── repository/              # Query SQL
├── service/                 # Aturan bisnis
├── util/                    # Utilitas tanggal dan response JSON
├── database.sql
├── main.go
├── go.mod
└── go.sum
```

## Pembagian Layer

- `handler`
  - Membaca request JSON dan parameter URL.
  - Memanggil service.
  - Mengirim response HTTP.
  - Tidak boleh berisi query SQL.

- `service`
  - Berisi validasi dan aturan bisnis.
  - Tidak bergantung pada `net/http`.
  - Menggunakan error bersama dari `service/errors.go`.

- `repository`
  - Berisi query SQL.
  - Menggunakan placeholder `?`.
  - Tidak berisi aturan bisnis.

- `model`
  - Berisi struct tabel, input, dan response.

## Fitur yang Sudah Selesai

Bagian Gema yang sudah tersedia:

1. Konfigurasi environment dan koneksi database.
2. `RunInTx` untuk transaksi database.
3. Middleware CORS.
4. Model tabel dan model pendukung.
5. Login admin.
6. Manajemen anggota.
7. Simpanan dan penarikan sukarela.
8. Utilitas tanggal, persentase, dan response JSON.
9. Unit test utilitas tanggal dan persentase.

## Endpoint yang Sudah Tersedia

### Login

```text
POST /api/login
```

Request:

```json
{
  "username": "admin",
  "password": "admin123"
}
```

Login menggunakan kolom `password_hash` secara langsung sesuai kebutuhan tugas kuliah. Belum menggunakan token atau session.

### Anggota

```text
GET    /api/anggota?keyword=&status=&page=&limit=
POST   /api/anggota
GET    /api/anggota/{id}
PUT    /api/anggota/{id}
PATCH  /api/anggota/{id}/nonaktif
```

Request tambah atau edit anggota:

```json
{
  "nama": "Budi Santoso",
  "alamat": "Jl. Merdeka No. 10",
  "no_hp": "081234567890"
}
```

Nomor anggota dibuat otomatis dengan format:

```text
AGT-TAHUN-0001
```

Anggota tidak dihapus secara fisik. Endpoint nonaktif mengubah status menjadi `Keluar`.

### Simpanan

```text
POST /api/simpanan
GET  /api/anggota/{id}/simpanan?jenis=
GET  /api/anggota/{id}/simpanan/ringkasan
```

Request tambah simpanan:

```json
{
  "id_anggota": 1,
  "jenis": "Sukarela",
  "tanggal": "2026-11-19",
  "nominal": 200000
}
```

Jenis simpanan yang valid:

```text
Pokok
Wajib
Sukarela
```

Simpanan pokok hanya boleh dibuat satu kali untuk setiap anggota.

### Penarikan Sukarela

```text
POST /api/penarikan-sukarela
GET  /api/anggota/{id}/penarikan
```

Request penarikan:

```json
{
  "id_anggota": 1,
  "tanggal": "2026-11-19",
  "nominal": 75000
}
```

Penarikan hanya dapat dilakukan dari saldo simpanan sukarela. Proses pengecekan saldo menggunakan transaksi dan penguncian baris anggota.

## Format Response

Response sukses:

```json
{
  "success": true,
  "data": {},
  "message": "OK"
}
```

Response error:

```json
{
  "success": false,
  "error": "pesan error"
}
```

Kode HTTP yang digunakan:

- `200` atau `201`: berhasil
- `400`: input tidak valid
- `401`: login gagal
- `404`: data tidak ditemukan
- `409`: konflik aturan bisnis
- `500`: kesalahan server

## Komponen yang Bisa Digunakan Anggota Berikutnya

### Konfigurasi

Gunakan `config.Load()` dari `config/config.go`.

```go
cfg, err := config.Load()
```

Konfigurasi aturan bisnis tersedia pada:

```go
cfg.BatasPinjamanKali
cfg.JasaPersenPerBulan
cfg.DendaPersenPerHari
cfg.AdministrasiPelunasanPersen
```

### Transaksi

Gunakan `models.RunInTx` untuk operasi yang mengubah beberapa tabel:

```go
err := models.RunInTx(models.DB, func(tx *sql.Tx) error {
    // query menggunakan tx
    return nil
})
```

Untuk operasi baca-ubah-tulis, gunakan `SELECT ... FOR UPDATE`.

### Utilitas

```go
util.AddMonths(tanggal, jumlahBulan)
util.HariTerlambat(jatuhTempo, tanggalBayar)
util.PersenDari(nominal, persen)
util.ParseTanggal("2026-11-19")
util.WriteJSON(w, status, payload)
util.WriteError(w, status, err)
```

### Routing

Setiap modul harus memiliki fungsi `RegisterRoutes` sendiri:

```go
func (h *PinjamanHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("POST /api/pinjaman", h.Create)
}
```

Setelah itu daftarkan handler di `main.go`.

## Modul yang Belum Dikerjakan

Modul berikut menjadi tanggung jawab anggota berikutnya:

### Samudra

- Pengajuan pinjaman
- Perhitungan batas pinjaman
- Pencairan pinjaman
- Pembuatan jadwal angsuran
- Pelunasan dipercepat
- Kartu pinjaman

### Brian

- Pembayaran angsuran
- Perhitungan denda
- Pembayaran sebagian
- Penutupan pinjaman setelah semua angsuran lunas
- Seluruh laporan:
  - Laporan simpanan
  - Kartu pinjaman
  - Riwayat pembayaran
  - Daftar tunggakan
  - Pendapatan jasa
  - Pendapatan denda

## Aturan Integrasi

1. Jangan commit `.env`.
2. Gunakan branch terpisah untuk setiap modul.
3. Jangan mengubah query atau tabel yang sudah ada tanpa koordinasi.
4. Semua nominal uang menggunakan `int64`.
5. Semua query menggunakan placeholder `?`.
6. Handler tidak boleh berisi SQL.
7. Operasi multi-tabel wajib menggunakan transaksi.
8. Jangan mengubah route modul lain tanpa koordinasi.
9. Jalankan `go test ./...` dan `go vet ./...` sebelum membuat pull request.
10. Gunakan error bersama dari `service/errors.go`.

## Alur Integrasi yang Disarankan

```text
login
  -> tambah anggota
  -> setor simpanan pokok dan wajib
  -> ajukan pinjaman
  -> cairkan pinjaman
  -> bayar angsuran
  -> bayar sebagian atau lunas
  -> pelunasan dipercepat jika diperlukan
  -> buka laporan
```
