package domain

type TicketKind string

const (
	TicketKindGA     TicketKind = "GA"
	TicketKindSeated TicketKind = "SEATED"
)

// PriceStatus menjaga harga tiket agar tidak berubah setelah dikunci.
// OPEN: harga masih bisa diubah Organizer.
// LOCKED: harga final, tidak bisa diubah — berlaku seterusnya meski tiket di-refund dan dijual ulang.
type PriceStatus string

const (
	PriceStatusOpen   PriceStatus = "OPEN"
	PriceStatusLocked PriceStatus = "LOCKED"
)

type TicketType struct {
	ID          string
	EventID     string
	Name        string
	Price       int64 // dalam satuan terkecil mata uang (sen/IDR)
	Kind        TicketKind
	TotalQuota  int
	PriceStatus PriceStatus
}
