package service

import (
	"testing"
	"time"

	"github.com/GemaSatya/ProjekPemweb/util"
)

func TestHitungJadwal(t *testing.T) {
	cair := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC)
	jadwal := HitungJadwal(100000, 3, 1000, cair)

	if len(jadwal) != 3 {
		t.Fatalf("jumlah jadwal = %d, want 3", len(jadwal))
	}

	wantPokok := []int64{33333, 33333, 33334}
	wantTempo := []time.Time{
		time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2024, time.March, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2024, time.April, 30, 0, 0, 0, 0, time.UTC),
	}

	var total int64
	for i, baris := range jadwal {
		if baris.Pokok != wantPokok[i] {
			t.Errorf("jadwal[%d].Pokok = %d, want %d", i, baris.Pokok, wantPokok[i])
		}
		if !baris.JatuhTempo.Equal(wantTempo[i]) {
			t.Errorf("jadwal[%d].JatuhTempo = %v, want %v", i, baris.JatuhTempo, wantTempo[i])
		}
		if baris.Jasa != 1000 {
			t.Errorf("jadwal[%d].Jasa = %d, want 1000", i, baris.Jasa)
		}
		if baris.Status != "Belum" {
			t.Errorf("jadwal[%d].Status = %s, want Belum", i, baris.Status)
		}
		total += baris.Pokok
	}
	if total != 100000 {
		t.Fatalf("total pokok = %d, want 100000", total)
	}
}

func TestHitungJadwalTenorSatu(t *testing.T) {
	cair := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.Local)
	jadwal := HitungJadwal(75000, 1, 750, cair)

	if len(jadwal) != 1 {
		t.Fatalf("jumlah jadwal = %d, want 1", len(jadwal))
	}
	if jadwal[0].Pokok != 75000 {
		t.Fatalf("pokok = %d, want 75000", jadwal[0].Pokok)
	}
	if !jadwal[0].JatuhTempo.Equal(util.AddMonths(cair, 1)) {
		t.Fatalf("jatuh tempo = %v, want %v", jadwal[0].JatuhTempo, util.AddMonths(cair, 1))
	}
}
