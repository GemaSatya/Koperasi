# Pembagian Tugas Backend: Sistem Informasi Koperasi Simpan Pinjam

Tim backend (Kelompok 2 PemWeb): **Gema, Samudra, Brian**.
Teknologi: **Golang 1.22+**, **MySQL**, `net/http`, `database/sql` (tanpa framework dan tanpa ORM).
Proyek ini hanya untuk tugas akhir mata kuliah, jadi **tidak ada deploy**. Cukup bisa dijalankan di komputer sendiri.

Dokumen ini berisi dua hal:

1. Langkah kerja tiap orang.
2. Prompt yang bisa disalin kalau ingin meminta bantuan LLM (Claude, ChatGPT, dan sejenisnya).

---

## 1. Ringkasan Pembagian

|                 | Gema                            | Samudra                        | Brian                        |
| --------------- | ------------------------------- | ------------------------------ | ---------------------------- |
| Fokus           | Dasar proyek, Anggota, Simpanan | Pinjaman, Pencairan, Pelunasan | Pembayaran angsuran, Laporan |
| Jumlah endpoint | 11                              | 8                              | 4 pembayaran + 5 laporan     |
| Bagian tersulit | `TarikSukarela`                 | `CairkanPinjaman`              | `BayarAngsuran`              |

---

## 2. Aturan Kerja Bersama

1. **Gema mengerjakan dasar proyek lebih dulu** (config, db, util, model, error). Selesai, lalu langsung di-merge ke `main`.
2. Samudra dan Brian bisa mulai lebih awal dengan fungsi hitung murni yang tidak butuh database (`HitungJadwal`, `HitungDenda`, `HitungPelunasan`) dan tesnya.
3. Satu branch untuk satu modul, contoh: `feat/anggota`, `feat/pinjaman`, `feat/pembayaran`.
4. `main` hanya diubah lewat pull request. Jangan commit file `.env`.
5. Agar `main.go` tidak sering bentrok, tiap orang membuat fungsi `RegisterRoutes(mux)` di handler miliknya. Gema hanya memanggilnya di `main.go`.
6. Pembagian lapisan kode:
   - `handler`: hanya baca JSON dan kirim response. Tidak ada SQL.
   - `service`: semua aturan bisnis dan hitungan. Tidak ada `http`.
   - `repository`: hanya SQL.
7. Semua uang memakai `int64` (jangan `float`). Semua query SQL memakai tanda `?` (jangan menyambung string).

---

## 3. Cara Memakai Prompt

1. Buka chat baru di LLM.
2. **Lampirkan dua file**: `INSTRUKSI_BACKEND_KOPERASI.md` dan file SQL database (`projekpemweb`).
3. Tempel **Prompt Konteks** (bagian 4) sebagai pesan pertama.
4. Tempel **prompt tugas** sesuai langkah yang sedang dikerjakan (bagian 5, 6, 7, atau 8).
5. Minta satu tugas dalam satu waktu. Jangan minta semuanya sekaligus agar hasilnya rapi dan mudah dicek.
6. Setelah kode dibuat, jalankan `go build ./...`, `go vet ./...`, dan `go test ./...`.

---

## 4. Prompt Konteks (tempel paling awal, untuk semua orang)

