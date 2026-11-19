package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/service"
)

type AnggotaHandler struct {
	anggotaService *service.AnggotaService
}

func NewAnggotaHandler(anggotaService *service.AnggotaService) *AnggotaHandler {
	return &AnggotaHandler{anggotaService: anggotaService}
}

func (h *AnggotaHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/anggota", h.List)
	mux.HandleFunc("POST /api/anggota", h.Create)
	mux.HandleFunc("GET /api/anggota/{id}", h.GetByID)
	mux.HandleFunc("PUT /api/anggota/{id}", h.Update)
	mux.HandleFunc("PATCH /api/anggota/{id}/nonaktif", h.Deactivate)
}

func (h *AnggotaHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 10)

	anggotaList, total, err := h.anggotaService.List(
		r.URL.Query().Get("keyword"),
		r.URL.Query().Get("status"),
		page,
		limit,
	)
	if err != nil {
		h.writeAnggotaError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data: map[string]interface{}{
			"items": anggotaList,
			"total": total,
			"page":  page,
			"limit": limit,
		},
		Message: "OK",
	})
}

func (h *AnggotaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.AnggotaInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}

	anggota, err := h.anggotaService.Tambah(input)
	if err != nil {
		h.writeAnggotaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data:    anggota,
		Message: "anggota berhasil ditambahkan",
	})
}

func (h *AnggotaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		h.writeAnggotaError(w, err)
		return
	}

	anggota, err := h.anggotaService.GetByID(id)
	if err != nil {
		h.writeAnggotaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data:    anggota,
		Message: "OK",
	})
}

func (h *AnggotaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		h.writeAnggotaError(w, err)
		return
	}

	var input models.AnggotaInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	if err := h.anggotaService.Edit(id, input); err != nil {
		h.writeAnggotaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Message: "anggota berhasil diubah",
	})
}

func (h *AnggotaHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		h.writeAnggotaError(w, err)
		return
	}

	if err := h.anggotaService.Nonaktifkan(id); err != nil {
		h.writeAnggotaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Message: "anggota berhasil dinonaktifkan",
	})
}

func (h *AnggotaHandler) writeAnggotaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInputAnggota), errors.Is(err, service.ErrStatusAnggota):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrAnggotaTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAnggotaMasihMeminjam):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func decodeJSON(r *http.Request, destination interface{}) error {
	return json.NewDecoder(r.Body).Decode(destination)
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
