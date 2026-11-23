# Rencana Pengembangan Samudra

Status: rencana kerja; Langkah 1, 2, dan 3 sudah diimplementasikan.

File ini menjadi acuan untuk bagian Samudra. Scope diambil dari:

- `README.md` bagian "Modul yang Belum Dikerjakan", sub-bagian Samudra.
- `PEMBAGIAN_TUGAS_BACKEND.md` bagian 6.
- `INSTRUKSI_BACKEND_KOPERASI.md` bagian 4, 6.4, 6.5, 6.7, 6.8, 7, dan 9.

Jika ada konflik antar dokumen, urutan prioritasnya adalah README, lalu `PEMBAGIAN_TUGAS_BACKEND.md` bagian 6, lalu `INSTRUKSI_BACKEND_KOPERASI.md`. `INSTRUKSI_BACKEND_KOPERASI.md` bagian 10 tidak dipakai untuk pembagian tugas karena berisi pembagian lama.

## Scope

Enam bagian yang dikerjakan Samudra:

1. Pengajuan pinjaman.
2. Perhitungan batas pinjaman.
3. Pencairan pinjaman.
4. Pembuatan jadwal angsuran.
5. Pelunasan dipercepat.
6. Kartu pinjaman.

Total endpoint: 8.

| Bagian | Endpoint |
| --- | --- |
| Pengajuan pinjaman | `POST /api/pinjaman` |
| Pengajuan pinjaman | `GET /api/pinjaman?status=&id_anggota=&page=&limit=` |
| Pengajuan pinjaman | `GET /api/pinjaman/{id}` |
| Batas pinjaman | `GET /api/anggota/{id}/batas-pinjaman` |
| Pencairan | `POST /api/pinjaman/{id}/cairkan` |
| Pelunasan dipercepat | `GET /api/pinjaman/{id}/pelunasan/hitung?tanggal=` |
| Pelunasan dipercepat | `POST /api/pinjaman/{id}/pelunasan` |
| Kartu pinjaman | `GET /api/laporan/kartu-pinjaman/{id_pinjaman}` |

Di luar scope:

- Endpoint pembayaran angsuran milik Brian.
- Hitung denda dan alokasi pembayaran milik Brian.
- Laporan selain kartu pinjaman.
- Perubahan skema database.
- Perubahan logika auth, anggota, dan simpanan.
- Middleware autentikasi tambahan. Spesifikasi menyatakan hanya `/api/login` yang memeriksa kredensial.

## Aturan kerja yang dipakai

- Satu branch untuk modul, `main` hanya lewat pull request.
- Rekomendasi: branch `feat/pinjaman-samudra`, dengan satu commit per langkah di bawah.
- Handler hanya decode JSON, memanggil service, dan mengirim response. Tidak ada SQL di handler.
- Service berisi aturan bisnis. Tidak mengimpor `net/http`.
- Repository hanya SQL. Semua method menerima `DBTX` supaya bisa dipakai dengan `*sql.DB` atau `*sql.Tx`.
- Semua nominal uang memakai `int64`. Persentase memakai `util.PersenDari`.
- Semua query SQL memakai placeholder `?`.
- Operasi yang menyentuh lebih dari satu tabel wajib memakai `models.RunInTx`.
- Baca-ubah-tulis memakai `SELECT ... FOR UPDATE`.
- Komentar singkat bahasa Indonesia hanya pada bagian yang tidak jelas dari kode.
- Satu langkah dikerjakan dan dicek dulu sebelum lanjut.
- Setelah setiap langkah: `go build ./...`, `go vet ./...`, `go test ./...`.

## Penyesuaian terhadap dokumen lama

Dokumen masih memakai beberapa nama yang tidak cocok dengan kode saat ini. Ikuti kode yang ada:

| Dokumen | Kode saat ini |
| --- | --- |
| `cmd/server/main.go` dan `internal/...` | struktur flat, `main.go` di root |
| `db.DBTX`, `db.RunInTx` | `models.DBTX`, `models.RunInTx` di `model/db.go` |
| `model.Jadwal` | `models.JadwalAngsuran` |
| database `projekpemweb` | default `pemweb` di config |
| `admin.password` | `admin.password_hash` |

Package model bernama `models`, bukan `model`.

## Peta dependensi