```text
Kamu adalah asisten programmer Golang. Saya mengerjakan backend Sistem Informasi Koperasi Simpan Pinjam untuk tugas akhir kuliah (tidak akan di-deploy).

Saya melampirkan dua file:
1. INSTRUKSI_BACKEND_KOPERASI.md : panduan lengkap (aturan bisnis, struktur proyek, daftar function, endpoint).
2. File SQL database projekpemweb : skema tabel yang sebenarnya.

Aturan yang harus kamu ikuti:
- Bahasa: Go 1.22 atau lebih baru. HTTP memakai net/http standar (ServeMux dengan pola "GET /path/{id}", ambil id dengan r.PathValue("id")).
- Driver: github.com/go-sql-driver/mysql. Akses DB memakai database/sql. Jangan pakai ORM atau framework.
- Struktur folder: cmd/server/main.go, internal/{config,db,model,repository,service,handler,middleware,util}.
- Pembagian lapisan: handler hanya decode JSON dan memanggil service, service berisi aturan bisnis tanpa package http, repository hanya berisi SQL dan menerima db.DBTX agar bisa dipakai dengan atau tanpa transaksi.
- Semua nominal uang memakai int64, jangan float. Persentase dihitung dengan integer lalu dibulatkan.
- Semua query SQL memakai placeholder "?", jangan menyambung string.
- Format response: {"success": true, "data": {...}, "message": "OK"} atau {"success": false, "error": "..."}.
- Kode HTTP: 200/201 sukses, 400 validasi gagal, 401 login gagal, 404 tidak ditemukan, 409 konflik aturan bisnis, 500 error server.
- Operasi yang mengubah lebih dari satu tabel wajib memakai transaksi (RunInTx) dan SELECT ... FOR UPDATE bila perlu.
- Nama tabel dan kolom harus sama persis dengan file SQL. Catatan: tabel admin memakai kolom password_hash (bukan password) dan ada kolom nama.
- Beri komentar singkat berbahasa Indonesia pada bagian yang penting.

Jawab dengan kode lengkap per file (tulis nama file di atas tiap blok kode). Jika ada yang kurang jelas, tanyakan dulu satu pertanyaan singkat. Jangan menambah fitur di luar yang diminta.

Mengerti? Tunggu tugas saya berikutnya.
```

---

## 5. Gema: Dasar Proyek, Anggota, Simpanan

### Langkah-langkah

1. Siapkan dasar proyek: `config`, `db` (koneksi + `RunInTx`), `util`, CORS, `main.go`, `.env.example`, dan file migrasi SQL.
2. Buat struct data (`model`) dan daftar error bersama.
3. Buat fitur Login (1 endpoint).
4. Buat fitur Anggota (5 endpoint).
5. Buat fitur Simpanan dan penarikan sukarela (5 endpoint).
6. Tulis tes untuk fungsi bantu tanggal dan persen.

### Prompt Langkah 1: Dasar proyek

```text
Tugas: buat dasar proyek backend sesuai bagian 2, 5, dan 6.0 pada file instruksi.

Yang dibuat:
1. go.mod dan perintah setup (go mod init koperasi, go get driver mysql dan godotenv).
2. internal/config/config.go : baca env APP_PORT, DB_DSN, BATAS_PINJAMAN_KALI, JASA_PERSEN_PER_BULAN, DENDA_PERSEN_PER_HARI, ADMINISTRASI_PELUNASAN_PERSEN (pakai godotenv jika ada file .env). Beri nilai default yang sesuai file instruksi.
3. internal/db/db.go : interface DBTX, fungsi Open, dan RunInTx (rollback jika error atau panic).
4. internal/util : AddMonths (jepit akhir bulan), HariTerlambat (minimal 0), PersenDari (int64, dibulatkan), ParseTanggal (YYYY-MM-DD), WriteJSON, WriteError (petakan error bisnis ke kode 400/404/409/500).
5. internal/middleware/cors.go.
6. cmd/server/main.go : buka koneksi, buat ServeMux, pasang CORS, jalankan server. Sediakan tempat memanggil RegisterRoutes(mux) dari tiap handler (Auth, Anggota, Simpanan, Pinjaman, Pembayaran, Pelunasan, Laporan), tulis sebagai komentar TODO jika handlernya belum ada.
7. .env.example.
8. migrations/001_schema.sql : salin dari file SQL yang saya lampirkan, jangan diubah.
```

### Prompt Langkah 2: Model dan error

