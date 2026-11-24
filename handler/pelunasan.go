package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type pelunasanRequest struct {
	Tanggal string `json:"tanggal"`
}

func tanggalPelunasan(r *http.Request) (time.Time, error) {
	var request pelunasanRequest
	if r.Body != nil {
		// Body boleh kosong; tanggal lalu diambil dari query atau hari ini.
		_ = json.NewDecoder(r.Body).Decode(&request)
	}
	if value := strings.TrimSpace(request.Tanggal); value != "" {
		tanggal, err := util.ParseTanggal(value)
		if err != nil {
			return time.Time{}, service.ErrTanggalTidakValid
		}
		return tanggal, nil
	}
	return tanggalDariQuery(r, "tanggal", true)
}

type PelunasanHandler struct {
	pelunasanService *service.PelunasanService
}

func NewPelunasanHandler(pelunasanService *service.PelunasanService) *PelunasanHandler {
	return &PelunasanHandler{pelunasanService: pelunasanService}
}

func (h *PelunasanHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/pinjaman/{id}/pelunasan/hitung", h.Hitung)
	mux.HandleFunc("POST /api/pinjaman/{id}/pelunasan", h.Proses)
}

func (h *PelunasanHandler) Hitung(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	tanggal, err := tanggalPelunasan(r)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	rincian, err := h.pelunasanService.HitungPelunasan(id, tanggal)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: rincian, Message: "OK"})
}

func (h *PelunasanHandler) Proses(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	tanggal, err := tanggalPelunasan(r)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}

	pelunasan, err := h.pelunasanService.ProsesPelunasan(id, tanggal)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data:    pelunasan,
		Message: "pelunasan berhasil diproses",
	})
}
