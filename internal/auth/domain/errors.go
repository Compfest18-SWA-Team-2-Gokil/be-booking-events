package domain

import "errors"

var (
	ErrEmailAlreadyTaken  = errors.New("email sudah terdaftar")
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrUserNotFound       = errors.New("user tidak ditemukan")
	ErrInvalidEmail       = errors.New("format email tidak valid")
	ErrNameRequired       = errors.New("nama tidak boleh kosong")
	ErrPasswordTooShort   = errors.New("password minimal 8 karakter")
	ErrInvalidRole        = errors.New("role tidak valid")
	ErrForbiddenRole      = errors.New("role ADMIN tidak bisa didaftarkan melalui endpoint publik")
	ErrNotGateOperator    = errors.New("user yang di-assign bukan GATE_OPERATOR")
	ErrInvalidToken       = errors.New("token tidak valid atau sudah kadaluarsa")
	ErrUsernameRequired   = errors.New("username wajib diisi")
	ErrInvalidUsername    = errors.New("format username tidak valid (3-30 karakter, huruf kecil, angka, underscore)")
	ErrUsernameAlreadyTaken = errors.New("username sudah digunakan")
	ErrNotEventOrganizer  = errors.New("bukan organizer dari event ini")
	ErrAlreadyAssigned    = errors.New("gate operator sudah di-assign ke event ini")
	ErrWrongPassword      = errors.New("password lama tidak sesuai")
	ErrNewPasswordTooShort = errors.New("password baru minimal 8 karakter")
	ErrSameUsername       = errors.New("username baru sama dengan username lama")
)
