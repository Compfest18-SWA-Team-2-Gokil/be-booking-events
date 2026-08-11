package domain

import "errors"

var (
	ErrNameRequired      = errors.New("nama tidak boleh kosong")
	ErrLocationRequired  = errors.New("lokasi tidak boleh kosong")
	ErrDateMustBeFuture  = errors.New("tanggal event harus di masa depan")
	ErrInvalidPrice      = errors.New("harga tidak boleh negatif")
	ErrInvalidQuota      = errors.New("kuota harus minimal 1")
	ErrInvalidKind       = errors.New("jenis tiket harus GA atau SEATED")
	ErrEventNotFound     = errors.New("event tidak ditemukan")
	ErrTicketTypeNotFound = errors.New("ticket type tidak ditemukan")
	ErrQuotaExceeded     = errors.New("jumlah provision melebihi total kuota yang tersisa")
)