```text
Tugas: buat internal/model dan daftar error bersama.

1. Satu struct untuk tiap tabel di file SQL (Anggota, Simpanan, Penarikan, Pinjaman, Jadwal, Pembayaran, Pelunasan, Admin) dengan tag json snake_case. Nominal pakai int64, tanggal pakai time.Time (atau *time.Time untuk kolom yang boleh NULL seperti tanggal_cair).
2. Struct tambahan yang disebut di file instruksi: AnggotaInput, SimpananInput, PinjamanInput, RingkasanSimpanan, PinjamanDetail, JadwalDetail, Tagihan, RincianPelunasan, FilterLaporan, KartuPinjaman, BarisSimpanan, BarisPembayaran, Tunggakan, PendapatanJasa, PendapatanDenda.
3. internal/service/errors.go berisi variabel error seperti pada bagian 5.2 file instruksi.

Struct Admin tidak boleh mengirim kolom password_hash ke JSON (pakai tag json:"-").
```

### Prompt Langkah 3: Login

```text
Tugas: buat fitur Login (bagian 6.1 file instruksi).

File: repository/admin.go, service/auth.go, handler/auth.go.
- POST /api/login dengan body {"username": "...", "password": "..."}.
- Cocokkan username dan password pada tabel admin. Kolom di database bernama password_hash, bandingkan langsung (tanpa hashing, ini hanya tugas kuliah).
- Berhasil: kembalikan id_admin, username, dan nama (tanpa password).
- Gagal: ErrLoginGagal dengan kode 401.
- Validasi: username dan password wajib diisi (400 jika kosong).
- Sertakan fungsi RegisterRoutes(mux).
```

### Prompt Langkah 4: Anggota

```text
Tugas: buat fitur Anggota (bagian 6.2 dan 7 file instruksi).

File: repository/anggota.go, service/anggota.go, handler/anggota.go.
Endpoint:
- GET /api/anggota?keyword=&status=&page=&limit= (cari nama, no_anggota, no_hp; ada paginasi dan total)
- POST /api/anggota
- GET /api/anggota/{id}
- PUT /api/anggota/{id} (ubah nama, alamat, no_hp)
- PATCH /api/anggota/{id}/nonaktif (status menjadi Keluar, ditolak jika masih punya pinjaman Aktif)

Aturan:
- no_anggota dibuat otomatis, format AGT-TAHUN-0001 (nomor urut terakhir tahun berjalan, pakai CountByYear).
- Status awal Aktif, tanggal_daftar default hari ini.
- Validasi: nama wajib maksimal 100 karakter, no_hp opsional angka 8 sampai 15 digit, id path harus angka positif (400 jika tidak).
- Anggota tidak dihapus fisik.
- Sertakan RegisterRoutes(mux).
```

### Prompt Langkah 5: Simpanan

```text
Tugas: buat fitur Simpanan dan penarikan sukarela (bagian 6.3 dan 7 file instruksi).

File: repository/simpanan.go, repository/penarikan.go, service/simpanan.go, handler/simpanan.go.
Endpoint:
- POST /api/simpanan
- GET /api/anggota/{id}/simpanan?jenis=
- GET /api/anggota/{id}/simpanan/ringkasan
- POST /api/penarikan-sukarela
- GET /api/anggota/{id}/penarikan

Aturan:
- Anggota harus berstatus Aktif.
- nominal bilangan bulat lebih dari 0, jenis harus Pokok, Wajib, atau Sukarela.
- Simpanan Pokok hanya boleh satu kali per anggota (ErrSimpananPokokAda).
- Penarikan hanya untuk Sukarela, maksimal sebesar saldo sukarela (ErrSaldoTidakCukup).
- saldo_sukarela = total setoran sukarela dikurangi total penarikan.
- TarikSukarela wajib memakai RunInTx dan penguncian agar saldo tidak minus saat dua request bersamaan.
- Ringkasan berisi total Pokok, Wajib, Sukarela, penarikan, saldo sukarela, dan grand total.
- Sertakan RegisterRoutes(mux).
```

### Prompt Langkah 6: Tes util

