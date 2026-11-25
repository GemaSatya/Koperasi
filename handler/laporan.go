package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type LaporanHandler struct {
	laporanService *service.LaporanService
}

func NewLaporanHandler(laporanService *service.LaporanService) *LaporanHandler {
	return &LaporanHandler{laporanService: laporanService}
}

func (h *LaporanHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/laporan/simpanan", h.Simpanan)
	mux.HandleFunc("GET /api/laporan/riwayat-pembayaran", h.RiwayatPembayaran)
	mux.HandleFunc("GET /api/laporan/tunggakan", h.Tunggakan)
	mux.HandleFunc("GET /api/laporan/pendapatan-jasa", h.PendapatanJasa)
	mux.HandleFunc("GET /api/laporan/pendapatan-denda", h.PendapatanDenda)
}

func (h *LaporanHandler) Simpanan(w http.ResponseWriter, r *http.Request) {
	filter, err := filterLaporan(r)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	items, err := h.laporanService.LaporanSimpanan(filter)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: items, Message: "OK"})
}

func (h *LaporanHandler) RiwayatPembayaran(w http.ResponseWriter, r *http.Request) {
	filter, err := filterLaporan(r)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	items, err := h.laporanService.RiwayatPembayaran(filter)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: items, Message: "OK"})
}

func (h *LaporanHandler) Tunggakan(w http.ResponseWriter, r *http.Request) {
	tanggal, err := tanggalQuery(r, "tanggal")
	if err != nil {
		writeLaporanError(w, service.ErrTanggalTidakValid)
		return
	}
	items, err := h.laporanService.DaftarTunggakan(tanggal)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: items, Message: "OK"})
}

func (h *LaporanHandler) PendapatanJasa(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := periodeWajib(r)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	pendapatan, err := h.laporanService.PendapatanJasa(dari, sampai)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: pendapatan, Message: "OK"})
}

func (h *LaporanHandler) PendapatanDenda(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := periodeWajib(r)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	pendapatan, err := h.laporanService.PendapatanDenda(dari, sampai)
	if err != nil {
		writeLaporanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: pendapatan, Message: "OK"})
}

func filterLaporan(r *http.Request) (models.FilterLaporan, error) {
	var filter models.FilterLaporan
	if value := r.URL.Query().Get("dari"); value != "" {
		parsed, err := util.ParseTanggal(value)
		if err != nil {
			return models.FilterLaporan{}, service.ErrTanggalTidakValid
		}
		filter.Dari = parsed
	}
	if value := r.URL.Query().Get("sampai"); value != "" {
		parsed, err := util.ParseTanggal(value)
		if err != nil {
			return models.FilterLaporan{}, service.ErrTanggalTidakValid
		}
		filter.Sampai = parsed
	}
	if value := r.URL.Query().Get("id_anggota"); value != "" {
		idAnggota, err := strconv.ParseInt(value, 10, 64)
		if err != nil || idAnggota <= 0 {
			return models.FilterLaporan{}, service.ErrInputAnggota
		}
		filter.IDAnggota = idAnggota
	}
	return filter, nil
}

func periodeWajib(r *http.Request) (time.Time, time.Time, error) {
	dariString := r.URL.Query().Get("dari")
	sampaiString := r.URL.Query().Get("sampai")
	if dariString == "" || sampaiString == "" {
		return time.Time{}, time.Time{}, service.ErrPeriodeTidakValid
	}
	dari, err := util.ParseTanggal(dariString)
	if err != nil {
		return time.Time{}, time.Time{}, service.ErrTanggalTidakValid
	}
	sampai, err := util.ParseTanggal(sampaiString)
	if err != nil {
		return time.Time{}, time.Time{}, service.ErrTanggalTidakValid
	}
	if dari.After(sampai) {
		return time.Time{}, time.Time{}, service.ErrPeriodeTidakValid
	}
	return dari, sampai, nil
}

func writeLaporanError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrPeriodeTidakValid),
		errors.Is(err, service.ErrTanggalTidakValid),
		errors.Is(err, service.ErrInputAnggota):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}
