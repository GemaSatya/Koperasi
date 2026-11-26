package handler

import (
	"net/http"

	"github.com/GemaSatya/ProjekPemweb/service"
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
	filter, err := filterDariRequest(r)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.laporanService.LaporanSimpanan(filter)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *LaporanHandler) KartuPinjaman(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id_pinjaman"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.laporanService.KartuPinjaman(id)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *LaporanHandler) RiwayatPembayaran(w http.ResponseWriter, r *http.Request) {
	filter, err := filterDariRequest(r)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.laporanService.RiwayatPembayaran(filter)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *LaporanHandler) Tunggakan(w http.ResponseWriter, r *http.Request) {
	tanggal, err := tanggalDariQuery(r, "tanggal", true)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.laporanService.DaftarTunggakan(tanggal)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *LaporanHandler) PendapatanJasa(w http.ResponseWriter, r *http.Request) {
	dari, err := tanggalDariQuery(r, "dari", false)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	sampai, err := tanggalDariQuery(r, "sampai", false)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.laporanService.PendapatanJasa(dari, sampai)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *LaporanHandler) PendapatanDenda(w http.ResponseWriter, r *http.Request) {
	dari, err := tanggalDariQuery(r, "dari", false)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	sampai, err := tanggalDariQuery(r, "sampai", false)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.laporanService.PendapatanDenda(dari, sampai)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}