```text
Tugas: buat file util_test.go untuk package util.

Uji:
- AddMonths: 31 Jan + 1 bulan, 29 Feb tahun kabisat, lintas tahun.
- HariTerlambat: tepat jatuh tempo (0), terlambat 1 hari, dibayar lebih awal (0).
- PersenDari: contoh pembulatan (misalnya 1% dari 6.000.000, dan 2% dari angka yang menghasilkan pecahan).
Gunakan table-driven test.
```

---

## 6. Samudra: Pinjaman, Pencairan, Pelunasan

> Catatan: kode Samudra memakai `db`, `model`, dan `util` buatan Gema. Jika belum di-merge, minta LLM menganggap package tersebut sudah ada dengan nama function seperti di file instruksi.

### Langkah-langkah

1. Buat fungsi hitung murni `HitungJadwal` dan tesnya (tidak perlu menunggu Gema).
2. Buat fitur Pengajuan Pinjaman (4 endpoint).
3. Buat fitur Pencairan (1 endpoint).
4. Buat fitur Pelunasan Dipercepat (2 endpoint).
5. Buat laporan Kartu Pinjaman (1 endpoint).
6. Tulis tes untuk `HitungPelunasan`.

### Prompt Langkah 1: HitungJadwal

```text
Tugas: buat fungsi murni HitungJadwal (bagian 6.5 file instruksi) beserta unit test.

Signature: HitungJadwal(jumlah int64, tenor int, jasaPerBulan int64, tanggalCair time.Time) []model.Jadwal
- Jumlah baris sama dengan tenor.
- pokok per angsuran = jumlah / tenor, sisa pembulatan masuk ke angsuran terakhir.
- jasa sama setiap bulan.
- jatuh_tempo ke-n = util.AddMonths(tanggalCair, n).
- status awal "Belum".

Tes: jumlah habis dibagi tenor, jumlah tidak habis dibagi tenor (cek sisa di angsuran terakhir), total pokok semua baris harus sama dengan jumlah, tenor 1, tanggal cair 31 Januari.
```

### Prompt Langkah 2: Pengajuan pinjaman

```text
Tugas: buat fitur Pengajuan Pinjaman (bagian 4, 6.4, dan 7 file instruksi).

File: repository/pinjaman.go, service/pinjaman.go, handler/pinjaman.go.
Endpoint:
- GET /api/anggota/{id}/batas-pinjaman (hasil HitungBatasMaksimal dan CekPinjamanAktif)
- POST /api/pinjaman
- GET /api/pinjaman?status=&id_anggota=
- GET /api/pinjaman/{id}

Aturan AjukanPinjaman, cek berurutan:
1. Anggota Aktif (ErrAnggotaTidakAktif).
2. Tenor 1 sampai 60.
3. Tidak ada pinjaman berstatus Aktif (ErrPinjamanAktifAda).
4. jumlah <= batas maksimal (ErrMelebihiBatas).
Lalu insert dengan status Aktif, tanggal_cair NULL, dan jasa_per_bulan terisi.

Rumus:
batas_maksimal = BATAS_PINJAMAN_KALI x (total pokok + total wajib + saldo sukarela)
jasa_per_bulan = jumlah x JASA_PERSEN_PER_BULAN / 100

Repository minimal: Insert, GetByID, GetByIDForUpdate, List, ExistsAktifByAnggota, UpdateStatus, UpdateCair.
Sertakan RegisterRoutes(mux). Gunakan fungsi saldo dan total simpanan dari service Simpanan (buat interface kecil jika package belum ada).
```

### Prompt Langkah 3: Pencairan

```text
Tugas: buat fitur Pencairan pinjaman (bagian 6.5 file instruksi).

File: service/pencairan.go, repository/jadwal.go (InsertBatch), tambahan di handler/pinjaman.go.
Endpoint: POST /api/pinjaman/{id}/cairkan dengan body {"tanggal_cair": "YYYY-MM-DD"}.

CairkanPinjaman harus dalam SATU transaksi (RunInTx):
1. Kunci baris pinjaman (SELECT ... FOR UPDATE).
2. Pastikan status Aktif dan tanggal_cair masih NULL, jika tidak kembalikan ErrStatusPinjaman (409).
3. Simpan tanggal_cair.
4. Buat jadwal angsuran lewat BuatJadwalAngsuran(tx, pinjaman) yang memanggil HitungJadwal lalu InsertBatch.
Pinjaman 12 bulan harus menghasilkan 12 baris jadwal.
```

