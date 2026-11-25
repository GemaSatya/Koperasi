package service

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

type fakeDBTX struct{}

func (fakeDBTX) Exec(string, ...interface{}) (sql.Result, error) { return nil, nil }
func (fakeDBTX) Query(string, ...interface{}) (*sql.Rows, error) { return nil, nil }
func (fakeDBTX) QueryRow(string, ...interface{}) *sql.Row        { return nil }

type pinjamanCallRecorder struct {
	calls []string
}

func (r *pinjamanCallRecorder) add(name string) {
	r.calls = append(r.calls, name)
}

type fakeTxRunner struct {
	ran      bool
	recorder *pinjamanCallRecorder
}

func (r *fakeTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	r.ran = true
	if r.recorder != nil {
		r.recorder.add("tx")
	}
	return fn(fakeDBTX{})
}

type fakeAnggotaLookup struct {
	anggota  models.Anggota
	err      error
	calls    int
	recorder *pinjamanCallRecorder
}

func (f *fakeAnggotaLookup) GetByID(int64) (models.Anggota, error) {
	f.calls++
	if f.recorder != nil {
		f.recorder.add("anggota")
	}
	return f.anggota, f.err
}

type fakeSimpananLookup struct {
	summary  models.RingkasanSimpanan
	err      error
	calls    int
	recorder *pinjamanCallRecorder
}

func (f *fakeSimpananLookup) TotalSimpanan(int64) (models.RingkasanSimpanan, error) {
	f.calls++
	if f.recorder != nil {
		f.recorder.add("simpanan")
	}
	return f.summary, f.err
}

type fakePinjamanRepo struct {
	inserted []models.Pinjaman

	exists     bool
	existsErr  error
	lockErr    error
	detail     models.PinjamanDetail
	detailErr  error
	list       []models.PinjamanDetail
	listErr    error
	count      int
	countErr   error
	lastStatus string
	lastID     int64
	lastOffset int
	lastLimit  int

	lockCalls   int
	existsCalls int
	insertCalls int
	listCalls   int
	countCalls  int
	recorder    *pinjamanCallRecorder
}

func (f *fakePinjamanRepo) Insert(_ repository.DBTX, pinjaman *models.Pinjaman) error {
	f.insertCalls++
	pinjaman.IDPinjaman = 42
	f.inserted = append(f.inserted, *pinjaman)
	return nil
}

func (f *fakePinjamanRepo) GetDetailByID(int64) (models.PinjamanDetail, error) {
	return f.detail, f.detailErr
}

func (f *fakePinjamanRepo) List(status string, idAnggota int64, offset, limit int) ([]models.PinjamanDetail, error) {
	f.listCalls++
	f.lastStatus = status
	f.lastID = idAnggota
	f.lastOffset = offset
	f.lastLimit = limit
	return f.list, f.listErr
}

func (f *fakePinjamanRepo) Count(status string, idAnggota int64) (int, error) {
	f.countCalls++
	f.lastStatus = status
	f.lastID = idAnggota
	return f.count, f.countErr
}

func (f *fakePinjamanRepo) ExistsAktifByAnggota(repository.DBTX, int64) (bool, error) {
	f.existsCalls++
	return f.exists, f.existsErr
}

func (f *fakePinjamanRepo) LockAnggota(repository.DBTX, int64) error {
	f.lockCalls++
	if f.recorder != nil {
		f.recorder.add("lock")
	}
	return f.lockErr
}

func newPinjamanTestService(
	repo *fakePinjamanRepo,
	anggota *fakeAnggotaLookup,
	simpanan *fakeSimpananLookup,
	cfg config.Config,
) (*PinjamanService, *fakeTxRunner, *pinjamanCallRecorder) {
	recorder := &pinjamanCallRecorder{}
	runner := &fakeTxRunner{recorder: recorder}
	repo.recorder = recorder
	anggota.recorder = recorder
	simpanan.recorder = recorder

	service := &PinjamanService{
		pinjamanRepository: repo,
		anggota:            anggota,
		simpanan:           simpanan,
		cfg:                cfg,
		txRunner:           runner,
	}
	return service, runner, recorder
}

