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

func TestHitungRincianPelunasan(t *testing.T) {
	tests := []struct {
		name        string
		sisaPokok   int64
		jasa        int64
		persenAdmin float64
		want        models.RincianPelunasan
	}{
		{
			name:        "belum ada pembayaran",
			sisaPokok:   1200000,
			jasa:        12000,
			persenAdmin: 2,
			want: models.RincianPelunasan{
				SisaPokok:    1200000,
				Jasa:         12000,
				Administrasi: 24000,
				Total:        1236000,
			},
		},
		{
			name:        "sudah bayar sebagian",
			sisaPokok:   600000,
			jasa:        12000,
			persenAdmin: 2,
			want: models.RincianPelunasan{
				SisaPokok:    600000,
				Jasa:         12000,
				Administrasi: 12000,
				Total:        624000,
			},
		},
		{
			name:        "pembulatan administrasi 2 persen",
			sisaPokok:   333333,
			jasa:        0,
			persenAdmin: 2,
			want: models.RincianPelunasan{
				SisaPokok:    333333,
				Jasa:         0,
				Administrasi: 6667,
				Total:        340000,
			},
		},
		{
			name:        "persen administrasi nol",
			sisaPokok:   500000,
			jasa:        5000,
			persenAdmin: 0,
			want: models.RincianPelunasan{
				SisaPokok:    500000,
				Jasa:         5000,
				Administrasi: 0,
				Total:        505000,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := HitungRincianPelunasan(test.sisaPokok, test.jasa, test.persenAdmin)
			if got != test.want {
				t.Errorf("rincian = %+v, ingin %+v", got, test.want)
			}
		})
	}
}

type pelunasanCallRecorder struct {
	calls []string
}

func (r *pelunasanCallRecorder) add(name string) {
	if r != nil {
		r.calls = append(r.calls, name)
	}
}

type fakePelunasanPinjamanRepo struct {
	pinjaman   models.Pinjaman
	getIDErr   error
	lockErr    error
	statusErr  error
	lastID     int64
	lastStatus string
	recorder   *pelunasanCallRecorder
}

func (f *fakePelunasanPinjamanRepo) GetByID(id int64) (models.Pinjaman, error) {
	f.recorder.add("get")
	return f.pinjaman, f.getIDErr
}

func (f *fakePelunasanPinjamanRepo) GetByIDForUpdate(_ repository.DBTX, id int64) (models.Pinjaman, error) {
	f.recorder.add("lock")
	return f.pinjaman, f.lockErr
}

func (f *fakePelunasanPinjamanRepo) UpdateStatus(_ repository.DBTX, id int64, status string) error {
	f.recorder.add("status")
	f.lastID = id
	f.lastStatus = status
	return f.statusErr
}

type fakePelunasanDataRepo struct {
	existing  models.Pelunasan
	getErr    error
	sisaPokok int64
	sisaErr   error
	jasa      int64
	jasaErr   error
	insertErr error
	tandaiErr error
	inserted  models.Pelunasan
	lastDBTX  repository.DBTX
	recorder  *pelunasanCallRecorder
}

func (f *fakePelunasanDataRepo) Insert(db repository.DBTX, pelunasan *models.Pelunasan) error {
	f.recorder.add("insert")
	f.lastDBTX = db
	if f.insertErr != nil {
		return f.insertErr
	}
	pelunasan.IDPelunasan = 42
	f.inserted = *pelunasan
	return nil
}

func (f *fakePelunasanDataRepo) GetByIDPinjaman(db repository.DBTX, idPinjaman int64) (models.Pelunasan, error) {
	f.recorder.add("cek")
	f.lastDBTX = db
	return f.existing, f.getErr
}

func (f *fakePelunasanDataRepo) TotalSisaPokok(db repository.DBTX, idPinjaman int64) (int64, error) {
	f.recorder.add("sisa")
	f.lastDBTX = db
	return f.sisaPokok, f.sisaErr
}

func (f *fakePelunasanDataRepo) JasaBulanBerjalan(db repository.DBTX, idPinjaman int64) (int64, error) {
	f.recorder.add("jasa")
	f.lastDBTX = db
	return f.jasa, f.jasaErr
}

func (f *fakePelunasanDataRepo) TandaiBelumLunas(db repository.DBTX, idPinjaman int64) error {
	f.recorder.add("tandai")
	f.lastDBTX = db
	return f.tandaiErr
}

