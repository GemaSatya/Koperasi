package service

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

type kartuCallRecorder struct {
	calls []string
}

func (r *kartuCallRecorder) add(name string) {
	r.calls = append(r.calls, name)
}

type fakeKartuPinjamanRepo struct {
	detail   models.PinjamanDetail
	err      error
	recorder *kartuCallRecorder
}

func (f *fakeKartuPinjamanRepo) GetDetailByID(id int64) (models.PinjamanDetail, error) {
	f.recorder.add("pinjaman")
	return f.detail, f.err
}

type fakeKartuJadwalRepo struct {
	jadwal     []models.JadwalDetail
	pembayaran []models.Pembayaran
	jadwalErr  error
	bayarErr   error
	recorder   *kartuCallRecorder
}

func (f *fakeKartuJadwalRepo) ListDetailByPinjaman(idPinjaman int64) ([]models.JadwalDetail, error) {
	f.recorder.add("jadwal")
	return f.jadwal, f.jadwalErr
}

func (f *fakeKartuJadwalRepo) ListPembayaranByPinjaman(idPinjaman int64) ([]models.Pembayaran, error) {
	f.recorder.add("pembayaran")
	return f.pembayaran, f.bayarErr
}

type fakeKartuPelunasanRepo struct {
	pelunasan models.Pelunasan
	err       error
	recorder  *kartuCallRecorder
}

func (f *fakeKartuPelunasanRepo) GetByIDPinjaman(db repository.DBTX, idPinjaman int64) (models.Pelunasan, error) {
	f.recorder.add("pelunasan")
	return f.pelunasan, f.err
}

func pinjamanDetailUntukKartu() models.PinjamanDetail {
	cair := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	return models.PinjamanDetail{
		Pinjaman: models.Pinjaman{
			IDPinjaman:       9,
			IDAnggota:        7,
			TanggalPengajuan: time.Date(2026, time.October, 25, 0, 0, 0, 0, time.UTC),
			TanggalCair:      &cair,
			JumlahPinjaman:   1200000,
			Tenor:            3,
			JasaPerBulan:     10000,
			Status:           "Aktif",
		},
		NamaAnggota: "Sari",
		NoAnggota:   "A-007",
	}
}

func jadwalUntukKartu() []models.JadwalDetail {
	return []models.JadwalDetail{
		{
			JadwalAngsuran: models.JadwalAngsuran{
				IDJadwal: 1, IDPinjaman: 9, AngsuranKe: 1,
				JatuhTempo: time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC),
				Pokok:      400000, Jasa: 10000, Status: "Belum",
			},
			DendaTerbayar: 5000, JasaTerbayar: 10000, PokokTerbayar: 200000,
			SisaJasa: 0, SisaPokok: 200000,
		},
		{
			JadwalAngsuran: models.JadwalAngsuran{
				IDJadwal: 2, IDPinjaman: 9, AngsuranKe: 2,
				JatuhTempo: time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
				Pokok:      400000, Jasa: 10000, Status: "Belum",
			},
			SisaJasa: 10000, SisaPokok: 400000,
		},
		{
			JadwalAngsuran: models.JadwalAngsuran{
				IDJadwal: 3, IDPinjaman: 9, AngsuranKe: 3,
				JatuhTempo: time.Date(2027, time.February, 1, 0, 0, 0, 0, time.UTC),
				Pokok:      400000, Jasa: 10000, Status: "Belum",
			},
			SisaJasa: 10000, SisaPokok: 400000,
		},
	}
}

func pelunasanUntukKartu() models.Pelunasan {
	return models.Pelunasan{
		IDPelunasan: 42, IDPinjaman: 9,
		Tanggal:      time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC),
		SisaPokok:    1000000,
		Jasa:         10000,
		Administrasi: 20000,
		Total:        1030000,
	}
}