### Prompt Langkah 4: Pelunasan

```text
Tugas: buat fitur Pelunasan Dipercepat (bagian 6.7 file instruksi).

File: repository/pelunasan.go, service/pelunasan.go, handler/pelunasan.go.
Endpoint:
- GET /api/pinjaman/{id}/pelunasan/hitung?tanggal=
- POST /api/pinjaman/{id}/pelunasan

Rincian hitungan:
- sisa_pokok = total pokok semua jadwal dikurangi total pokok_dibayar
- jasa = sisa jasa pada jadwal berstatus Belum dengan angsuran_ke terkecil (satu bulan jasa)
- administrasi = sisa_pokok x ADMINISTRASI_PELUNASAN_PERSEN / 100
- total = sisa_pokok + jasa + administrasi

ProsesPelunasan dalam satu transaksi:
1. Kunci pinjaman (FOR UPDATE), harus Aktif dan sudah cair (jika tidak, ErrStatusPinjaman).
2. Hitung ulang di dalam transaksi.
3. Insert ke tabel pelunasan.
4. Ubah semua jadwal Belum menjadi Lunas.
5. Ubah status pinjaman menjadi Lunas.
Pelunasan kedua kali harus ditolak. Buat juga GetPelunasan (mengembalikan nil jika belum ada).
Sertakan RegisterRoutes(mux).
```

### Prompt Langkah 5: Kartu Pinjaman

```text
Tugas: buat laporan Kartu Pinjaman (bagian 6.8 file instruksi).

Endpoint: GET /api/laporan/kartu-pinjaman/{id_pinjaman}
Isi response: data pinjaman beserta nama anggota, daftar jadwal angsuran, pembayaran tiap angsuran (pokok, jasa, denda yang sudah dibayar), sisa pokok, status, dan data pelunasan jika ada.
Letakkan di service/laporan_kartu.go dan handler/laporan_kartu.go (file terpisah dari laporan milik Brian agar tidak bentrok). Sertakan RegisterRoutes(mux). Kembalikan 404 jika pinjaman tidak ada.
```

### Prompt Langkah 6: Tes HitungPelunasan

```text
Tugas: buat unit test untuk perhitungan pelunasan.

Pisahkan rumus ke fungsi murni (tanpa DB), misalnya HitungRincianPelunasan(sisaPokok, jasaBulanIni int64, persenAdmin float64) RincianPelunasan, lalu uji:
- belum ada pembayaran sama sekali
- sudah bayar sebagian
- pembulatan administrasi 2%
Pinjaman berstatus Lunas harus ditolak (uji di level service dengan data tiruan jika memungkinkan).
```

---

## 7. Brian: Pembayaran Angsuran dan Laporan

### Langkah-langkah

1. Buat fungsi hitung murni `HitungDenda` dan alokasi pembayaran, beserta tesnya (tidak perlu menunggu Gema).
2. Buat fitur Pembayaran Angsuran (4 endpoint).
3. Buat laporan Simpanan dan Riwayat Pembayaran (2 endpoint).
4. Buat laporan Daftar Tunggakan (1 endpoint).
5. Buat laporan Pendapatan Jasa dan Pendapatan Denda (2 endpoint).

### Prompt Langkah 1: HitungDenda dan alokasi

