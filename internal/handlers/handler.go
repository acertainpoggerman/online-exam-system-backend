package handlers

import "github.com/acertainpoggerman/online-exam-system/internal/services"

type Handler struct {
	service *services.Service
}

func NewHandler(svc *services.Service) *Handler {
	return &Handler{service: svc}
}