func TestKartuPinjaman(t *testing.T) {
	jadwalErr := errors.New("gagal membaca jadwal")
	bayarErr := errors.New("gagal membaca pembayaran")
	pelunasanErr := errors.New("gagal membaca pelunasan")
	pembayaran := []models.Pembayaran{{
		IDPembayaran: 5, IDJadwal: 1,
		TanggalBayar: time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC),
		TotalBayar:   215000, DendaDibayar: 5000, JasaDibayar: 10000, PokokDibayar: 200000,
	}}

	tests := []struct {
		name          string
		id            int64
		detailErr     error
		jadwalErr     error
		bayarErr      error
		pelunasanErr  error
		adaPelunasan  bool
		wantErr       error
		wantCalls     []string
		wantSisaPokok int64
	}{
		{
			name:          "kartu tanpa pelunasan",
			id:            9,
			pelunasanErr:  sql.ErrNoRows,
			wantCalls:     []string{"pinjaman", "jadwal", "pembayaran", "pelunasan"},
			wantSisaPokok: 1000000,
		},
		{
			name:          "kartu dengan pelunasan",
			id:            9,
			adaPelunasan:  true,
			wantCalls:     []string{"pinjaman", "jadwal", "pembayaran", "pelunasan"},
			wantSisaPokok: 0,
		},
		{
			name:      "pinjaman tidak ditemukan",
			id:        9,
			detailErr: sql.ErrNoRows,
			wantErr:   ErrPinjamanTidakDitemukan,
			wantCalls: []string{"pinjaman"},
		},
		{
			name:      "jadwal gagal dibaca",
			id:        9,
			jadwalErr: jadwalErr,
			wantErr:   jadwalErr,
			wantCalls: []string{"pinjaman", "jadwal"},
		},
		{
			name:      "pembayaran gagal dibaca",
			id:        9,
			bayarErr:  bayarErr,
			wantErr:   bayarErr,
			wantCalls: []string{"pinjaman", "jadwal", "pembayaran"},
		},
		{
			name:         "pelunasan gagal dibaca",
			id:           9,
			pelunasanErr: pelunasanErr,
			wantErr:      pelunasanErr,
			wantCalls:    []string{"pinjaman", "jadwal", "pembayaran", "pelunasan"},
		},
		{
			name:      "id tidak valid",
			id:        0,
			wantErr:   ErrInputPinjaman,
			wantCalls: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &kartuCallRecorder{calls: make([]string, 0)}
			pinjamanRepo := &fakeKartuPinjamanRepo{
				detail: pinjamanDetailUntukKartu(), err: test.detailErr, recorder: recorder,
			}
			jadwalRepo := &fakeKartuJadwalRepo{
				jadwal: jadwalUntukKartu(), pembayaran: pembayaran,
				jadwalErr: test.jadwalErr, bayarErr: test.bayarErr, recorder: recorder,
			}
			pelunasanRepo := &fakeKartuPelunasanRepo{
				pelunasan: pelunasanUntukKartu(), err: test.pelunasanErr, recorder: recorder,
			}
			service := &LaporanKartuService{
				pinjamanRepository:  pinjamanRepo,
				jadwalRepository:    jadwalRepo,
				pelunasanRepository: pelunasanRepo,
			}

			got, err := service.KartuPinjaman(test.id)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, ingin %v", err, test.wantErr)
				}
			} else if err != nil {
				t.Fatalf("KartuPinjaman error = %v", err)
			}
			if len(recorder.calls) != len(test.wantCalls) {
				t.Fatalf("urutan panggilan = %v, ingin %v", recorder.calls, test.wantCalls)
			}
			for i, call := range test.wantCalls {
				if recorder.calls[i] != call {
					t.Fatalf("urutan panggilan = %v, ingin %v", recorder.calls, test.wantCalls)
				}
			}
			if test.wantErr != nil {
				return
			}

			if got.SisaPokok != test.wantSisaPokok {
				t.Errorf("sisa pokok = %d, ingin %d", got.SisaPokok, test.wantSisaPokok)
			}
			if len(got.Jadwal) != 3 {
				t.Errorf("jumlah jadwal = %d, ingin 3", len(got.Jadwal))
			}
			if len(got.Pembayaran) != 1 || got.Pembayaran[0].IDPembayaran != 5 {
				t.Errorf("pembayaran = %+v, ingin satu baris dengan id 5", got.Pembayaran)
			}
			if got.Pinjaman.NamaAnggota != "Sari" || got.Pinjaman.NoAnggota != "A-007" {
				t.Errorf("data anggota = %+v, ingin Sari/A-007", got.Pinjaman)
			}
			if test.adaPelunasan {
				if got.Pelunasan == nil || *got.Pelunasan != pelunasanUntukKartu() {
					t.Errorf("pelunasan = %+v, ingin %+v", got.Pelunasan, pelunasanUntukKartu())
				}
			} else if got.Pelunasan != nil {
				t.Errorf("pelunasan = %+v, ingin nil", got.Pelunasan)
			}
		})
	}
}
