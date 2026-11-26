package payslip

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestSlipFilename(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"Budi Santoso", "2026-10 - Budi Santoso - Slip Gaji.pdf"},
		{"  Budi   Santoso\t", "2026-10 - Budi Santoso - Slip Gaji.pdf"},
		{"A/B\\C:D*E?F\"G|H<I>J", "2026-10 - A-B-C-D-E-F-G-H-I-J - Slip Gaji.pdf"},
		{"Siti\x00Aminah", "2026-10 - Siti-Aminah - Slip Gaji.pdf"},
		{"Zoë Müller", "2026-10 - Zoë Müller - Slip Gaji.pdf"},
	}

	for _, c := range cases {
		if got := SlipFilename(2026, 10, c.name); got != c.want {
			t.Errorf("SlipFilename(%q) = %q, want %q", c.name, got, c.want)
		}
	}

	if got, want := ZipFilename(2026, 3), "2026-03 - Slip Gaji.zip"; got != want {
		t.Errorf("ZipFilename = %q, want %q", got, want)
	}
}

func TestContentDisposition(t *testing.T) {
	got := ContentDisposition("2026-10 - Zoë (A) - Slip Gaji.pdf")
	want := `attachment; filename="2026-10 - Zo_ (A) - Slip Gaji.pdf"; filename*=UTF-8''2026-10%20-%20Zo%C3%AB%20%28A%29%20-%20Slip%20Gaji.pdf`
	if got != want {
		t.Errorf("ContentDisposition =\n %s\nwant\n %s", got, want)
	}
}

