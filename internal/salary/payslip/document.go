// Package payslip renders pay slips (slip gaji) of salary snapshots as PDFs
// on the pharmacy letterhead.
//
// It knows nothing of the database: the salary package turns a snapshot into
// a Document and asks for its PDF.
package payslip

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ErrNotConfigured is returned when the pharmacy details for the letterhead
// are not set.
var ErrNotConfigured = errors.New("pharmacy details are not configured")

// Letterhead holds the pharmacy details printed at the top of every slip.
type Letterhead struct {
	Name          string `mapstructure:"name"`
	LicenseNumber string `mapstructure:"license_number"`
	Address       string `mapstructure:"address"`
	Email         string `mapstructure:"email"`
	Phone         string `mapstructure:"phone"`
}

// Configured reports whether every letterhead detail is set. Slips are not
// rendered until they are, so none goes out with a half-empty letterhead.
func (l Letterhead) Configured() bool {
	for _, v := range []string{l.Name, l.LicenseNumber, l.Address, l.Email, l.Phone} {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}

	return true
}

// Document is a pay slip to render. Every number comes from the stored
// snapshot; nothing is recomputed here.
type Document struct {
	Letterhead Letterhead

	SnapshotID   int64
	EmployeeName string
	// Year and Month (1-12) of the salary period.
	Year  int
	Month int
	// CreatedAt is when the snapshot was made. It is used as the PDF's
	// creation date so a slip downloaded twice is the same file.
	CreatedAt time.Time

	Components []Component
	// TotalWithoutDebt is printed only when it differs from Total.
	TotalWithoutDebt decimal.Decimal
	Total            decimal.Decimal
	ExtraInfos       []ExtraInfo
}

// Component is a row of the slip, in the snapshot's order.
type Component struct {
	Description string
	// Multiplier is printed as Jumlah, Amount as Fee / (Penalti).
	Multiplier decimal.Decimal
	Amount     decimal.Decimal
	Total      decimal.Decimal
}

// ExtraInfo is a line of Informasi Tambahan.
type ExtraInfo struct {
	Title       string
	Description string
}

// File is a rendered slip or archive of slips.
type File struct {
	Filename string
	Content  []byte
}