func pinjamanTestConfig() config.Config {
	return config.Config{
		BatasPinjamanKali:  3,
		JasaPersenPerBulan: 1,
	}
}

func validPinjamanInput() models.PinjamanInput {
	return models.PinjamanInput{
		IDAnggota:        7,
		TanggalPengajuan: time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC),
		JumlahPinjaman:   1000000,
		Tenor:            12,
	}
}

func TestHitungJasaPerBulan(t *testing.T) {
	tests := []struct {
		name   string
		jumlah int64
		persen float64
		want   int64
	}{
		{name: "satu persen", jumlah: 1000000, persen: 1, want: 10000},
		{name: "pembulatan dua persen", jumlah: 333333, persen: 2, want: 6667},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &PinjamanService{cfg: config.Config{JasaPersenPerBulan: test.persen}}
			if got := service.HitungJasaPerBulan(test.jumlah); got != test.want {
				t.Errorf("jasa = %d, ingin %d", got, test.want)
			}
		})
	}
}

func TestHitungBatasMaksimal(t *testing.T) {
	simpanan := &fakeSimpananLookup{
		summary: models.RingkasanSimpanan{GrandTotal: 2500000},
	}
	service := &PinjamanService{
		simpanan: simpanan,
		cfg:      pinjamanTestConfig(),
	}

	got, err := service.HitungBatasMaksimal(7)
	if err != nil {
		t.Fatalf("HitungBatasMaksimal error = %v", err)
	}
	if got != 7500000 {
		t.Errorf("batas = %d, ingin 7500000", got)
	}
	if simpanan.calls != 1 {
		t.Errorf("panggilan TotalSimpanan = %d, ingin 1", simpanan.calls)
	}

	if _, err := service.HitungBatasMaksimal(0); !errors.Is(err, ErrInputPinjaman) {
		t.Errorf("id 0 error = %v, ingin ErrInputPinjaman", err)
	}
	if simpanan.calls != 1 {
		t.Errorf("panggilan TotalSimpanan = %d, ingin 1", simpanan.calls)
	}
}

