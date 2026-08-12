package appconfig

import (
	"os"
	"strings"
)

// IsDebug returns true jika APP_DEBUG=true di environment.
// Saat debug aktif, internal server error akan menampilkan pesan asli dari error
// sehingga lebih mudah untuk di-diagnosa selama development.
func IsDebug() bool {
	return strings.EqualFold(os.Getenv("APP_DEBUG"), "true")
}
