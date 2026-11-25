package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type PembayaranHandler struct {
	pembayaranService *service.PembayaranService
}

func NewPembayaranHandler(pembayaranService *service.PembayaranService) *PembayaranHandler {
	return &PembayaranHandler{pembayaranService: pembayaranService}
}

func (h *PembayaranHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/pinjaman/{id}/jadwal", h.Jadwal)
	mux.HandleFunc("GET /api/jadwal/{id}/tagihan", h.Tagihan)
	mux.HandleFunc("POST /api/jadwal/{id}/bayar", h.Bayar)
	mux.HandleFunc("GET /api/pinjaman/{id}/pembayaran", h.Riwayat)
}

func (h *PembayaranHandler) Jadwal(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id pinjaman tidak valid")
		return
	}

	jadwal, err := h.pembayaranService.GetJadwal(id)
	if err != nil {
		writePembayaranError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: jadwal, Message: "OK"})
}

func (h *PembayaranHandler) Tagihan(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id jadwal tidak valid")
		return
	}
	tanggal, err := tanggalQuery(r, "tanggal")
	if err != nil {
		writeError(w, http.StatusBadRequest, "tanggal tidak valid")
		return
	}

	tagihan, err := h.pembayaranService.HitungTagihan(id, tanggal)
	if err != nil {
		writePembayaranError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: tagihan, Message: "OK"})
}

type bayarRequest struct {
	TanggalBayar string `json:"tanggal_bayar"`
	Nominal      int64  `json:"nominal"`
}

func (h *PembayaranHandler) Bayar(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id jadwal tidak valid")
		return
	}

	var input bayarRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	tanggal, err := util.ParseTanggal(input.TanggalBayar)
	if err != nil {
		writeError(w, http.StatusBadRequest, "tanggal_bayar harus berformat YYYY-MM-DD")
		return
	}

	pembayaran, err := h.pembayaranService.BayarAngsuran(id, tanggal, input.Nominal)
	if err != nil {
		writePembayaranError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: pembayaran, Message: "pembayaran berhasil"})
}

func (h *PembayaranHandler) Riwayat(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id pinjaman tidak valid")
		return
	}

	items, err := h.pembayaranService.RiwayatPembayaranPinjaman(id)
	if err != nil {
		writePembayaranError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: items, Message: "OK"})
}

// tanggalQuery memakai hari ini bila query kosong.
func tanggalQuery(r *http.Request, key string) (time.Time, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return time.Now(), nil
	}
	return util.ParseTanggal(value)
}

func writePembayaranError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrJadwalTidakDitemukan), errors.Is(err, service.ErrPinjamanTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrStatusPinjaman), errors.Is(err, service.ErrJadwalLunas):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrJadwalTidakValid),
		errors.Is(err, service.ErrTanggalTidakValid),
		errors.Is(err, service.ErrNominalTidakValid),
		errors.Is(err, service.ErrNominalMelebihiTagihan),
		errors.Is(err, service.ErrInputPinjaman):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}
