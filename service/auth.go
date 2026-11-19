package service

import (
	"database/sql"
	"errors"
	"strings"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

type AuthService struct {
	adminRepository *repository.AdminRepository
}

func NewAuthService(adminRepository *repository.AdminRepository) *AuthService {
	return &AuthService{adminRepository: adminRepository}
}

func (s *AuthService) Login(username, password string) (models.Admin, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return models.Admin{}, ErrLoginGagal
	}

	admin, err := s.adminRepository.FindByCredentials(username, password)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Admin{}, ErrLoginGagal
	}
	if err != nil {
		return models.Admin{}, err
	}

	return admin, nil
}
