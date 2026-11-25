package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type PinjamanHandler struct {
	pinjamanService  *service.PinjamanService
	pencairanService *service.PencairanService
}

func NewPinjamanHandler(
	pinjamanService *service.PinjamanService,
	pencairanService *service.PencairanService,
) *PinjamanHandler {
	return &PinjamanHandler{
		pinjamanService:  pinjamanService,
		pencairanService: pencairanService,
	}
}

func (h *PinjamanHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/anggota/{id}/batas-pinjaman", h.BatasPinjaman)
	mux.HandleFunc("POST /api/pinjaman", h.Create)
	mux.HandleFunc("GET /api/pinjaman", h.List)
	mux.HandleFunc("GET /api/pinjaman/{id}", h.GetByID)
	mux.HandleFunc("POST /api/pinjaman/{id}/cairkan", h.Cairkan)
}

type pinjamanRequest struct {
	IDAnggota        int64  `json:"id_anggota"`
	TanggalPengajuan string `json:"tanggal_pengajuan"`
	JumlahPinjaman   int64  `json:"jumlah_pinjaman"`
	Tenor            int    `json:"tenor"`
}

type pencairanRequest struct {
	TanggalCair string `json:"tanggal_cair"`
}

func (h *PinjamanHandler) BatasPinjaman(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id anggota tidak valid")
		return
	}

	batas, aktif, err := h.pinjamanService.BatasPinjaman(id)
	if err != nil {
		h.writePinjamanError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data: map[string]interface{}{
			"batas_maksimal": batas,
			"pinjaman_aktif": aktif,
		},
		Message: "OK",
	})
}

func (h *PinjamanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request pinjamanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}

	tanggal, err := util.ParseTanggal(request.TanggalPengajuan)
	if err != nil {
		writeError(w, http.StatusBadRequest, service.ErrTanggalTidakValid.Error())
		return
	}

	pinjaman, err := h.pinjamanService.AjukanPinjaman(models.PinjamanInput{
		IDAnggota:        request.IDAnggota,
		TanggalPengajuan: tanggal,
		JumlahPinjaman:   request.JumlahPinjaman,
		Tenor:            request.Tenor,
	})
	if err != nil {
		h.writePinjamanError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data:    pinjaman,
		Message: "pengajuan pinjaman berhasil",
	})
}

func (h *PinjamanHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 10)

	idAnggota, err := queryIDAnggotaPinjaman(r)
	if err != nil {
		h.writePinjamanError(w, err)
		return
	}

	items, total, err := h.pinjamanService.ListPinjaman(
		strings.TrimSpace(r.URL.Query().Get("status")),
		idAnggota,
		page,
		limit,
	)
	if err != nil {
		h.writePinjamanError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data: map[string]interface{}{
			"items": items,
			"total": total,
			"page":  page,
			"limit": limit,
		},
		Message: "OK",
	})
}

func (h *PinjamanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id pinjaman tidak valid")
		return
	}

	pinjaman, err := h.pinjamanService.GetPinjaman(id)
	if err != nil {
		h.writePinjamanError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data:    pinjaman,
		Message: "OK",
	})
}

func (h *PinjamanHandler) Cairkan(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id pinjaman tidak valid")
		return
	}

	var request pencairanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}

	tanggalCair, err := util.ParseTanggal(request.TanggalCair)
	if err != nil {
		writeError(w, http.StatusBadRequest, service.ErrTanggalTidakValid.Error())
		return
	}

	jumlahJadwal, err := h.pencairanService.CairkanPinjaman(id, tanggalCair)
	if err != nil {
		h.writePinjamanError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data: map[string]interface{}{
			"id_pinjaman":   id,
			"tanggal_cair":  request.TanggalCair,
			"jumlah_jadwal": jumlahJadwal,
		},
		Message: "pencairan pinjaman berhasil",
	})
}

func (h *PinjamanHandler) writePinjamanError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInputPinjaman),
		errors.Is(err, service.ErrInputAnggota),
		errors.Is(err, service.ErrTenorTidakValid),
		errors.Is(err, service.ErrJumlahTidakValid),
		errors.Is(err, service.ErrTanggalTidakValid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrPinjamanTidakDitemukan),
		errors.Is(err, service.ErrAnggotaTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAnggotaTidakAktif),
		errors.Is(err, service.ErrPinjamanAktifAda),
		errors.Is(err, service.ErrMelebihiBatas),
		errors.Is(err, service.ErrStatusPinjaman),
		errors.Is(err, service.ErrJadwalTidakValid):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func queryIDAnggotaPinjaman(r *http.Request) (int64, error) {
	value := strings.TrimSpace(r.URL.Query().Get("id_anggota"))
	if value == "" {
		return 0, nil
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 0 {
		return 0, service.ErrInputPinjaman
	}
	return id, nil
}
