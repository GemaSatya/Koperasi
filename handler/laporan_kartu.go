package handler

import (
	"errors"
	"net/http"

	"github.com/GemaSatya/ProjekPemweb/service"
)

type LaporanKartuHandler struct {
	kartuService *service.LaporanKartuService
}

func NewLaporanKartuHandler(kartuService *service.LaporanKartuService) *LaporanKartuHandler {
	return &LaporanKartuHandler{kartuService: kartuService}
}

func (h *LaporanKartuHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/laporan/kartu-pinjaman/{id_pinjaman}", h.KartuPinjaman)
}

func (h *LaporanKartuHandler) KartuPinjaman(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id_pinjaman"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id pinjaman tidak valid")
		return
	}

	kartu, err := h.kartuService.KartuPinjaman(id)
	if err != nil {
		writeKartuError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data:    kartu,
		Message: "OK",
	})
}

func writeKartuError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInputPinjaman):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrPinjamanTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}
