package service

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

func TestHitungJadwal(t *testing.T) {
	cair := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		jumlah      int64
		tenor       int
		jasa        int64
		tanggalCair time.Time
		wantLen     int
		wantPokok   []int64
		wantJasa    int64
		wantTempo   []time.Time
		wantStatus  string
	}{
		{
			name:        "jumlah habis dibagi tenor",
			jumlah:      6000000,
			tenor:       12,
			jasa:        60000,
			tanggalCair: cair,
			wantLen:     12,
			wantPokok: []int64{
				500000, 500000, 500000, 500000, 500000, 500000,
				500000, 500000, 500000, 500000, 500000, 500000,
			},
			wantJasa: 60000,
			wantTempo: []time.Time{
				time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
				time.Date(2027, time.January, 31, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
		{
			name:        "sisa pembulatan masuk angsuran terakhir",
			jumlah:      1000000,
			tenor:       3,
			jasa:        10000,
			tanggalCair: time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC),
			wantLen:     3,
			wantPokok:   []int64{333333, 333333, 333334},
			wantJasa:    10000,
			wantTempo: []time.Time{
				time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
		{
			name:        "tenor satu",
			jumlah:      750000,
			tenor:       1,
			jasa:        7500,
			tanggalCair: time.Date(2026, time.November, 30, 0, 0, 0, 0, time.UTC),
			wantLen:     1,
			wantPokok:   []int64{750000},
			wantJasa:    7500,
			wantTempo: []time.Time{
				time.Date(2026, time.December, 30, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
		{
			name:        "tanggal cair 31 Januari tahun kabisat",
			jumlah:      120000,
			tenor:       1,
			jasa:        1200,
			tanggalCair: time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC),
			wantLen:     1,
			wantPokok:   []int64{120000},
			wantJasa:    1200,
			wantTempo: []time.Time{
				time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
			},
			wantStatus: "Belum",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := HitungJadwal(test.jumlah, test.tenor, test.jasa, test.tanggalCair)
			if len(got) != test.wantLen {
				t.Fatalf("jumlah baris = %d, ingin %d", len(got), test.wantLen)
			}

			var totalPokok int64
			for i, row := range got {
				totalPokok += row.Pokok

				if row.AngsuranKe != i+1 {
					t.Errorf("angsuran ke-%d = %d, ingin %d", i, row.AngsuranKe, i+1)
				}
				if row.Pokok != test.wantPokok[i] {
					t.Errorf("pokok baris %d = %d, ingin %d", i+1, row.Pokok, test.wantPokok[i])
				}
				if row.Jasa != test.wantJasa {
					t.Errorf("jasa baris %d = %d, ingin %d", i+1, row.Jasa, test.wantJasa)
				}
				if row.Status != test.wantStatus {
					t.Errorf("status baris %d = %q, ingin %q", i+1, row.Status, test.wantStatus)
				}
				if row.IDPinjaman != 0 {
					t.Errorf("IDPinjaman baris %d = %d, ingin 0", i+1, row.IDPinjaman)
				}
			}

			if totalPokok != test.jumlah {
				t.Errorf("total pokok = %d, ingin %d", totalPokok, test.jumlah)
			}

			for i, want := range test.wantTempo {
				index := i
				if len(test.wantTempo) == 2 && i == 1 {
					index = len(got) - 1
				}
				if !got[index].JatuhTempo.Equal(want) {
					t.Errorf("jatuh tempo baris %d = %v, ingin %v", index+1, got[index].JatuhTempo, want)
				}
			}
		})
	}
}

func TestHitungJadwalTenorTidakValid(t *testing.T) {
	for _, tenor := range []int{0, -1} {
		if got := HitungJadwal(100000, tenor, 1000, time.Now()); got != nil {
			t.Errorf("tenor %d menghasilkan %v, ingin nil", tenor, got)
		}
	}
}

type pencairanCallRecorder struct {
	calls []string
}

func (r *pencairanCallRecorder) add(name string) {
	r.calls = append(r.calls, name)
}

type fakePencairanDBTX struct{}

func (fakePencairanDBTX) Exec(string, ...interface{}) (sql.Result, error) { return nil, nil }
func (fakePencairanDBTX) Query(string, ...interface{}) (*sql.Rows, error) { return nil, nil }
func (fakePencairanDBTX) QueryRow(string, ...interface{}) *sql.Row        { return nil }

type fakePencairanPinjamanRepo struct {
	pinjaman  models.Pinjaman
	getErr    error
	updateErr error

	getCalls    int
	updateCalls int
	lastID      int64
	lastTanggal time.Time
	recorder    *pencairanCallRecorder
}

func (f *fakePencairanPinjamanRepo) GetByIDForUpdate(repository.DBTX, int64) (models.Pinjaman, error) {
	f.getCalls++
	if f.recorder != nil {
		f.recorder.add("get")
	}
	return f.pinjaman, f.getErr
}

func (f *fakePencairanPinjamanRepo) UpdateCair(_ repository.DBTX, id int64, tanggal time.Time) error {
	f.updateCalls++
	f.lastID = id
	f.lastTanggal = tanggal
	if f.recorder != nil {
		f.recorder.add("update")
	}
	return f.updateErr
}

type fakeJadwalInsertRepo struct {
	err      error
	calls    int
	inserted []models.JadwalAngsuran
	recorder *pencairanCallRecorder
}

func (f *fakeJadwalInsertRepo) InsertBatch(_ repository.DBTX, jadwal []models.JadwalAngsuran) error {
	f.calls++
	f.inserted = append([]models.JadwalAngsuran(nil), jadwal...)
	if f.recorder != nil {
		f.recorder.add("insert")
	}
	return f.err
}

type fakePencairanTxRunner struct {
	ran        bool
	committed  bool
	rolledBack bool
	recorder   *pencairanCallRecorder
}

func (r *fakePencairanTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	r.ran = true
	if r.recorder != nil {
		r.recorder.add("tx")
	}
	if err := fn(fakePencairanDBTX{}); err != nil {
		r.rolledBack = true
		return err
	}
	r.committed = true
	return nil
}

func pinjamanUntukDicairkan() models.Pinjaman {
	return models.Pinjaman{
		IDPinjaman:       9,
		IDAnggota:        7,
		TanggalPengajuan: time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
		JumlahPinjaman:   1200000,
		Tenor:            12,
		JasaPerBulan:     12000,
		Status:           "Aktif",
	}
}

func TestCairkanPinjaman(t *testing.T) {
	tanggalCair := time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC)
	insertErr := errors.New("gagal insert jadwal")
	updateErr := errors.New("gagal update tanggal cair")
	sudahCair := pinjamanUntukDicairkan()
	sudahCair.TanggalCair = &tanggalCair
	lunas := pinjamanUntukDicairkan()
	lunas.Status = "Lunas"
	tenorKosong := pinjamanUntukDicairkan()
	tenorKosong.Tenor = 0

	tests := []struct {
		name         string
		id           int64
		tanggal      time.Time
		pinjaman     models.Pinjaman
		getErr       error
		updateErr    error
		insertErr    error
		wantErr      error
		wantJadwal   int
		wantTx       bool
		wantCommit   bool
		wantRollback bool
		wantUpdate   int
		wantInsert   int
		wantOrder    []string
	}{
		{
			name:       "pencairan berhasil",
			id:         9,
			tanggal:    tanggalCair,
			pinjaman:   pinjamanUntukDicairkan(),
			wantJadwal: 12,
			wantTx:     true,
			wantCommit: true,
			wantUpdate: 1,
			wantInsert: 1,
			wantOrder:  []string{"tx", "get", "update", "insert"},
		},
		{
			name:         "pinjaman tidak ditemukan",
			id:           9,
			tanggal:      tanggalCair,
			pinjaman:     pinjamanUntukDicairkan(),
			getErr:       sql.ErrNoRows,
			wantErr:      ErrPinjamanTidakDitemukan,
			wantTx:       true,
			wantRollback: true,
			wantOrder:    []string{"tx", "get"},
		},
		{
			name:         "status sudah lunas",
			id:           9,
			tanggal:      tanggalCair,
			pinjaman:     lunas,
			wantErr:      ErrStatusPinjaman,
			wantTx:       true,
			wantRollback: true,
			wantOrder:    []string{"tx", "get"},
		},
		{
			name:         "sudah pernah dicairkan",
			id:           9,
			tanggal:      tanggalCair,
			pinjaman:     sudahCair,
			wantErr:      ErrStatusPinjaman,
			wantTx:       true,
			wantRollback: true,
			wantOrder:    []string{"tx", "get"},
		},
		{
			name:         "data tenor tidak valid",
			id:           9,
			tanggal:      tanggalCair,
			pinjaman:     tenorKosong,
			wantErr:      ErrJadwalTidakValid,
			wantTx:       true,
			wantRollback: true,
			wantOrder:    []string{"tx", "get"},
		},
		{
			name:      "id tidak valid",
			id:        0,
			tanggal:   tanggalCair,
			pinjaman:  pinjamanUntukDicairkan(),
			wantErr:   ErrInputPinjaman,
			wantOrder: []string{},
		},
		{
			name:      "tanggal kosong",
			id:        9,
			tanggal:   time.Time{},
			pinjaman:  pinjamanUntukDicairkan(),
			wantErr:   ErrInputPinjaman,
			wantOrder: []string{},
		},
		{
			name:         "insert jadwal gagal",
			id:           9,
			tanggal:      tanggalCair,
			pinjaman:     pinjamanUntukDicairkan(),
			insertErr:    insertErr,
			wantErr:      insertErr,
			wantTx:       true,
			wantRollback: true,
			wantUpdate:   1,
			wantInsert:   1,
			wantOrder:    []string{"tx", "get", "update", "insert"},
		},
		{
			name:         "update tanggal cair gagal",
			id:           9,
			tanggal:      tanggalCair,
			pinjaman:     pinjamanUntukDicairkan(),
			updateErr:    updateErr,
			wantErr:      updateErr,
			wantTx:       true,
			wantRollback: true,
			wantUpdate:   1,
			wantOrder:    []string{"tx", "get", "update"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &pencairanCallRecorder{calls: make([]string, 0)}
			pinjamanRepo := &fakePencairanPinjamanRepo{
				pinjaman:  test.pinjaman,
				getErr:    test.getErr,
				updateErr: test.updateErr,
				recorder:  recorder,
			}
			jadwalRepo := &fakeJadwalInsertRepo{err: test.insertErr, recorder: recorder}
			runner := &fakePencairanTxRunner{recorder: recorder}
			service := &PencairanService{
				pinjamanRepository: pinjamanRepo,
				jadwalRepository:   jadwalRepo,
				txRunner:           runner,
			}

			jumlah, err := service.CairkanPinjaman(test.id, test.tanggal)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, ingin %v", err, test.wantErr)
				}
			} else if err != nil {
				t.Fatalf("CairkanPinjaman error = %v", err)
			}
			if jumlah != test.wantJadwal {
				t.Errorf("jumlah jadwal = %d, ingin %d", jumlah, test.wantJadwal)
			}
			if runner.ran != test.wantTx {
				t.Errorf("transaksi dijalankan = %v, ingin %v", runner.ran, test.wantTx)
			}
			if runner.committed != test.wantCommit {
				t.Errorf("transaksi di-commit = %v, ingin %v", runner.committed, test.wantCommit)
			}
			if runner.rolledBack != test.wantRollback {
				t.Errorf("transaksi di-rollback = %v, ingin %v", runner.rolledBack, test.wantRollback)
			}
			if pinjamanRepo.getCalls != 1 && test.wantTx {
				t.Errorf("panggilan GetByIDForUpdate = %d, ingin 1", pinjamanRepo.getCalls)
			}
			if pinjamanRepo.updateCalls != test.wantUpdate {
				t.Errorf("panggilan UpdateCair = %d, ingin %d", pinjamanRepo.updateCalls, test.wantUpdate)
			}
			if jadwalRepo.calls != test.wantInsert {
				t.Errorf("panggilan InsertBatch = %d, ingin %d", jadwalRepo.calls, test.wantInsert)
			}
			if !reflect.DeepEqual(recorder.calls, test.wantOrder) {
				t.Errorf("urutan panggilan = %v, ingin %v", recorder.calls, test.wantOrder)
			}

			if test.wantUpdate > 0 {
				if pinjamanRepo.lastID != test.id {
					t.Errorf("id yang di-update = %d, ingin %d", pinjamanRepo.lastID, test.id)
				}
				if !pinjamanRepo.lastTanggal.Equal(test.tanggal) {
					t.Errorf("tanggal yang di-update = %v, ingin %v", pinjamanRepo.lastTanggal, test.tanggal)
				}
			}

			if test.name == "pencairan berhasil" {
				if len(jadwalRepo.inserted) != 12 {
					t.Fatalf("baris jadwal = %d, ingin 12", len(jadwalRepo.inserted))
				}
				var totalPokok int64
				for i, row := range jadwalRepo.inserted {
					totalPokok += row.Pokok
					if row.IDPinjaman != 9 {
						t.Errorf("IDPinjaman baris %d = %d, ingin 9", i+1, row.IDPinjaman)
					}
					if row.Jasa != 12000 {
						t.Errorf("jasa baris %d = %d, ingin 12000", i+1, row.Jasa)
					}
					if row.Status != "Belum" {
						t.Errorf("status baris %d = %q, ingin Belum", i+1, row.Status)
					}
				}
				if totalPokok != test.pinjaman.JumlahPinjaman {
					t.Errorf("total pokok = %d, ingin %d", totalPokok, test.pinjaman.JumlahPinjaman)
				}
			}
		})
	}
}