type fakePelunasanTxRunner struct {
	ran        bool
	committed  bool
	rolledBack bool
	recorder   *pelunasanCallRecorder
}

func (r *fakePelunasanTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	r.ran = true
	r.recorder.add("tx")
	if err := fn(fakeDBTX{}); err != nil {
		r.rolledBack = true
		return err
	}
	r.committed = true
	return nil
}

func pinjamanUntukPelunasan() models.Pinjaman {
	cair := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	return models.Pinjaman{
		IDPinjaman:       9,
		IDAnggota:        7,
		TanggalPengajuan: time.Date(2026, time.October, 25, 0, 0, 0, 0, time.UTC),
		TanggalCair:      &cair,
		JumlahPinjaman:   1200000,
		Tenor:            12,
		JasaPerBulan:     12000,
		Status:           "Aktif",
	}
}

func newPelunasanServiceUntukTes(
	recorder *pelunasanCallRecorder,
	runner *fakePelunasanTxRunner,
	pinjamanRepo *fakePelunasanPinjamanRepo,
	dataRepo *fakePelunasanDataRepo,
) *PelunasanService {
	return &PelunasanService{
		pinjamanRepository:  pinjamanRepo,
		pelunasanRepository: dataRepo,
		persenAdministrasi:  2,
		txRunner:            runner,
	}
}

func TestHitungPelunasan(t *testing.T) {
	tanggal := time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC)
	lunas := pinjamanUntukPelunasan()
	lunas.Status = "Lunas"
	belumCair := pinjamanUntukPelunasan()
	belumCair.TanggalCair = nil
	queryErr := errors.New("gagal membaca sisa pokok")

	tests := []struct {
		name        string
		id          int64
		tanggal     time.Time
		pinjaman    models.Pinjaman
		getIDErr    error
		sisaPokok   int64
		sisaErr     error
		wantErr     error
		wantRincian models.RincianPelunasan
	}{
		{
			name:     "belum ada pembayaran",
			id:       9,
			tanggal:  tanggal,
			pinjaman: pinjamanUntukPelunasan(),
			wantRincian: models.RincianPelunasan{
				IDPinjaman: 9, SisaPokok: 1200000, Jasa: 12000, Administrasi: 24000, Total: 1236000,
			},
		},
		{
			name:      "sudah bayar sebagian",
			id:        9,
			tanggal:   tanggal,
			pinjaman:  pinjamanUntukPelunasan(),
			sisaPokok: 450000,
			wantRincian: models.RincianPelunasan{
				IDPinjaman: 9, SisaPokok: 450000, Jasa: 12000, Administrasi: 9000, Total: 471000,
			},
		},
		{
			name:     "pinjaman tidak ditemukan",
			id:       9,
			tanggal:  tanggal,
			pinjaman: pinjamanUntukPelunasan(),
			getIDErr: sql.ErrNoRows,
			wantErr:  ErrPinjamanTidakDitemukan,
		},
		{
			name:     "status lunas",
			id:       9,
			tanggal:  tanggal,
			pinjaman: lunas,
			wantErr:  ErrStatusPinjaman,
		},
		{
			name:     "belum dicairkan",
			id:       9,
			tanggal:  tanggal,
			pinjaman: belumCair,
			wantErr:  ErrStatusPinjaman,
		},
		{
			name:     "query gagal",
			id:       9,
			tanggal:  tanggal,
			pinjaman: pinjamanUntukPelunasan(),
			sisaErr:  queryErr,
			wantErr:  queryErr,
		},
		{
			name:    "id tidak valid",
			id:      0,
			tanggal: tanggal,
			wantErr: ErrInputPinjaman,
		},
		{
			name:    "tanggal kosong",
			id:      9,
			wantErr: ErrTanggalTidakValid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &pelunasanCallRecorder{}
			runner := &fakePelunasanTxRunner{recorder: recorder}
			pinjamanRepo := &fakePelunasanPinjamanRepo{
				pinjaman: test.pinjaman,
				getIDErr: test.getIDErr,
				recorder: recorder,
			}
			sisaPokok := test.sisaPokok
			if sisaPokok == 0 {
				sisaPokok = 1200000
			}
			dataRepo := &fakePelunasanDataRepo{
				sisaPokok: sisaPokok,
				jasa:      12000,
				sisaErr:   test.sisaErr,
				recorder:  recorder,
			}
			service := newPelunasanServiceUntukTes(recorder, runner, pinjamanRepo, dataRepo)

			got, err := service.HitungPelunasan(test.id, test.tanggal)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, ingin %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("HitungPelunasan error = %v", err)
			}
			if got != test.wantRincian {
				t.Errorf("rincian = %+v, ingin %+v", got, test.wantRincian)
			}
			if runner.ran {
				t.Error("hitung tidak boleh membuka transaksi")
			}
		})
	}
}