```text
Fondasi Gema sudah tersedia
(config, models, util, errors, DB, RunInTx)
        |
        +--> HitungJadwal dan tes
        |          |
        |          v
        +--> Pengajuan pinjaman dan batas pinjaman
                   |
                   v
        +--> Pencairan dan InsertBatch jadwal
                   |
         +---------+----------+
         v                    v
  Pelunasan dipercepat   Kartu pinjaman
         |                    |
         +--------+-----------+
                  v
           wiring main.go
```

Alasan urutan:

1. Fungsi hitung murni tidak butuh database dan lebih cepat diuji.
2. Pengajuan membuat baris `pinjaman` yang dibutuhkan pencairan.
3. Pencairan membuat baris `jadwal_angsuran` yang dibutuhkan pelunasan dan kartu.
4. Pelunasan membuat data `pelunasan` yang dibaca kartu.
5. `main.go` diubah terakhir agar konflik branch kecil.

Bagian tersulit sesuai `PEMBAGIAN_TUGAS_BACKEND.md` adalah `CairkanPinjaman`, karena menggabungkan lock, transaksi, validasi status, dan insert batch.

## Langkah 0: baseline

- [x] Pastikan working tree bersih.
- [x] Pastikan Go lokal bisa membangun modul. `go.mod` saat ini memakai `go 1.27.1`.
- [x] Pastikan format response dan kode HTTP sesuai spesifikasi.
- [x] Pastikan tidak ada perubahan route modul lain.

Format response:

```json
{"success": true, "data": {}, "message": "OK"}
{"success": false, "error": "pesan error"}
```

Kode HTTP:

- `200` untuk operasi baca dan operasi yang tidak membuat resource baru.
- `201` untuk `POST /api/pinjaman`.
- `400` untuk validasi input.
- `404` untuk data tidak ditemukan.
- `409` untuk konflik aturan bisnis.
- `500` untuk error server.

## Langkah 1: `HitungJadwal` dan tes

File:

- `service/pencairan.go`
- `service/pencairan_test.go`

Fungsi:

```go
HitungJadwal(
    jumlah int64,
    tenor int,
    jasaPerBulan int64,
    tanggalCair time.Time,
) []models.JadwalAngsuran
```

Aturan:

- Jumlah baris sama dengan `tenor`.
- `pokokDasar = jumlah / tenor`.
- Sisa pembulatan masuk ke angsuran terakhir.
- Jasa sama setiap bulan.
- `jatuh_tempo` ke-n = `util.AddMonths(tanggalCair, n)`.
- Status awal `Belum`.
- Service menjamin `tenor` antara 1 dan 60 sebelum memanggil fungsi ini.

Tes table-driven:

- jumlah habis dibagi tenor
- jumlah tidak habis dibagi tenor
- total pokok semua baris sama dengan jumlah pinjaman
- tenor 1
- tanggal cair 31 Januari

Selesai jika:

- fungsi murni, tanpa DB dan tanpa HTTP
- `go test ./...` lulus
- `go vet ./...` lulus

## Langkah 2: pengajuan pinjaman dan batas pinjaman

File:

- `repository/pinjaman.go`
- `service/pinjaman.go`
- `service/pinjaman_test.go`
- `handler/pinjaman.go`
- `service/errors.go`, hanya jika perlu error bersama tambahan

### Repository

Method minimal sesuai dokumen:

- `Insert`
- `GetByID`
- `GetByIDForUpdate`
- `List`
- `ExistsAktifByAnggota`
- `UpdateStatus`
- `UpdateCair`

Tambahan yang disarankan:

- `Count` untuk total pagination.
- `GetDetailByID` dengan join ke `anggota`, atau service yang menggabungkan `PinjamanDetail`.
- `LockAnggota(tx, id)` dengan `SELECT id_anggota FROM anggota WHERE id_anggota = ? FOR UPDATE`.

`LockAnggota` dipakai untuk mencegah dua request bersamaan membuat dua pinjaman aktif.

### Service

Fungsi:

- `CekPinjamanAktif(idAnggota int64) (bool, error)`
- `HitungBatasMaksimal(idAnggota int64) (int64, error)`
- `HitungJasaPerBulan(jumlah int64) int64`
- `AjukanPinjaman(in models.PinjamanInput) (models.Pinjaman, error)`
- `GetPinjaman(id int64) (models.PinjamanDetail, error)`
- `ListPinjaman(status string, idAnggota int64, page, limit int)`

Dependency lewat interface kecil:

- `AnggotaService.GetByID` untuk status anggota.
- `SimpananService.TotalSimpanan` untuk total simpanan.
- `Config.BatasPinjamanKali` dan `Config.JasaPersenPerBulan`.

Rumus:

