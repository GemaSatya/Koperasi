package handler

import (
	"errors"
	"net/http"
	"strings"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type SimpananHandler struct {
	simpananService *service.SimpananService
}

func NewSimpananHandler(simpananService *service.SimpananService) *SimpananHandler {
	return &SimpananHandler{simpananService: simpananService}
}

func (h *SimpananHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/simpanan", h.CreateSimpanan)
	mux.HandleFunc("GET /api/anggota/{id}/simpanan", h.ListSimpanan)
	mux.HandleFunc("GET /api/anggota/{id}/simpanan/ringkasan", h.Summary)
	mux.HandleFunc("POST /api/penarikan-sukarela", h.CreatePenarikan)
	mux.HandleFunc("GET /api/anggota/{id}/penarikan", h.ListPenarikan)
}

type simpananRequest struct {
	IDAnggota int64  `json:"id_anggota"`
	Jenis     string `json:"jenis"`
	Tanggal   string `json:"tanggal"`
	Nominal   int64  `json:"nominal"`
}

type penarikanRequest struct {
	IDAnggota int64  `json:"id_anggota"`
	Tanggal   string `json:"tanggal"`
	Nominal   int64  `json:"nominal"`
}

func (h *SimpananHandler) CreateSimpanan(w http.ResponseWriter, r *http.Request) {
	var request simpananRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	tanggal, err := util.ParseTanggal(request.Tanggal)
	if err != nil {
		writeError(w, http.StatusBadRequest, service.ErrTanggalTidakValid.Error())
		return
	}

	simpanan, err := h.simpananService.TambahSimpanan(models.SimpananInput{
		IDAnggota: request.IDAnggota,
		Jenis:     strings.TrimSpace(request.Jenis),
		Tanggal:   tanggal,
		Nominal:   request.Nominal,
	})
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data:    simpanan,
		Message: "simpanan berhasil ditambahkan",
	})
}

func (h *SimpananHandler) ListSimpanan(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	result, err := h.simpananService.RiwayatSimpanan(id, r.URL.Query().Get("jenis"))
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *SimpananHandler) Summary(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	result, err := h.simpananService.TotalSimpanan(id)
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *SimpananHandler) CreatePenarikan(w http.ResponseWriter, r *http.Request) {
	var request penarikanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	tanggal, err := util.ParseTanggal(request.Tanggal)
	if err != nil {
		writeError(w, http.StatusBadRequest, service.ErrTanggalTidakValid.Error())
		return
	}

	penarikan, err := h.simpananService.TarikSukarela(models.PenarikanInput{
		IDAnggota: request.IDAnggota,
		Tanggal:   tanggal,
		Nominal:   request.Nominal,
	})
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data:    penarikan,
		Message: "penarikan sukarela berhasil",
	})
}

func (h *SimpananHandler) ListPenarikan(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	result, err := h.simpananService.RiwayatPenarikan(id)
	if err != nil {
		h.writeSimpananError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *SimpananHandler) writeSimpananError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInputAnggota),
		errors.Is(err, service.ErrJenisSimpanan),
		errors.Is(err, service.ErrNominalTidakValid),
		errors.Is(err, service.ErrTanggalTidakValid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrAnggotaTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAnggotaTidakAktif),
		errors.Is(err, service.ErrSimpananPokokAda),
		errors.Is(err, service.ErrSaldoTidakCukup):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}
