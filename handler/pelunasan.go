package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

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

type pelunasanRequest struct {
	Tanggal string `json:"tanggal"`
}

func (h *PelunasanHandler) Hitung(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id pinjaman tidak valid")
		return
	}

	tanggal, err := tanggalPelunasanDariQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, service.ErrTanggalTidakValid.Error())
		return
	}

	rincian, err := h.pelunasanService.HitungPelunasan(id, tanggal)
	if err != nil {
		writePelunasanError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data:    rincian,
		Message: "OK",
	})
}

func (h *PelunasanHandler) Proses(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id pinjaman tidak valid")
		return
	}

	var request pelunasanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}

	tanggal, err := util.ParseTanggal(strings.TrimSpace(request.Tanggal))
	if err != nil {
		writeError(w, http.StatusBadRequest, service.ErrTanggalTidakValid.Error())
		return
	}

	pelunasan, err := h.pelunasanService.ProsesPelunasan(id, tanggal)
	if err != nil {
		writePelunasanError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data:    pelunasan,
		Message: "pelunasan berhasil",
	})
}

func tanggalPelunasanDariQuery(r *http.Request) (time.Time, error) {
	value := strings.TrimSpace(r.URL.Query().Get("tanggal"))
	if value == "" {
		return time.Now(), nil
	}
	return util.ParseTanggal(value)
}

func writePelunasanError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInputPinjaman),
		errors.Is(err, service.ErrTanggalTidakValid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrPinjamanTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrStatusPinjaman):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}
