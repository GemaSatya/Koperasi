package service

import (
	"database/sql"
	"errors"
	"time"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type PembayaranService struct {
	db             *sql.DB
	pembayaranRepo *repository.PembayaranRepository
	jadwalRepo     *repository.JadwalRepository
	pinjamanRepo   *repository.PinjamanRepository
	pelunasanRepo  *repository.PelunasanRepository
	dendaPersen    float64
}

func NewPembayaranService(
	db *sql.DB,
	pembayaranRepo *repository.PembayaranRepository,
	jadwalRepo *repository.JadwalRepository,
	pinjamanRepo *repository.PinjamanRepository,
	pelunasanRepo *repository.PelunasanRepository,
	cfg config.Config,
) *PembayaranService {
	return &PembayaranService{
		db:             db,
		pembayaranRepo: pembayaranRepo,
		jadwalRepo:     jadwalRepo,
		pinjamanRepo:   pinjamanRepo,
		pelunasanRepo:  pelunasanRepo,
		dendaPersen:    cfg.DendaPersenPerHari,
	}
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func clampMin0(nilai int64) int64 {
	if nilai < 0 {
		return 0
	}
	return nilai
}

// HitungDenda menghitung denda keterlambatan. Fungsi murni, tanpa database.
func HitungDenda(sisaTagihan int64, hariTerlambat int, persenPerHari float64) int64 {
	if sisaTagihan <= 0 || hariTerlambat <= 0 || persenPerHari <= 0 {
		return 0
	}
	return util.PersenDari(sisaTagihan, persenPerHari) * int64(hariTerlambat)
}

// AlokasiPembayaran membagi uang pembayaran secara berurutan:
// denda, lalu jasa, lalu pokok. Fungsi murni, tanpa database.
func AlokasiPembayaran(nominal, dendaSisa, jasaSisa, pokokSisa int64) (denda, jasa, pokok int64, err error) {
	dendaSisa = clampMin0(dendaSisa)
	jasaSisa = clampMin0(jasaSisa)
	pokokSisa = clampMin0(pokokSisa)

	if nominal <= 0 {
		return 0, 0, 0, ErrNominalTidakValid
	}
	if nominal > dendaSisa+jasaSisa+pokokSisa {
		return 0, 0, 0, ErrNominalMelebihiTagihan
	}

	denda = min64(nominal, dendaSisa)
	sisa := nominal - denda
	jasa = min64(sisa, jasaSisa)
	pokok = min64(sisa-jasa, pokokSisa)
	return denda, jasa, pokok, nil
}

// hitungTagihan menghitung sisa pokok, sisa jasa, dan denda yang masih harus dibayar.
func (s *PembayaranService) hitungTagihan(db repository.DBTX, jadwal models.JadwalAngsuran, tanggalAcuan time.Time) (models.Tagihan, error) {
	dendaTerbayar, jasaTerbayar, pokokTerbayar, err := s.pembayaranRepo.SumByJadwal(db, jadwal.IDJadwal)
	if err != nil {
		return models.Tagihan{}, err
	}

	sisaPokok := clampMin0(jadwal.Pokok - pokokTerbayar)
	sisaJasa := clampMin0(jadwal.Jasa - jasaTerbayar)
	hari := util.HariTerlambat(jadwal.JatuhTempo, tanggalAcuan)
	dendaTerutang := clampMin0(
		HitungDenda(sisaPokok+sisaJasa, hari, s.dendaPersen) - dendaTerbayar,
	)

	return models.Tagihan{
		IDJadwal:      jadwal.IDJadwal,
		SisaPokok:     sisaPokok,
		SisaJasa:      sisaJasa,
		DendaTerutang: dendaTerutang,
		Total:         dendaTerutang + sisaJasa + sisaPokok,
	}, nil
}

func (s *PembayaranService) GetJadwal(idPinjaman int64) ([]models.JadwalDetail, error) {
	if idPinjaman <= 0 {
		return nil, ErrInputAnggota
	}
	if _, err := s.pinjamanRepo.GetByID(s.db, idPinjaman); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinjamanTidakDitemukan
		}
		return nil, err
	}
	return s.jadwalRepo.ListDetailByPinjaman(s.db, idPinjaman)
}

func (s *PembayaranService) HitungTagihan(idJadwal int64, tanggalAcuan time.Time) (models.Tagihan, error) {
	if idJadwal <= 0 || tanggalAcuan.IsZero() {
		return models.Tagihan{}, ErrInputAnggota
	}
	jadwal, err := s.jadwalRepo.GetByID(s.db, idJadwal)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Tagihan{}, ErrJadwalTidakDitemukan
	}
	if err != nil {
		return models.Tagihan{}, err
	}
	return s.hitungTagihan(s.db, jadwal, tanggalAcuan)
}

