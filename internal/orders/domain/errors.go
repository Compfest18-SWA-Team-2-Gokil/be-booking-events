package domain

import "errors"

var (
	ErrOrderNotFound       = errors.New("order tidak ditemukan")
	ErrPaymentNotFound     = errors.New("payment tidak ditemukan")
	ErrNoHeldUnits         = errors.New("tidak ada tiket yang sedang di-hold")
	ErrOrderNotPaid        = errors.New("order belum dibayar")
	ErrOrderNotPending     = errors.New("order tidak dalam status yang bisa dibayar")
	ErrAlreadyRefunded        = errors.New("order sudah direfund")
	ErrRefundNotRequested     = errors.New("tidak ada permintaan refund dari buyer")
	ErrPaymentFailed          = errors.New("pembayaran gagal diproses")
	ErrTicketAlreadyAdmitted  = errors.New("tiket sudah digunakan/admitted, refund tidak dapat diajukan")
	ErrPaymentDiscrepancy     = errors.New("tiket telah kedaluwarsa dan terjual ke pembeli lain (payment discrepancy)")
)
