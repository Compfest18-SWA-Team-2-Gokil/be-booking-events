package domain

import "errors"

var (
	ErrInvalidQRPayload   = errors.New("qr payload tidak valid: field wajib kosong")
	ErrInvalidSignature   = errors.New("signature QR tidak valid atau telah dimanipulasi")
	ErrTicketNotConfirmed = errors.New("tiket tidak ditemukan atau statusnya bukan CONFIRMED")
	ErrAlreadyAdmitted    = errors.New("tiket sudah digunakan masuk (sudah ADMITTED atau tidak lagi CONFIRMED)")
)
