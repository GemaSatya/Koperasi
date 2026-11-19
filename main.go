package main

import (
	"log"
	"net/http"

	"github.com/GemaSatya/ProjekPemweb/config"
	"github.com/GemaSatya/ProjekPemweb/handler"
	"github.com/GemaSatya/ProjekPemweb/middleware"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
	"github.com/GemaSatya/ProjekPemweb/service"
)

func main() {

	mux := http.NewServeMux()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("konfigurasi tidak valid: %v", err)
	}

	if err := models.ConnectDatabase(cfg.DBDSN); err != nil {
		log.Fatalf("database tidak terhubung: %v", err)
	}
	defer func() {
		if err := models.CloseDatabase(); err != nil {
			log.Printf("gagal menutup koneksi database: %v", err)
		}
	}()

	adminRepository := repository.NewAdminRepository(models.DB)
	authService := service.NewAuthService(adminRepository)
	authHandler := handler.NewAuthHandler(authService)
	authHandler.RegisterRoutes(mux)

	anggotaRepository := repository.NewAnggotaRepository(models.DB)
	anggotaService := service.NewAnggotaService(anggotaRepository)
	anggotaHandler := handler.NewAnggotaHandler(anggotaService)
	anggotaHandler.RegisterRoutes(mux)

	simpananRepository := repository.NewSimpananRepository(models.DB)
	penarikanRepository := repository.NewPenarikanRepository(models.DB)
	simpananService := service.NewSimpananService(
		models.DB,
		simpananRepository,
		penarikanRepository,
	)
	simpananHandler := handler.NewSimpananHandler(simpananService)
	simpananHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: middleware.CORS(mux),
	}

	log.Printf("server berjalan di http://localhost:%s", cfg.AppPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("gagal memulai server: %v", err)
	}
}
