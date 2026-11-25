package service

import "errors"

var (
	ErrAnggotaTidakDitemukan = errors.New("anggota tidak ditemukan")
	ErrAnggotaTidakAktif     = errors.New("anggota tidak aktif")
	ErrInputAnggota          = errors.New("data anggota tidak valid")
	ErrAnggotaMasihMeminjam  = errors.New("anggota masih memiliki pinjaman aktif")
	ErrStatusAnggota         = errors.New("status anggota tidak valid")

	ErrJenisSimpanan     = errors.New("jenis simpanan tidak valid")
	ErrNominalTidakValid = errors.New("nominal harus lebih dari 0")
	ErrTanggalTidakValid = errors.New("tanggal tidak valid")
	ErrSaldoTidakCukup   = errors.New("saldo simpanan sukarela tidak cukup")
	ErrSimpananPokokAda  = errors.New("simpanan pokok sudah pernah dibayar")

	ErrInputPinjaman          = errors.New("data pinjaman tidak valid")
	ErrPinjamanTidakDitemukan = errors.New("pinjaman tidak ditemukan")
	ErrTenorTidakValid        = errors.New("tenor harus antara 1 dan 60 bulan")
	ErrJumlahTidakValid       = errors.New("jumlah pinjaman harus lebih dari 0")
	ErrJadwalTidakValid       = errors.New("jadwal angsuran tidak valid")
	ErrPinjamanAktifAda       = errors.New("anggota masih memiliki pinjaman aktif")
	ErrMelebihiBatas          = errors.New("jumlah pinjaman melebihi batas maksimal")
	ErrStatusPinjaman         = errors.New("status pinjaman tidak sesuai untuk proses ini")
	ErrNominalMelebihiTagihan = errors.New("nominal melebihi total tagihan")
	ErrJadwalTidakDitemukan   = errors.New("jadwal angsuran tidak ditemukan")
	ErrJadwalLunas            = errors.New("jadwal angsuran sudah lunas")
	ErrPeriodeTidakValid      = errors.New("periode laporan tidak valid")
	ErrLoginGagal             = errors.New("username atau password salah")
)
