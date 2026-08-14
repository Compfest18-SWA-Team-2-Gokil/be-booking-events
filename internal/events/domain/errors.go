package domain

import "errors"

var (
	ErrNameRequired                  = errors.New("nama tidak boleh kosong")
	ErrLocationRequired              = errors.New("lokasi tidak boleh kosong")
	ErrDateMustBeFuture              = errors.New("tanggal event harus di masa depan")
	ErrInvalidPrice                  = errors.New("harga tidak boleh negatif")
	ErrInvalidQuota                  = errors.New("kuota harus minimal 1")
	ErrInvalidKind                   = errors.New("jenis tiket harus GA atau SEATED")
	ErrInvalidCategory               = errors.New("kategori harus salah satu dari: music, olahraga, seni, workshop")
	ErrEventNotFound                 = errors.New("event tidak ditemukan")
	ErrTicketTypeNotFound            = errors.New("ticket type tidak ditemukan")
	ErrQuotaExceeded                 = errors.New("jumlah provision melebihi total kuota yang tersisa")
	ErrNotEventOrganizer             = errors.New("anda bukan organizer event ini")
	ErrCannotDeleteWithSoldTickets   = errors.New("event tidak bisa dihapus karena sudah ada tiket terjual")
	ErrCannotDeleteTTWithSoldTickets = errors.New("ticket type tidak bisa dihapus karena sudah ada tiket terjual")
	ErrPriceLocked                   = errors.New("harga sudah dikunci karena ada tiket terjual, tidak bisa diubah")
	ErrQuotaBelowSold                = errors.New("kuota tidak bisa dikurangi di bawah jumlah tiket yang sudah terjual")
)