```text
batas_maksimal = BATAS_PINJAMAN_KALI x GrandTotal simpanan
GrandTotal     = Pokok + Wajib + SaldoSukarela
jasa_per_bulan = util.PersenDari(jumlah, JASA_PERSEN_PER_BULAN)
```

Gunakan `models.RingkasanSimpanan.GrandTotal`, bukan `TotalSukarela` saja.

### Urutan validasi `AjukanPinjaman`

1. `id_anggota > 0`, `jumlah > 0`, `tanggal_pengajuan` tidak kosong.
2. Anggota ada dan berstatus `Aktif`.
3. Tenor antara 1 dan 60.
4. Belum ada pinjaman berstatus `Aktif` untuk anggota itu.
5. `jumlah <= batas maksimal`.
6. Insert dengan `status = "Aktif"`, `tanggal_cair = NULL`, `jasa_per_bulan` terisi.

Gunakan `models.RunInTx`:

1. lock baris anggota
2. cek pinjaman aktif
3. insert pinjaman

`SimpananService.TotalSimpanan` saat ini memakai `s.db`, bukan `tx`. Tidak diubah dalam scope ini. Lock anggota dipakai untuk aturan satu pinjaman aktif, bukan untuk konsistensi saldo simpanan secara ketat.

### Handler

Route:

- `GET /api/anggota/{id}/batas-pinjaman`
- `POST /api/pinjaman`
- `GET /api/pinjaman?status=&id_anggota=&page=&limit=`
- `GET /api/pinjaman/{id}`

Aturan handler:

- `tanggal_pengajuan` diterima sebagai string `YYYY-MM-DD`, lalu diparse dengan `util.ParseTanggal`.
- Path id memakai `service.ParsePositiveID`.
- `status` hanya boleh kosong, `Aktif`, atau `Lunas`.
- `page` dan `limit` mengikuti pola anggota, default `1` dan `10`, maksimal `100`.
- Respons batas-pinjaman berisi `batas_maksimal` dan `pinjaman_aktif`.

Error tambahan yang mungkin diperlukan di `service/errors.go`:

- `ErrPinjamanTidakDitemukan`
- `ErrTenorTidakValid`
- `ErrJumlahTidakValid`

Selesai jika:

- empat endpoint terdaftar
- order validasi sesuai dokumen
- `go build ./...`, `go vet ./...`, `go test ./...` lulus

## Langkah 3: pencairan dan pembuatan jadwal angsuran

File:

- `repository/jadwal.go`
- `service/pencairan.go`
- `handler/pinjaman.go`

Di `repository/jadwal.go`, Implementasikan hanya bagian Samudra:

- `InsertBatch(db DBTX, jadwal []models.JadwalAngsuran) error`

Brian nanti bisa menaruh method baca jadwal di `repository/jadwal_read.go`.

### `CairkanPinjaman`

Endpoint:

```text
POST /api/pinjaman/{id}/cairkan
```

Body:

```json
{"tanggal_cair": "2026-11-20"}
```

Urutan dalam satu `models.RunInTx`:

1. validasi id positif dan tanggal tidak kosong
2. `GetByIDForUpdate(tx, id)`
3. tolak jika pinjaman tidak ditemukan
4. tolak jika status bukan `Aktif`
5. tolak jika `TanggalCair` sudah terisi
6. `UpdateCair(tx, id, tanggalCair)`
7. panggil `BuatJadwalAngsuran`
8. commit

`BuatJadwalAngsuran` memakai `HitungJadwal`, lalu `InsertBatch`.

`CairkanPinjaman` mengembalikan `(jumlahJadwal int, error)` agar handler bisa mengisi `jumlah_jadwal` tanpa query tambahan.

Jasa yang dipakai harus `Pinjaman.JasaPerBulan` yang tersimpan saat pengajuan. Jangan hitung ulang dari config saat pencairan.

Respons sukses memakai `200`:

```json
{"id_pinjaman": 1, "tanggal_cair": "2026-11-20", "jumlah_jadwal": 12}
```

Selesai jika:

- pinjaman tenor 12 menghasilkan 12 baris jadwal
- total pokok jadwal sama dengan jumlah pinjaman
- pencairan kedua mengembalikan `409`
- kegagalan `InsertBatch` me-rollback update tanggal cair

## Langkah 4: pelunasan dipercepat

File:

- `repository/pelunasan.go`
- `service/pelunasan.go`
- `service/pelunasan_test.go`
- `handler/pelunasan.go`

### Fungsi hitung murni