func TestRupiah(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0", "Rp0"},
		{"999", "Rp999"},
		{"1000", "Rp1.000"},
		{"1234567", "Rp1.234.567"},
		{"-50000", "(Rp50.000)"},
		{"1234.5", "Rp1.234,50"},
		{"1234.567", "Rp1.234,57"},
		{"-0.4", "(Rp0,40)"},
	}

	for _, c := range cases {
		if got := rupiah(decimal.RequireFromString(c.in)); got != c.want {
			t.Errorf("rupiah(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestQuantity(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0", "0"},
		{"22", "22"},
		{"1.5", "1,5"},
		{"1250", "1.250"},
		{"-1", "-1"},
		{"0.333333", "0,33"},
	}

	for _, c := range cases {
		if got := quantity(decimal.RequireFromString(c.in)); got != c.want {
			t.Errorf("quantity(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTerbilang(t *testing.T) {
	raw, err := os.ReadFile("testdata/terbilang.json")
	if err != nil {
		t.Fatalf("read cases: %v", err)
	}

	var cases [][2]any
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse cases: %v", err)
	}

	for _, c := range cases {
		n, want := uint64(c[0].(float64)), c[1].(string)
		if got := terbilang(n); got != want {
			t.Errorf("terbilang(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestTerbilangRupiah(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1200000", "Satu juta dua ratus ribu rupiah"},
		{"0", "Nol rupiah"},
		{"-15000", "Minus lima belas ribu rupiah"},
	}

	for _, c := range cases {
		if got := terbilangRupiah(decimal.RequireFromString(c.in)); got != c.want {
			t.Errorf("terbilangRupiah(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPeriodName(t *testing.T) {
	if got := periodName(2026, 10); got != "Oktober 2026" {
		t.Errorf("periodName = %q", got)
	}
}

func testLetterhead() Letterhead {
	return Letterhead{
		Name:          "Apotek Contoh",
		LicenseNumber: "0000000000",
		Address:       "Jalan Contoh No. 1, Kota Contoh",
		Email:         "kontak@example.com",
		Phone:         "0800000000",
	}
}

func testDocument() Document {
	d := decimal.RequireFromString
	return Document{
		Letterhead:   testLetterhead(),
		SnapshotID:   42,
		EmployeeName: "Budi Santoso",
		Year:         2026,
		Month:        10,
		CreatedAt:    time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC),
		Components: []Component{
			{Description: "Banyak shift jaga", Multiplier: d("22"), Amount: d("90000"), Total: d("1980000")},
			{Description: "Banyak jam lembur", Multiplier: d("3.5"), Amount: d("14286"), Total: d("50001")},
			{Description: "Terlambat", Multiplier: d("2"), Amount: d("-10000"), Total: d("-20000")},
			{Description: "Potongan utang 🙂", Multiplier: d("1"), Amount: d("-100000"), Total: d("-100000")},
		},
		TotalWithoutDebt: d("2010001"),
		Total:            d("1910001"),
		ExtraInfos: []ExtraInfo{
			{Title: "Sisa Cuti", Description: "10 hari"},
		},
	}
}

func TestRender(t *testing.T) {
	start := time.Now()
	pdf, err := Render(testDocument())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 300*time.Millisecond {
		t.Logf("Render took %s, over the 300 ms target", elapsed)
	}

	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("output does not start with %%PDF: %q", pdf[:min(len(pdf), 16)])
	}

	// PAYSLIP_SAMPLE=/path/to/slip.pdf go test -run TestRender$ writes the
	// slip, to look at the layout.
	if path := os.Getenv("PAYSLIP_SAMPLE"); path != "" {
		if err := os.WriteFile(path, pdf, 0o644); err != nil {
			t.Fatalf("write sample: %v", err)
		}
	}

	again, err := Render(testDocument())
	if err != nil {
		t.Fatalf("Render again: %v", err)
	}
	if !bytes.Equal(pdf, again) {
		t.Error("rendering the same document twice gave different PDFs")
	}
}

func TestRenderManyComponents(t *testing.T) {
	doc := testDocument()
	for i := range 80 {
		doc.Components = append(doc.Components, Component{
			Description: fmt.Sprintf("Komponen tambahan %d dengan keterangan yang cukup panjang supaya baris ini terbungkus", i),
			Multiplier:  decimal.NewFromInt(1),
			Amount:      decimal.NewFromInt(1000),
			Total:       decimal.NewFromInt(1000),
		})
	}
	for i := range 40 {
		doc.ExtraInfos = append(doc.ExtraInfos, ExtraInfo{Title: fmt.Sprintf("Info %d", i), Description: "Keterangan"})
	}

	if _, err := Render(doc); err != nil {
		t.Fatalf("Render: %v", err)
	}
}

func TestRenderEmpty(t *testing.T) {
	doc := testDocument()
	doc.Components = nil
	doc.ExtraInfos = nil
	doc.Total = decimal.Zero
	doc.TotalWithoutDebt = decimal.Zero

	if _, err := Render(doc); err != nil {
		t.Fatalf("Render: %v", err)
	}
}

func TestRenderNotConfigured(t *testing.T) {
	doc := testDocument()
	doc.Letterhead.Phone = " "

	if _, err := Render(doc); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Render error = %v, want ErrNotConfigured", err)
	}
}

func TestZipDuplicateNames(t *testing.T) {
	files := []File{
		{Filename: "2026-10 - Siti - Slip Gaji.pdf", Content: []byte("siti")},
		{Filename: "2026-10 - Budi - Slip Gaji.pdf", Content: []byte("budi 1")},
		{Filename: "2026-10 - Budi - Slip Gaji.pdf", Content: []byte("budi 2")},
		{Filename: "2026-10 - Budi - Slip Gaji.pdf", Content: []byte("budi 3")},
	}

	content, err := Zip(files, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}

	want := []struct{ name, content string }{
		{"2026-10 - Budi - Slip Gaji (2).pdf", "budi 2"},
		{"2026-10 - Budi - Slip Gaji (3).pdf", "budi 3"},
		{"2026-10 - Budi - Slip Gaji.pdf", "budi 1"},
		{"2026-10 - Siti - Slip Gaji.pdf", "siti"},
	}
	if len(r.File) != len(want) {
		t.Fatalf("zip has %d files, want %d", len(r.File), len(want))
	}

	for i, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		got, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}

		if f.Name != want[i].name || string(got) != want[i].content {
			t.Errorf("file %d = %q (%q), want %q (%q)", i, f.Name, got, want[i].name, want[i].content)
		}
	}
}