func TestProsesPelunasan(t *testing.T) {
	tanggal := time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC)
	insertErr := errors.New("gagal insert pelunasan")
	tandaiErr := errors.New("gagal menandai jadwal")
	statusErr := errors.New("gagal update status")
	sisaErr := errors.New("gagal menghitung sisa pokok")

	lunas := pinjamanUntukPelunasan()
	lunas.Status = "Lunas"
	belumCair := pinjamanUntukPelunasan()
	belumCair.TanggalCair = nil

	tests := []struct {
		name         string
		id           int64
		tanggal      time.Time
		pinjaman     models.Pinjaman
		lockErr      error
		getErr       error
		sisaErr      error
		insertErr    error
		tandaiErr    error
		statusErr    error
		wantErr      error
		wantCalls    []string
		wantCommit   bool
		wantRollback bool
	}{
		{
			name:       "pelunasan berhasil",
			id:         9,
			tanggal:    tanggal,
			pinjaman:   pinjamanUntukPelunasan(),
			getErr:     sql.ErrNoRows,
			wantCalls:  []string{"tx", "lock", "cek", "sisa", "jasa", "insert", "tandai", "status"},
			wantCommit: true,
		},
		{
			name:         "pelunasan kedua ditolak",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     pinjamanUntukPelunasan(),
			getErr:       nil,
			wantErr:      ErrStatusPinjaman,
			wantCalls:    []string{"tx", "lock", "cek"},
			wantRollback: true,
		},
		{
			name:         "status lunas",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     lunas,
			getErr:       sql.ErrNoRows,
			wantErr:      ErrStatusPinjaman,
			wantCalls:    []string{"tx", "lock"},
			wantRollback: true,
		},
		{
			name:         "belum dicairkan",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     belumCair,
			getErr:       sql.ErrNoRows,
			wantErr:      ErrStatusPinjaman,
			wantCalls:    []string{"tx", "lock"},
			wantRollback: true,
		},
		{
			name:         "pinjaman tidak ditemukan",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     pinjamanUntukPelunasan(),
			lockErr:      sql.ErrNoRows,
			getErr:       sql.ErrNoRows,
			wantErr:      ErrPinjamanTidakDitemukan,
			wantCalls:    []string{"tx", "lock"},
			wantRollback: true,
		},
		{
			name:         "hitung sisa pokok gagal",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     pinjamanUntukPelunasan(),
			getErr:       sql.ErrNoRows,
			sisaErr:      sisaErr,
			wantErr:      sisaErr,
			wantCalls:    []string{"tx", "lock", "cek", "sisa"},
			wantRollback: true,
		},
		{
			name:         "insert gagal",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     pinjamanUntukPelunasan(),
			getErr:       sql.ErrNoRows,
			insertErr:    insertErr,
			wantErr:      insertErr,
			wantCalls:    []string{"tx", "lock", "cek", "sisa", "jasa", "insert"},
			wantRollback: true,
		},
		{
			name:         "tandai jadwal gagal",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     pinjamanUntukPelunasan(),
			getErr:       sql.ErrNoRows,
			tandaiErr:    tandaiErr,
			wantErr:      tandaiErr,
			wantCalls:    []string{"tx", "lock", "cek", "sisa", "jasa", "insert", "tandai"},
			wantRollback: true,
		},
		{
			name:         "update status gagal",
			id:           9,
			tanggal:      tanggal,
			pinjaman:     pinjamanUntukPelunasan(),
			getErr:       sql.ErrNoRows,
			statusErr:    statusErr,
			wantErr:      statusErr,
			wantCalls:    []string{"tx", "lock", "cek", "sisa", "jasa", "insert", "tandai", "status"},
			wantRollback: true,
		},
		{
			name:    "id tidak valid",
			id:      0,
			tanggal: tanggal,
			getErr:  sql.ErrNoRows,
			wantErr: ErrInputPinjaman,
		},
		{
			name:    "tanggal kosong",
			id:      9,
			getErr:  sql.ErrNoRows,
			wantErr: ErrTanggalTidakValid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &pelunasanCallRecorder{}
			runner := &fakePelunasanTxRunner{recorder: recorder}
			pinjamanRepo := &fakePelunasanPinjamanRepo{
				pinjaman:  test.pinjaman,
				lockErr:   test.lockErr,
				statusErr: test.statusErr,
				recorder:  recorder,
			}
			dataRepo := &fakePelunasanDataRepo{
				getErr:    test.getErr,
				sisaPokok: 600000,
				jasa:      12000,
				sisaErr:   test.sisaErr,
				insertErr: test.insertErr,
				tandaiErr: test.tandaiErr,
				recorder:  recorder,
			}
			service := newPelunasanServiceUntukTes(recorder, runner, pinjamanRepo, dataRepo)

			got, err := service.ProsesPelunasan(test.id, test.tanggal)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, ingin %v", err, test.wantErr)
				}
			} else if err != nil {
				t.Fatalf("ProsesPelunasan error = %v", err)
			}
			if runner.ran != (len(test.wantCalls) > 0) {
				t.Errorf("transaksi dijalankan = %v, ingin %v", runner.ran, len(test.wantCalls) > 0)
			}
			if runner.committed != test.wantCommit {
				t.Errorf("transaksi di-commit = %v, ingin %v", runner.committed, test.wantCommit)
			}
			if runner.rolledBack != test.wantRollback {
				t.Errorf("transaksi di-rollback = %v, ingin %v", runner.rolledBack, test.wantRollback)
			}
			if !reflect.DeepEqual(recorder.calls, test.wantCalls) {
				t.Errorf("urutan panggilan = %v, ingin %v", recorder.calls, test.wantCalls)
			}

			if test.wantErr == nil {
				want := models.Pelunasan{
					IDPelunasan:  42,
					IDPinjaman:   9,
					Tanggal:      tanggal,
					SisaPokok:    600000,
					Jasa:         12000,
					Administrasi: 12000,
					Total:        624000,
				}
				if got != want {
					t.Errorf("pelunasan = %+v, ingin %+v", got, want)
				}
				if pinjamanRepo.lastID != test.id || pinjamanRepo.lastStatus != "Lunas" {
					t.Errorf("update status = (%d, %q), ingin (%d, %q)",
						pinjamanRepo.lastID, pinjamanRepo.lastStatus, test.id, "Lunas")
				}
				if dataRepo.inserted != want {
					t.Errorf("data yang diinsert = %+v, ingin %+v", dataRepo.inserted, want)
				}
				if dataRepo.lastDBTX == nil {
					t.Error("query pelunasan tidak memakai tx")
				}
			}
		})
	}
}