```go
HitungRincianPelunasan(
    sisaPokok int64,
    jasaBulanIni int64,
    persenAdmin float64,
) models.RincianPelunasan
```

Gunakan `util.PersenDari(sisaPokok, persenAdmin)`.

Tes:

- belum ada pembayaran
- sudah bayar sebagian
- pembulatan administrasi 2 persen

### Repository

Method:

- `Insert`
- `GetByIDPinjaman`
- `TotalSisaPokok(tx, idPinjaman)`
- `JasaBulanBerjalan(tx, idPinjaman)`
- `TandaiBelumLunas(tx, idPinjaman)`

Aturan hitungan:

```text
sisa_pokok      = SUM(jadwal.pokok) - SUM(pembayaran.pokok_dibayar)
jasa_bulan_ini  = sisa jasa pada jadwal Belum dengan angsuran_ke terkecil
administrasi    = util.PersenDari(sisa_pokok, ADMINISTRASI_PELUNASAN_PERSEN)
total           = sisa_pokok + jasa_bulan_ini + administrasi
```

Jika tidak ada jadwal `Belum`, jasa bulan berjalan bernilai `0`.

### Service

Fungsi:

- `HitungPelunasan(idPinjaman int64, tanggal time.Time) (models.RincianPelunasan, error)`
- `ProsesPelunasan(idPinjaman int64, tanggal time.Time) (models.Pelunasan, error)`
- `GetPelunasan(idPinjaman int64) (*models.Pelunasan, error)`

Validasi:

- pinjaman ada
- status `Aktif`
- `tanggal_cair` tidak `NULL`

Urutan `ProsesPelunasan` dalam satu transaksi:

1. lock pinjaman dengan `GetByIDForUpdate`
2. validasi ulang status dan tanggal cair
3. tolak jika sudah ada baris `pelunasan`
4. hitung ulang dengan query yang memakai `tx`
5. insert `pelunasan`
6. tandai semua jadwal `Belum` menjadi `Lunas`
7. ubah status pinjaman menjadi `Lunas`
8. commit

Pelunasan kedua ditolak dengan `ErrStatusPinjaman`.

Denda tidak masuk ke rumus pelunasan. Denda adalah bagian Brian.

### Endpoint

- `GET /api/pinjaman/{id}/pelunasan/hitung?tanggal=`
- `POST /api/pinjaman/{id}/pelunasan`

Keputusan yang disarankan:

- POST memakai body `{"tanggal": "YYYY-MM-DD"}`.
- GET memakai `tanggal` dari query. Jika kosong, default hari ini.

Selesai jika:

- tes rumus murni lulus
- pelunasan menutup pinjaman dan semua jadwal
- pelunasan kedua mengembalikan `409`
- tidak ada logika denda di modul ini

Catatan implementasi:

- `HitungPelunasan` membaca pinjaman tanpa lock; `ProsesPelunasan` membaca ulang dengan `GetByIDForUpdate` di dalam transaksi.
- `GetByIDPinjaman` memakai `DBTX` agar pengecekan pelunasan kedua ikut terkunci dalam transaksi yang sama.
- `GetPelunasan` mengembalikan `nil` saat belum ada baris pelunasan.
- POST memakai body `{"tanggal": "YYYY-MM-DD"}`; GET tanpa `tanggal` memakai tanggal hari ini.

## Langkah 5: kartu pinjaman

File:

- `repository/laporan_kartu.go`
- `service/laporan_kartu.go`
- `handler/laporan_kartu.go`

File ini sengaja dipisah dari `laporan.go` milik Brian.

Endpoint:

```text
GET /api/laporan/kartu-pinjaman/{id_pinjaman}
```

Path value yang dipakai adalah `id_pinjaman`, jadi gunakan:

```go
r.PathValue("id_pinjaman")
```

Data yang dirakit:

- data pinjaman
- nama dan nomor anggota
- daftar jadwal angsuran
- pembayaran per angsuran, yaitu denda, jasa, dan pokok yang sudah dibayar
- data pelunasan jika ada
- sisa pokok
- status pinjaman

Untuk `models.JadwalDetail`:

- `DendaTerbayar`, `JasaTerbayar`, `PokokTerbayar` diambil dari tabel `pembayaran`.
- `SisaJasa` dan `SisaPokok` dihitung dari jadwal dikurangi pembayaran.
- `SisaDenda` diset `0` dalam scope ini, karena denda adalah perhitungan berjalan milik Brian dan tidak disimpan sebagai tagihan tersendiri.