// BayarAngsuran memproses pembayaran penuh maupun sebagian dalam satu transaksi.
// Nilai yang dikembalikan: pembayaran, status jadwal terbaru, dan error.
func (s *PembayaranService) BayarAngsuran(idJadwal int64, tanggalBayar time.Time, nominal int64) (models.Pembayaran, string, error) {
	if idJadwal <= 0 || tanggalBayar.IsZero() {
		return models.Pembayaran{}, "", ErrInputAnggota
	}

	var pembayaran models.Pembayaran
	var statusJadwal string

	err := models.RunInTx(s.db, func(tx *sql.Tx) error {
		jadwal, err := s.jadwalRepo.GetByIDForUpdate(tx, idJadwal)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrJadwalTidakDitemukan
		}
		if err != nil {
			return err
		}

		pinjaman, err := s.pinjamanRepo.GetByIDForUpdate(tx, jadwal.IDPinjaman)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPinjamanTidakDitemukan
		}
		if err != nil {
			return err
		}
		if pinjaman.Status != "Aktif" {
			return ErrStatusPinjaman
		}
		if jadwal.Status == "Lunas" {
			return ErrJadwalLunas
		}

		tagihan, err := s.hitungTagihan(tx, jadwal, tanggalBayar)
		if err != nil {
			return err
		}
		denda, jasa, pokok, err := AlokasiPembayaran(
			nominal,
			tagihan.DendaTerutang,
			tagihan.SisaJasa,
			tagihan.SisaPokok,
		)
		if err != nil {
			return err
		}

		pembayaran, err = s.pembayaranRepo.Insert(tx, models.Pembayaran{
			IDJadwal:     idJadwal,
			TanggalBayar: tanggalBayar,
			TotalBayar:   nominal,
			DendaDibayar: denda,
			JasaDibayar:  jasa,
			PokokDibayar: pokok,
		})
		if err != nil {
			return err
		}

		statusJadwal, err = s.updateStatusJadwal(tx, jadwal)
		if err != nil {
			return err
		}
		return s.cekDanTutupPinjaman(tx, pinjaman.IDPinjaman)
	})
	if err != nil {
		return models.Pembayaran{}, "", err
	}
	return pembayaran, statusJadwal, nil
}

// updateStatusJadwal menandai Lunas hanya jika pokok dan jasa terbayar penuh.
func (s *PembayaranService) updateStatusJadwal(tx *sql.Tx, jadwal models.JadwalAngsuran) (string, error) {
	_, jasaTerbayar, pokokTerbayar, err := s.pembayaranRepo.SumByJadwal(tx, jadwal.IDJadwal)
	if err != nil {
		return "", err
	}

	status := "Belum"
	if pokokTerbayar >= jadwal.Pokok && jasaTerbayar >= jadwal.Jasa {
		status = "Lunas"
	}
	if err := s.jadwalRepo.UpdateStatus(tx, jadwal.IDJadwal, status); err != nil {
		return "", err
	}
	return status, nil
}

// cekDanTutupPinjaman menutup pinjaman ketika seluruh angsuran sudah lunas.
func (s *PembayaranService) cekDanTutupPinjaman(tx *sql.Tx, idPinjaman int64) error {
	tersisa, err := s.jadwalRepo.CountUnpaid(tx, idPinjaman)
	if err != nil {
		return err
	}
	if tersisa == 0 {
		return s.pinjamanRepo.UpdateStatus(tx, idPinjaman, "Lunas")
	}
	return nil
}

func (s *PembayaranService) HitungSisaPinjaman(idPinjaman int64) (int64, error) {
	if idPinjaman <= 0 {
		return 0, ErrInputAnggota
	}
	jadwal, err := s.jadwalRepo.ListDetailByPinjaman(s.db, idPinjaman)
	if err != nil {
		return 0, err
	}
	var sisa int64
	for _, baris := range jadwal {
		sisa += baris.SisaPokok
	}

	// Pelunasan dipercepat tidak membuat baris pembayaran, jadi pokok yang
	// ditutup lewat pelunasan harus dikurangkan dari sisa.
	pelunasan, err := s.pelunasanRepo.GetByPinjaman(s.db, idPinjaman)
	if err != nil {
		return 0, err
	}
	if pelunasan != nil {
		sisa = clampMin0(sisa - pelunasan.SisaPokok)
	}
	return sisa, nil
}

func (s *PembayaranService) RiwayatPembayaranPinjaman(idPinjaman int64) ([]models.Pembayaran, error) {
	if idPinjaman <= 0 {
		return nil, ErrInputAnggota
	}
	if _, err := s.pinjamanRepo.GetByID(s.db, idPinjaman); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPinjamanTidakDitemukan
		}
		return nil, err
	}
	return s.pembayaranRepo.ListByPinjaman(s.db, idPinjaman)
}