func TestGetPelunasan(t *testing.T) {
	tanggal := time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC)
	want := models.Pelunasan{
		IDPelunasan: 42, IDPinjaman: 9, Tanggal: tanggal,
		SisaPokok: 600000, Jasa: 12000, Administrasi: 12000, Total: 624000,
	}
	queryErr := errors.New("gagal membaca pelunasan")

	tests := []struct {
		name    string
		id      int64
		getErr  error
		wantErr error
		wantNil bool
	}{
		{name: "ditemukan", id: 9},
		{name: "belum ada pelunasan", id: 9, getErr: sql.ErrNoRows, wantNil: true},
		{name: "query gagal", id: 9, getErr: queryErr, wantErr: queryErr},
		{name: "id tidak valid", id: 0, wantErr: ErrInputPinjaman},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataRepo := &fakePelunasanDataRepo{existing: want, getErr: test.getErr}
			service := &PelunasanService{
				pinjamanRepository:  &fakePelunasanPinjamanRepo{},
				pelunasanRepository: dataRepo,
				persenAdministrasi:  2,
			}

			got, err := service.GetPelunasan(test.id)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, ingin %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetPelunasan error = %v", err)
			}
			if test.wantNil {
				if got != nil {
					t.Errorf("pelunasan = %+v, ingin nil", got)
				}
				return
			}
			if got == nil || *got != want {
				t.Errorf("pelunasan = %+v, ingin %+v", got, want)
			}
		})
	}
}
