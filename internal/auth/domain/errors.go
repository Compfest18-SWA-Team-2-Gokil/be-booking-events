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
)