```text
Tugas: buat dua fungsi murni (tanpa DB) beserta unit test, sesuai bagian 4 dan 6.6 file instruksi.

1. HitungDenda(sisaTagihan int64, hariTerlambat int) int64
   denda = sisaTagihan x DENDA_PERSEN_PER_HARI / 100 x hariTerlambat. Hari terlambat 0 berarti denda 0.

2. AlokasiPembayaran(nominal, dendaSisa, jasaSisa, pokokSisa int64) (denda, jasa, pokok int64, err error)
   Urutan: denda dulu, lalu jasa, lalu pokok. Tolak (ErrNominalMelebihiTagihan) jika nominal <= 0 atau nominal lebih besar dari dendaSisa + jasaSisa + pokokSisa.

Tes: tepat jatuh tempo, telat 1 hari, telat 30 hari, bayar kurang dari denda, pas denda + jasa, pas lunas, kelebihan (ditolak).
```

### Prompt Langkah 2: Pembayaran angsuran

```text
Tugas: buat fitur Pembayaran Angsuran (bagian 6.6 dan 7 file instruksi).

File: repository/pembayaran.go, repository/jadwal.go (bagian baca: GetByID, GetByIDForUpdate, ListByPinjaman, UpdateStatus, CountUnpaid; koordinasi dengan Samudra yang membuat InsertBatch di file yang sama, atau pisahkan ke jadwal_read.go), service/pembayaran.go, handler/pembayaran.go.
Endpoint:
- GET /api/pinjaman/{id}/jadwal (jadwal + terbayar + sisa per baris)
- GET /api/jadwal/{id}/tagihan?tanggal= (sisa pokok, sisa jasa, denda, total)
- POST /api/jadwal/{id}/bayar dengan body {"tanggal_bayar": "YYYY-MM-DD", "nominal": 300000}
- GET /api/pinjaman/{id}/pembayaran

Algoritma BayarAngsuran (satu transaksi):
1. Kunci jadwal dan pinjaman (FOR UPDATE). Tolak jika pinjaman bukan Aktif atau jadwal sudah Lunas.
2. pokokSisa = pokok jadwal dikurangi total pokok_dibayar. jasaSisa dengan cara sama.
   dendaSisa = HitungDenda(pokokSisa + jasaSisa, hariTerlambat) dikurangi total denda_dibayar (minimal 0).
3. Tolak jika nominal <= 0 atau lebih besar dari total tagihan.
4. Alokasi: denda, lalu jasa, lalu pokok.
5. Insert ke tabel pembayaran.
6. UpdateStatusJadwal (Lunas jika pokok dan jasa penuh), lalu CekDanTutupPinjaman (pinjaman Lunas jika semua jadwal Lunas).
Response bayar: id_pembayaran, total_bayar, denda_dibayar, jasa_dibayar, pokok_dibayar, status_jadwal.
Gunakan fungsi HitungDenda dan AlokasiPembayaran dari langkah 1. Sertakan RegisterRoutes(mux).
```

### Prompt Langkah 3: Laporan Simpanan dan Riwayat Pembayaran

```text
Tugas: buat dua laporan (bagian 6.8 file instruksi).

File: repository/laporan.go, service/laporan.go, handler/laporan.go.
1. GET /api/laporan/simpanan?dari=&sampai=&id_anggota=
   Per anggota: total Pokok, Wajib, Sukarela, penarikan, dan saldo sukarela. Filter periode dan anggota bersifat opsional.
2. GET /api/laporan/riwayat-pembayaran?dari=&sampai=&id_anggota=
   Daftar pembayaran (JOIN pembayaran, jadwal_angsuran, pinjaman, anggota) berisi nama anggota, angsuran ke, tanggal bayar, denda, jasa, pokok, total.

Gunakan struct FilterLaporan (Dari, Sampai, IDAnggota opsional). Semua query memakai placeholder "?". Sertakan RegisterRoutes(mux).
```

### Prompt Langkah 4: Daftar Tunggakan

```text
Tugas: buat laporan Daftar Tunggakan (bagian 6.8 file instruksi).

Endpoint: GET /api/laporan/tunggakan?tanggal= (default hari ini).
Ambil jadwal berstatus Belum dengan jatuh_tempo < tanggal, pada pinjaman Aktif. Gunakan query dari bagian 6.8 file instruksi (subquery SUM pembayaran per jadwal).
Hasil per baris: nama anggota, no_anggota, id_pinjaman, angsuran_ke, jatuh_tempo, sisa pokok, sisa jasa, hari terlambat, denda berjalan.
Denda berjalan dihitung di Go memakai HitungDenda (bukan di SQL) agar rumusnya hanya ada satu tempat.
```