func TestAjukanPinjaman(t *testing.T) {
	tanggalKosong := validPinjamanInput()
	tanggalKosong.TanggalPengajuan = time.Time{}

	tests := []struct {
		name        string
		input       models.PinjamanInput
		anggota     models.Anggota
		lockErr     error
		exists      bool
		grandTotal  int64
		wantErr     error
		wantJasa    int64
		wantAnggota int
		wantLock    int
		wantOrder   []string
	}{
		{
			name:        "pengajuan valid",
			input:       validPinjamanInput(),
			anggota:     models.Anggota{Status: "Aktif"},
			grandTotal:  1000000,
			wantJasa:    10000,
			wantAnggota: 1,
			wantLock:    1,
			wantOrder:   []string{"simpanan", "tx", "lock", "anggota"},
		},
		{
			name:       "anggota tidak ditemukan saat mengunci",
			input:      validPinjamanInput(),
			anggota:    models.Anggota{Status: "Aktif"},
			lockErr:    sql.ErrNoRows,
			grandTotal: 1000000,
			wantErr:    ErrAnggotaTidakDitemukan,
			wantLock:   1,
		},
		{
			name:        "anggota tidak aktif",
			input:       validPinjamanInput(),
			anggota:     models.Anggota{Status: "Keluar"},
			grandTotal:  1000000,
			wantErr:     ErrAnggotaTidakAktif,
			wantAnggota: 1,
			wantLock:    1,
		},
		{
			name:        "tenor terlalu kecil",
			input:       models.PinjamanInput{IDAnggota: 7, TanggalPengajuan: time.Now(), JumlahPinjaman: 1000000, Tenor: 0},
			anggota:     models.Anggota{Status: "Aktif"},
			grandTotal:  1000000,
			wantErr:     ErrTenorTidakValid,
			wantAnggota: 1,
			wantLock:    1,
		},
		{
			name:        "tenor terlalu besar",
			input:       models.PinjamanInput{IDAnggota: 7, TanggalPengajuan: time.Now(), JumlahPinjaman: 1000000, Tenor: 61},
			anggota:     models.Anggota{Status: "Aktif"},
			grandTotal:  1000000,
			wantErr:     ErrTenorTidakValid,
			wantAnggota: 1,
			wantLock:    1,
		},
		{
			name:        "sudah punya pinjaman aktif",
			input:       validPinjamanInput(),
			anggota:     models.Anggota{Status: "Aktif"},
			exists:      true,
			grandTotal:  1000000,
			wantErr:     ErrPinjamanAktifAda,
			wantAnggota: 1,
			wantLock:    1,
		},
		{
			name:        "melebihi batas",
			input:       models.PinjamanInput{IDAnggota: 7, TanggalPengajuan: time.Now(), JumlahPinjaman: 4000000, Tenor: 12},
			anggota:     models.Anggota{Status: "Aktif"},
			grandTotal:  1000000,
			wantErr:     ErrMelebihiBatas,
			wantAnggota: 1,
			wantLock:    1,
		},
		{
			name:       "jumlah tidak valid",
			input:      models.PinjamanInput{IDAnggota: 7, TanggalPengajuan: time.Now(), JumlahPinjaman: 0, Tenor: 12},
			grandTotal: 1000000,
			wantErr:    ErrJumlahTidakValid,
		},
		{
			name:       "tanggal kosong",
			input:      tanggalKosong,
			grandTotal: 1000000,
			wantErr:    ErrTanggalTidakValid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakePinjamanRepo{exists: test.exists, lockErr: test.lockErr}
			anggota := &fakeAnggotaLookup{anggota: test.anggota}
			simpanan := &fakeSimpananLookup{
				summary: models.RingkasanSimpanan{GrandTotal: test.grandTotal},
			}
			service, runner, recorder := newPinjamanTestService(repo, anggota, simpanan, pinjamanTestConfig())

			got, err := service.AjukanPinjaman(test.input)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, ingin %v", err, test.wantErr)
				}
				if len(repo.inserted) != 0 {
					t.Errorf("insert dipanggil sebanyak %d, ingin 0", len(repo.inserted))
				}
			} else {
				if err != nil {
					t.Fatalf("AjukanPinjaman error = %v", err)
				}
				if len(repo.inserted) != 1 {
					t.Fatalf("insert dipanggil sebanyak %d, ingin 1", len(repo.inserted))
				}
				inserted := repo.inserted[0]
				if inserted.IDPinjaman != 42 || inserted.Status != "Aktif" || inserted.TanggalCair != nil {
					t.Errorf("data pinjaman tersimpan = %+v", inserted)
				}
				if inserted.JasaPerBulan != test.wantJasa {
					t.Errorf("jasa tersimpan = %d, ingin %d", inserted.JasaPerBulan, test.wantJasa)
				}
				if inserted.IDPinjaman != got.IDPinjaman {
					t.Errorf("ID hasil = %d, ingin %d", got.IDPinjaman, inserted.IDPinjaman)
				}
			}

			if anggota.calls != test.wantAnggota {
				t.Errorf("panggilan anggota = %d, ingin %d", anggota.calls, test.wantAnggota)
			}
			if repo.lockCalls != test.wantLock {
				t.Errorf("panggilan lock = %d, ingin %d", repo.lockCalls, test.wantLock)
			}
			if runner.ran != (test.wantLock > 0) {
				t.Errorf("transaksi dijalankan = %v, ingin %v", runner.ran, test.wantLock > 0)
			}
			if test.wantOrder != nil && !reflect.DeepEqual(recorder.calls, test.wantOrder) {
				t.Errorf("urutan panggilan = %v, ingin %v", recorder.calls, test.wantOrder)
			}
		})
	}
}

