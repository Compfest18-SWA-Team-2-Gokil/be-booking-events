package domain

import "errors"

var (
	ErrInvalidToken  = errors.New("queue token tidak valid atau signature salah")
	ErrTokenExpired  = errors.New("queue token sudah kadaluarsa")
	ErrAlreadyInQueue = errors.New("user sudah terdaftar dalam antrean event ini")
	ErrNotInQueue    = errors.New("user tidak ditemukan dalam antrean")
)
