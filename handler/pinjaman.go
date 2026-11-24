package handler

import (
	"net/http"
	"strings"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type PinjamanHandler struct {
	pinjamanService *service.PinjamanService
}

func NewPinjamanHandler(pinjamanService *service.PinjamanService) *PinjamanHandler {
	return &PinjamanHandler{pinjamanService: pinjamanService}
}

func (h *PinjamanHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/anggota/{id}/batas-pinjaman", h.BatasPinjaman)
	mux.HandleFunc("POST /api/pinjaman", h.Ajukan)
	mux.HandleFunc("GET /api/pinjaman", h.List)
	mux.HandleFunc("GET /api/pinjaman/{id}", h.GetByID)
	mux.HandleFunc("POST /api/pinjaman/{id}/cairkan", h.Cairkan)
}

type pinjamanRequest struct {
	IDAnggota        int64  `json:"id_anggota"`
	JumlahPinjaman   int64  `json:"jumlah_pinjaman"`
	Tenor            int    `json:"tenor"`
	TanggalPengajuan string `json:"tanggal_pengajuan"`
}

type pencairanRequest struct {
	TanggalCair string `json:"tanggal_cair"`
}

func (h *PinjamanHandler) BatasPinjaman(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	batas, err := h.pinjamanService.HitungBatasMaksimal(id)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	aktif, err := h.pinjamanService.CekPinjamanAktif(id)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data: map[string]interface{}{
			"batas_maksimal":     batas,
			"ada_pinjaman_aktif": aktif,
		},
		Message: "OK",
	})
}

func (h *PinjamanHandler) Ajukan(w http.ResponseWriter, r *http.Request) {
	var request pinjamanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}

	tanggalPengajuan := time.Time{}
	if strings.TrimSpace(request.TanggalPengajuan) != "" {
		var err error
		tanggalPengajuan, err = util.ParseTanggal(request.TanggalPengajuan)
		if err != nil {
			writeTransaksiError(w, service.ErrTanggalTidakValid)
			return
		}
	}

	pinjaman, err := h.pinjamanService.AjukanPinjaman(models.PinjamanInput{
		IDAnggota:        request.IDAnggota,
		JumlahPinjaman:   request.JumlahPinjaman,
		Tenor:            request.Tenor,
		TanggalPengajuan: tanggalPengajuan,
	})
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, successResponse{
		Success: true,
		Data:    pinjaman,
		Message: "pinjaman berhasil diajukan",
	})
}

func (h *PinjamanHandler) List(w http.ResponseWriter, r *http.Request) {
	var idAnggota int64
	var err error
	if value := strings.TrimSpace(r.URL.Query().Get("id_anggota")); value != "" {
		idAnggota, err = service.ParsePositiveID(value)
		if err != nil {
			writeTransaksiError(w, err)
			return
		}
	}

	daftar, total, err := h.pinjamanService.ListPinjaman(
		strings.TrimSpace(r.URL.Query().Get("status")),
		idAnggota,
		queryInt(r, "page", 1),
		queryInt(r, "limit", 10),
	)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data: map[string]interface{}{
			"items": daftar,
			"total": total,
		},
		Message: "OK",
	})
}

func (h *PinjamanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	detail, err := h.pinjamanService.GetPinjaman(id)
	if err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{Success: true, Data: detail, Message: "OK"})
}

func (h *PinjamanHandler) Cairkan(w http.ResponseWriter, r *http.Request) {
	id, err := service.ParsePositiveID(r.PathValue("id"))
	if err != nil {
		writeTransaksiError(w, err)
		return
	}

	var request pencairanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	tanggalCair, err := util.ParseTanggal(request.TanggalCair)
	if err != nil {
		writeTransaksiError(w, service.ErrTanggalTidakValid)
		return
	}

	if err := h.pinjamanService.CairkanPinjaman(id, tanggalCair); err != nil {
		writeTransaksiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Message: "pinjaman berhasil dicairkan",
	})
}