func TestCekPinjamanAktif(t *testing.T) {
	repo := &fakePinjamanRepo{exists: true}
	service := &PinjamanService{pinjamanRepository: repo}

	got, err := service.CekPinjamanAktif(7)
	if err != nil || !got {
		t.Errorf("CekPinjamanAktif = %v, %v; ingin true, nil", got, err)
	}
	if repo.existsCalls != 1 {
		t.Errorf("panggilan ExistsAktifByAnggota = %d, ingin 1", repo.existsCalls)
	}

	if _, err := service.CekPinjamanAktif(0); !errors.Is(err, ErrInputPinjaman) {
		t.Errorf("id 0 error = %v, ingin ErrInputPinjaman", err)
	}
}

func TestBatasPinjaman(t *testing.T) {
	repo := &fakePinjamanRepo{exists: true}
	simpanan := &fakeSimpananLookup{
		summary: models.RingkasanSimpanan{GrandTotal: 2000000},
	}
	service := &PinjamanService{
		pinjamanRepository: repo,
		simpanan:           simpanan,
		cfg:                pinjamanTestConfig(),
	}

	batas, aktif, err := service.BatasPinjaman(7)
	if err != nil {
		t.Fatalf("BatasPinjaman error = %v", err)
	}
	if batas != 6000000 || !aktif {
		t.Errorf("BatasPinjaman = %d, %v; ingin 6000000, true", batas, aktif)
	}
}

func TestGetPinjaman(t *testing.T) {
	detail := models.PinjamanDetail{
		Pinjaman:    models.Pinjaman{IDPinjaman: 9},
		NamaAnggota: "Sari",
		NoAnggota:   "AGT-2026-0001",
	}
	repo := &fakePinjamanRepo{detail: detail}
	service := &PinjamanService{pinjamanRepository: repo}

	got, err := service.GetPinjaman(9)
	if err != nil {
		t.Fatalf("GetPinjaman error = %v", err)
	}
	if got.NamaAnggota != "Sari" || got.IDPinjaman != 9 {
		t.Errorf("detail = %+v", got)
	}

	if _, err := service.GetPinjaman(0); !errors.Is(err, ErrInputPinjaman) {
		t.Errorf("id 0 error = %v, ingin ErrInputPinjaman", err)
	}

	repo.detailErr = sql.ErrNoRows
	if _, err := service.GetPinjaman(9); !errors.Is(err, ErrPinjamanTidakDitemukan) {
		t.Errorf("error = %v, ingin ErrPinjamanTidakDitemukan", err)
	}
}

func TestListPinjaman(t *testing.T) {
	items := []models.PinjamanDetail{{Pinjaman: models.Pinjaman{IDPinjaman: 3}}}
	repo := &fakePinjamanRepo{list: items, count: 11}
	service := &PinjamanService{pinjamanRepository: repo}

	got, total, err := service.ListPinjaman("Aktif", 7, 2, 10)
	if err != nil {
		t.Fatalf("ListPinjaman error = %v", err)
	}
	if len(got) != 1 || total != 11 {
		t.Errorf("hasil = %d item, total %d; ingin 1 item, total 11", len(got), total)
	}
	if repo.lastStatus != "Aktif" || repo.lastID != 7 || repo.lastOffset != 10 || repo.lastLimit != 10 {
		t.Errorf("argumen repo = %q, %d, %d, %d", repo.lastStatus, repo.lastID, repo.lastOffset, repo.lastLimit)
	}

	tests := []struct {
		name    string
		status  string
		id      int64
		page    int
		limit   int
		wantErr error
	}{
		{name: "halaman 0", page: 0, limit: 10, wantErr: ErrInputPinjaman},
		{name: "limit 0", page: 1, limit: 0, wantErr: ErrInputPinjaman},
		{name: "limit 101", page: 1, limit: 101, wantErr: ErrInputPinjaman},
		{name: "status tidak dikenal", status: "Lama", page: 1, limit: 10, wantErr: ErrInputPinjaman},
		{name: "id anggota negatif", id: -1, page: 1, limit: 10, wantErr: ErrInputPinjaman},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := service.ListPinjaman(test.status, test.id, test.page, test.limit)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("error = %v, ingin %v", err, test.wantErr)
			}
		})
	}
}
