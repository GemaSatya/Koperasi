package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

// tanggalDariQuery membaca tanggal dari query string. Jika kosong dan
// defaultHariIni true, tanggal hari ini yang dipakai.
func tanggalDariQuery(r *http.Request, key string, defaultHariIni bool) (time.Time, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		if defaultHariIni {
			return util.ParseTanggal(time.Now().Format("2006-01-02"))
		}
		return time.Time{}, nil
	}
	tanggal, err := util.ParseTanggal(value)
	if err != nil {
		return time.Time{}, service.ErrTanggalTidakValid
	}
	return tanggal, nil
}

// writeTransaksiError memetakan error bisnis ke kode HTTP yang sesuai.
func writeTransaksiError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInputAnggota),
		errors.Is(err, service.ErrTanggalTidakValid),
		errors.Is(err, service.ErrTenorTidakValid),
		errors.Is(err, service.ErrPeriodeTidakValid),
		errors.Is(err, service.ErrNominalTidakValid),
		errors.Is(err, service.ErrNominalMelebihiTagihan):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrAnggotaTidakDitemukan),
		errors.Is(err, service.ErrPinjamanTidakDitemukan),
		errors.Is(err, service.ErrJadwalTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAnggotaTidakAktif),
		errors.Is(err, service.ErrPinjamanAktifAda),
		errors.Is(err, service.ErrMelebihiBatas),
		errors.Is(err, service.ErrStatusPinjaman),
		errors.Is(err, service.ErrJadwalLunas),
		errors.Is(err, service.ErrAnggotaMasihMeminjam):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

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

type pembayaranRequest struct {
	TanggalBayar string `json:"tanggal_bayar"`
	Nominal      int64  `json:"nominal"`
}

func (h *PembayaranHandler) Jadwal(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.pembayaranService.GetJadwal(id)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *PembayaranHandler) Tagihan(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	tanggal, err := tanggalDariQuery(r, "tanggal", true)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.pembayaranService.HitungTagihan(id, tanggal)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

func (h *PembayaranHandler) Bayar(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}

	var request pembayaranRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	tanggal, err := util.ParseTanggal(request.TanggalBayar)
	if err != nil {
		writeTransaksiError(w, service.ErrTanggalTidakValid)
		return
	}

	pembayaran, statusJadwal, err := h.pembayaranService.BayarAngsuran(id, tanggal, request.Nominal)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data: map[string]interface{}{
			"id_pembayaran": pembayaran.IDPembayaran,
			"id_jadwal":     pembayaran.IDJadwal,
			"tanggal_bayar": pembayaran.TanggalBayar,
			"total_bayar":   pembayaran.TotalBayar,
			"denda_dibayar": pembayaran.DendaDibayar,
			"jasa_dibayar":  pembayaran.JasaDibayar,
			"pokok_dibayar": pembayaran.PokokDibayar,
			"status_jadwal": statusJadwal,
		},
		Message: "pembayaran berhasil dicatat",
	})
}

func (h *PembayaranHandler) Riwayat(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	result, err := h.pembayaranService.RiwayatPembayaranPinjaman(id)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: result, Message: "OK"})
}

// filterDariRequest membaca filter laporan dari query string.
func filterDariRequest(r *http.Request) (models.FilterLaporan, error) {
	dari, err := tanggalDariQuery(r, "dari", false)
	if err != nil {
		return models.FilterLaporan{}, err
	}
	sampai, err := tanggalDariQuery(r, "sampai", false)
	if err != nil {
		return models.FilterLaporan{}, err
	}

	var idAnggota int64
	if value := strings.TrimSpace(r.URL.Query().Get("id_anggota")); value != "" {
		idAnggota, err = service.ParsePositiveID(value)
		if err != nil {
			return models.FilterLaporan{}, err
		}
	}

	return models.FilterLaporan{Dari: dari, Sampai: sampai, IDAnggota: idAnggota}, nil
}