### Prompt Langkah 5: Pendapatan Jasa dan Denda

```text
Tugas: buat dua laporan pendapatan (bagian 6.8 file instruksi).

1. GET /api/laporan/pendapatan-jasa?dari=&sampai=
   Total = SUM(pembayaran.jasa_dibayar) ditambah SUM(pelunasan.jasa) pada periode, dengan rincian per bulan (format YYYY-MM).
2. GET /api/laporan/pendapatan-denda?dari=&sampai=
   Total = SUM(pembayaran.denda_dibayar) pada periode, dengan rincian per bulan.
Bulan tanpa data boleh tidak muncul. Validasi: dari dan sampai wajib diisi, format YYYY-MM-DD, dari tidak boleh lebih besar dari sampai (400).
```

---

## 8. Dikerjakan Bertiga di Akhir

### Langkah-langkah

1. Gabungkan semua branch, pastikan `go build ./...` berhasil.
2. Uji alur lengkap lewat Postman atau curl:
   `login` > `tambah anggota` > `setor pokok dan wajib` > `ajukan pinjaman` > `cairkan` > `bayar angsuran (sebagian lalu lunas)` > `pelunasan dipercepat` > `buka semua laporan`
3. Cocokkan angka laporan dengan hitungan manual.
4. Rapikan pesan error dan validasi input.
5. Tulis `README.md`: cara menjalankan (`go run ./cmd/server`), isi `.env`, dan contoh request tiap endpoint.

### Prompt: Review kode (tempel kode modul yang mau dicek)

```text
Tolong review kode Golang berikut untuk modul [NAMA MODUL] sesuai file instruksi yang saya lampirkan.

Periksa:
- Apakah ada query SQL yang menyambung string (rawan SQL injection)?
- Apakah ada nominal yang memakai float?
- Apakah operasi multi-tabel sudah memakai transaksi dan FOR UPDATE?
- Apakah nama tabel dan kolom sama persis dengan file SQL?
- Apakah kode HTTP dan format response sudah sesuai bagian 5.2?
- Apakah ada aturan bisnis di file instruksi yang belum terpenuhi?

Beri daftar masalah berurutan dari yang paling penting, lalu kode perbaikannya. Jangan ubah hal yang sudah benar.

[TEMPEL KODE DI SINI]
```

### Prompt: Menulis README dan contoh request

```text
Tugas: tulis README.md berbahasa Indonesia untuk backend ini.

Isi:
1. Penjelasan singkat proyek.
2. Cara menjalankan di komputer sendiri (instal Go dan MySQL, import SQL, isi .env, go run ./cmd/server).
3. Tabel seluruh endpoint (method, path, fungsi).
4. Contoh request dan response JSON untuk tiap endpoint, termasuk contoh response error.
Bahasa sederhana, tanpa em dash. Tidak perlu bagian deploy.

Daftar endpoint ada di bagian 7 file instruksi.
```

---

## 9. Tips Memakai Hasil dari LLM

1. **Jangan percaya begitu saja.** Selalu jalankan `go build`, `go vet`, dan `go test`.
2. **Cek nama kolom.** LLM kadang mengarang nama kolom. Bandingkan dengan file SQL.
3. **Uji dengan data nyata** di MySQL, terutama bagian denda, pembayaran sebagian, dan pelunasan.
4. **Satu tugas satu chat** (atau minimal satu tugas satu pesan) supaya jawaban tidak terpotong dan mudah direvisi.
5. **Kalau hasilnya salah**, tempel pesan error atau hasil yang salah lalu jelaskan yang diharapkan. Jangan hanya bilang "tidak jalan".
6. **Pahami kodenya.** Saat presentasi, setiap anggota harus bisa menjelaskan modul miliknya sendiri.
