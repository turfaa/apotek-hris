package payslip

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var monthNames = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// periodName returns the month in Indonesian, e.g. "Oktober 2026".
func periodName(year, month int) string {
	if month < 1 || month > 12 {
		return fmt.Sprintf("%04d-%02d", year, month)
	}

	return fmt.Sprintf("%s %d", monthNames[month-1], year)
}

// rupiah formats an amount the Indonesian way: "Rp1.234.567", with ",50"
// only when there is a fraction, and negatives in parentheses, e.g.
// "(Rp50.000)", like Fee / (Penalti) on the dashboard.
func rupiah(d decimal.Decimal) string {
	s := "Rp" + formatNumber(d.Abs(), 2, true)
	if d.IsNegative() {
		return "(" + s + ")"
	}

	return s
}

// quantity formats a multiplier the Indonesian way, e.g. "1.250" or "1,5",
// with a minus sign when negative.
func quantity(d decimal.Decimal) string {
	s := formatNumber(d.Abs(), 2, false)
	if d.IsNegative() {
		return "-" + s
	}

	return s
}

// formatNumber writes a non-negative d with "." between thousands and ","
// before the decimals, rounded to places. Whole numbers have no decimals.
// With fixed, a fraction always has all its places ("1,50"); otherwise
// trailing zeros are dropped ("1,5").
func formatNumber(d decimal.Decimal, places int32, fixed bool) string {
	d = d.Round(places)

	whole := d.Truncate(0)
	fraction := d.Sub(whole)

	intPart := groupThousands(whole.String())
	if fraction.IsZero() {
		return intPart
	}

	frac := strings.TrimPrefix(fraction.StringFixed(places), "0.")
	if !fixed {
		frac = strings.TrimRight(frac, "0")
	}

	return intPart + "," + frac
}

func groupThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}

	var b strings.Builder
	head := len(digits) % 3
	if head > 0 {
		b.WriteString(digits[:head])
	}
	for i := head; i < len(digits); i += 3 {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(digits[i : i+3])
	}

	return b.String()
}

// terbilangRupiah writes a whole rupiah amount in words, e.g. "Satu juta dua
// ratus ribu rupiah". Fractions are dropped: slip totals are whole rupiah.
func terbilangRupiah(d decimal.Decimal) string {
	words := terbilang(d.Abs().Truncate(0).BigInt().Uint64()) + " rupiah"
	if d.Truncate(0).IsNegative() {
		words = "minus " + words
	}

	return strings.ToUpper(words[:1]) + words[1:]
}