Tidak perlu menunggu service pembayaran Brian. Modul ini hanya membaca tabel `pembayaran`.

Selesai jika:

- pinjaman tidak ada menghasilkan `404`
- jumlah baris jadwal sesuai tenor
- pembayaran dan pelunasan muncul dengan benar
- sisa pokok cocok dengan hitungan manual

Catatan implementasi:

- `SisaDenda` diset `0` pada setiap jadwal; denda berjalan tetap milik modul Brian.
- `SisaPokok` dihitung dari `jadwal.pokok - pokok_dibayar` per angsuran, lalu dijumlahkan.
- Jika data pelunasan ada, `SisaPokok` kartu diset `0` karena pelunasan menutup pinjaman tanpa membuat baris `pembayaran`.
- `PinjamanRepository.GetDetailByID` dan `PelunasanRepository.GetByIDPinjaman` dipakai ulang lewat interface kecil di service.

## Langkah 6: wiring `main.go`

File:

- `main.go`

Perubahan:

- passing `cfg` ke constructor service Samudra
- membuat repository, service, dan handler Samudra
- memanggil tiga `RegisterRoutes(mux)` dari handler Samudra

Tidak boleh:

- mengubah route modul lain
- menambah auth check
- mengubah logika Gema

## Peta file yang akan dibuat atau diubah

| File | Status | Langkah |
| --- | --- | --- |
| `service/pencairan.go` | baru | 1, 3 |
| `service/pencairan_test.go` | baru | 1 |
| `repository/pinjaman.go` | baru | 2 |
| `service/pinjaman.go` | baru | 2 |
| `service/pinjaman_test.go` | baru | 2 |
| `handler/pinjaman.go` | baru | 2, 3 |
| `repository/jadwal.go` | baru | 3 |
| `repository/pelunasan.go` | baru | 4 |
| `service/pelunasan.go` | baru | 4 |
| `service/pelunasan_test.go` | baru | 4 |
| `handler/pelunasan.go` | baru | 4 |
| `repository/laporan_kartu.go` | baru | 5 |
| `service/laporan_kartu.go` | baru | 5 |
| `handler/laporan_kartu.go` | baru | 5 |
| `service/errors.go` | diubah bila perlu | 2 |
| `main.go` | diubah | 6 |

## Verifikasi

Setiap langkah:

```bash
go build ./...
go vet ./...
go test ./...
```

Verifikasi manual memerlukan MySQL dan izin menjalankan server:

1. tambah anggota
2. setor simpanan pokok dan wajib
3. cek batas pinjaman
4. ajukan pinjaman
5. cairkan pinjaman
6. cek jumlah jadwal sama dengan tenor
7. siapkan data pembayaran lewat Brian atau insert manual untuk pengujian
8. hitung pelunasan
9. proses pelunasan
10. cek pinjaman dan semua jadwal berstatus `Lunas`
11. buka kartu pinjaman

Verifikasi manual dilakukan dengan MariaDB 13.0.2 pada database isolasi `pemweb_samudra_test`: pinjaman `600000` dengan tenor 6, pembayaran sebagian `40000`, hitung pelunasan `560000 + 0 + 11200 = 571200`, pelunasan kedua ditolak `409`, pinjaman dan enam jadwal berstatus `Lunas`, kartu pinjaman menampilkan pembayaran dan pelunasan.

## Keputusan yang perlu dikonfirmasi

- [x] Branch: `feat/pinjaman-samudra` dengan commit per langkah; tujuh commit dipindah ke branch ini dan `main` dikembalikan ke `14fb9da`.
- [x] Body `POST /api/pinjaman/{id}/pelunasan`: `{"tanggal": "YYYY-MM-DD"}` (bukan query).
- [x] `tanggal` kosong pada endpoint hitung pelunasan: default hari ini.
- [x] Respons batas pinjaman: `{"batas_maksimal": ..., "pinjaman_aktif": ...}`.
- [x] `SisaDenda` di kartu pinjaman: diset `0`.

## Checklist progres

- [x] Langkah 0: baseline
- [x] Langkah 1: `HitungJadwal` dan tes
- [x] Langkah 2: pengajuan pinjaman dan batas pinjaman
- [x] Langkah 3: pencairan dan jadwal angsuran
- [x] Langkah 4: pelunasan dipercepat
- [x] Langkah 5: kartu pinjaman
- [x] Langkah 6: wiring `main.go`
- [x] Verifikasi statis
- [x] Verifikasi manual dengan MySQL
