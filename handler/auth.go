package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/GemaSatya/ProjekPemweb/service"
	"github.com/GemaSatya/ProjekPemweb/util"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/login", h.Login)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type successResponse = util.SuccessResponse

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	if strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
		writeError(w, http.StatusBadRequest, "username dan password wajib diisi")
		return
	}

	admin, err := h.authService.Login(input.Username, input.Password)
	if err != nil {
		if errors.Is(err, service.ErrLoginGagal) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Data:    admin,
		Message: "OK",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	util.WriteJSON(w, status, payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	util.WriteError(w, status, errors.New(message))
}
