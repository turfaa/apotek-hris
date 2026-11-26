package payslip

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-pdf/fpdf"
)

var (
	//go:embed assets/Carlito-Regular.ttf
	carlitoRegular []byte
	//go:embed assets/Carlito-Bold.ttf
	carlitoBold []byte
	//go:embed assets/logo.png
	logoPNG []byte
)

const (
	fontFamily = "Carlito"
	logoName   = "logo"

	pageMarginTop    = 15.0
	pageMarginSide   = 20.0
	pageMarginBottom = 22.0

	bodyFontSize   = 12.0
	bodyLineHeight = 6.0

	tableFontSize   = 11.0
	tableLineHeight = 5.0
	tablePadding    = 1.2

	fieldLabelWidth = 30.0
	fieldColonWidth = 4.0

	extraInfoLabelWidth = 55.0

	footerFontSize = 9.0
)

// Footer lines, printed at the bottom of every page.
const (
	footerGenerated    = "Dokumen ini dibuat otomatis dan sah tanpa tanda tangan."
	footerConfidential = "Rahasia — hanya untuk karyawan ybs."
)

// headerFill is the grey of the table header and the take-home total.
var headerFill = [3]int{235, 235, 235}

// Render writes the slip as an A4 portrait PDF.
func Render(doc Document) ([]byte, error) {
	if !doc.Letterhead.Configured() {
		return nil, ErrNotConfigured
	}

	s := newSlip(pdfSafeDocument(doc))
	s.writeLetterhead()
	s.writeTitle()
	s.writeFields()
	s.writeComponents()
	s.writeExtraInfos()

	var buf bytes.Buffer
	if err := s.pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("write pdf: %w", err)
	}

	return buf.Bytes(), nil
}

// slip is a pay slip PDF being written.
type slip struct {
	pdf *fpdf.Fpdf
	doc Document
}

func newSlip(doc Document) *slip {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pageMarginSide, pageMarginTop, pageMarginSide)
	pdf.SetAutoPageBreak(true, pageMarginBottom)
	pdf.SetCellMargin(0)

	// Fixed dates and catalog order keep the output the same every time a
	// snapshot's slip is downloaded.
	pdf.SetCreationDate(doc.CreatedAt)
	pdf.SetModificationDate(doc.CreatedAt)
	pdf.SetCatalogSort(true)
	pdf.SetTitle(fmt.Sprintf("Slip Gaji %s - %s", periodName(doc.Year, doc.Month), doc.EmployeeName), true)
	pdf.SetAuthor(doc.Letterhead.Name, true)

	pdf.AddUTF8FontFromBytes(fontFamily, "", carlitoRegular)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", carlitoBold)
	pdf.RegisterImageOptionsReader(logoName, fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(logoPNG))

	s := &slip{pdf: pdf, doc: doc}
	pdf.SetFooterFunc(s.writeFooter)
	pdf.AddPage()

	return s
}

func (s *slip) contentWidth() float64 {
	w, _ := s.pdf.GetPageSize()
	return w - 2*pageMarginSide
}

func (s *slip) pageBottom() float64 {
	_, h := s.pdf.GetPageSize()
	return h - pageMarginBottom
}

// ensureSpace starts a new page when height does not fit on this one.
func (s *slip) ensureSpace(height float64) bool {
	if s.pdf.GetY()+height <= s.pageBottom() {
		return false
	}

	s.pdf.AddPage()
	return true
}

// writeLetterhead writes the logo, the pharmacy's name, license number,
// address and contacts, then a double rule.
func (s *slip) writeLetterhead() {
	pdf := s.pdf
	lh := s.doc.Letterhead

	const (
		logoHeight     = 14.0
		logoGap        = 5.0
		nameFontSize   = 12.0
		nameHeight     = 5.5
		detailFontSize = 9.5
		lineHeight     = 4.0
	)

	info := pdf.GetImageInfo(logoName)
	logoWidth := logoHeight * info.Width() / info.Height()

	textX := pageMarginSide + logoWidth + logoGap
	textW := s.contentWidth() - logoWidth - logoGap

	pdf.SetFont(fontFamily, "", detailFontSize)
	var lines []string
	lines = append(lines, "Nomor Izin: "+lh.LicenseNumber)
	lines = append(lines, wrapText(pdf, lh.Address, textW)...)
	lines = append(lines, fmt.Sprintf("Email: %s, Telepon: %s", lh.Email, lh.Phone))

	textHeight := nameHeight + float64(len(lines))*lineHeight
	blockHeight := max(logoHeight, textHeight)

	top := pdf.GetY()
	pdf.ImageOptions(logoName, pageMarginSide, top+(blockHeight-logoHeight)/2, logoWidth, logoHeight, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	pdf.SetXY(textX, top+(blockHeight-textHeight)/2)
	pdf.SetFont(fontFamily, "B", nameFontSize)
	pdf.CellFormat(textW, nameHeight, strings.ToUpper(lh.Name), "", 2, "L", false, 0, "")

	pdf.SetFont(fontFamily, "", detailFontSize)
	for _, line := range lines {
		pdf.CellFormat(textW, lineHeight, line, "", 2, "L", false, 0, "")
	}

	ruleY := top + blockHeight + 2
	right := pageMarginSide + s.contentWidth()

	pdf.SetLineWidth(0.6)
	pdf.Line(pageMarginSide, ruleY, right, ruleY)
	pdf.SetLineWidth(0.2)
	pdf.Line(pageMarginSide, ruleY+1, right, ruleY+1)

	pdf.SetXY(pageMarginSide, ruleY+1)
}

// writeTitle writes the centered SLIP GAJI and the period.
func (s *slip) writeTitle() {
	pdf := s.pdf

	pdf.Ln(7)
	pdf.SetFont(fontFamily, "B", 16)
	pdf.CellFormat(0, 8, "SLIP GAJI", "", 1, "C", false, 0, "")
	pdf.SetFont(fontFamily, "", bodyFontSize)
	pdf.CellFormat(0, bodyLineHeight, "Periode "+periodName(s.doc.Year, s.doc.Month), "", 1, "C", false, 0, "")
}

// writeFields writes who and which period the slip is for, with the labels
// and colons lined up.
func (s *slip) writeFields() {
	pdf := s.pdf
	valueX := pageMarginSide + fieldLabelWidth + fieldColonWidth
	valueW := s.contentWidth() - fieldLabelWidth - fieldColonWidth

	fields := []struct {
		label, value string
		bold         bool
	}{
		{"Nama", s.doc.EmployeeName, true},
		{"Periode", periodName(s.doc.Year, s.doc.Month), false},
		{"No. Snapshot", fmt.Sprintf("%d", s.doc.SnapshotID), false},
	}

	pdf.Ln(6)
	for _, f := range fields {
		pdf.SetFont(fontFamily, "", bodyFontSize)
		pdf.SetX(pageMarginSide)
		pdf.CellFormat(fieldLabelWidth, bodyLineHeight, f.label, "", 0, "L", false, 0, "")
		pdf.CellFormat(fieldColonWidth, bodyLineHeight, ":", "", 0, "L", false, 0, "")

		style := ""
		if f.bold {
			style = "B"
		}
		pdf.SetFont(fontFamily, style, bodyFontSize)
		for i, line := range wrapText(pdf, f.value, valueW) {
			if i > 0 {
				pdf.SetX(valueX)
			}
			pdf.CellFormat(valueW, bodyLineHeight, line, "", 1, "L", false, 0, "")
		}
	}
}

// column is a column of the component table.
type column struct {
	header string
	// width in mm; 0 takes what the other columns leave.
	width float64
	align string
}

var columns = []column{
	{header: "No", width: 10, align: "C"},
	{header: "Komponen", align: "L"},
	{header: "Jumlah", width: 20, align: "R"},
	{header: "Fee / (Penalti)", width: 33, align: "R"},
	{header: "Total", width: 33, align: "R"},
}

func (s *slip) columnWidths() []float64 {
	fixed := 0.0
	for _, c := range columns {
		fixed += c.width
	}

	widths := make([]float64, len(columns))
	for i, c := range columns {
		widths[i] = c.width
		if c.width == 0 {
			widths[i] = s.contentWidth() - fixed
		}
	}

	return widths
}

// tableRow is a row of the component table, its cells wrapped to their
// columns.
type tableRow struct {
	cells [][]string
	style string
	fill  bool
}

func (r tableRow) height() float64 {
	lines := 1
	for _, cell := range r.cells {
		lines = max(lines, len(cell))
	}

	return float64(lines)*tableLineHeight + 2*tablePadding
}

func (s *slip) row(widths []float64, style string, fill bool, cells ...string) tableRow {
	s.pdf.SetFont(fontFamily, style, tableFontSize)

	r := tableRow{cells: make([][]string, len(cells)), style: style, fill: fill}
	for i, c := range cells {
		r.cells[i] = wrapText(s.pdf, c, widths[i]-2*tablePadding)
	}

	return r
}

// writeComponents writes the component table and its totals. The header is
// repeated on every page, and the totals never start a page alone: the last
// component moves with them.
func (s *slip) writeComponents() {
	widths := s.columnWidths()

	header := s.row(widths, "B", true, headerCells()...)
	rows := make([]tableRow, len(s.doc.Components))
	for i, c := range s.doc.Components {
		rows[i] = s.row(widths, "", false,
			fmt.Sprintf("%d", i+1),
			c.Description,
			quantity(c.Multiplier),
			rupiah(c.Amount),
			rupiah(c.Total),
		)
	}

	// The totals span the first four columns.
	totalWidths := []float64{widths[0] + widths[1] + widths[2] + widths[3], widths[4]}
	var totals []tableRow
	if !s.doc.TotalWithoutDebt.Equal(s.doc.Total) {
		totals = append(totals, s.row(totalWidths, "", false, "Total Gaji (Tanpa Utang)", rupiah(s.doc.TotalWithoutDebt)))
	}
	totals = append(totals, s.row(totalWidths, "B", true, "TOTAL GAJI DITERIMA", rupiah(s.doc.Total)))

	totalsHeight := 0.0
	for _, t := range totals {
		totalsHeight += t.height()
	}
	terbilang := "Terbilang: " + terbilangRupiah(s.doc.Total)
	s.pdf.SetFont(fontFamily, "", bodyFontSize-1)
	terbilangLines := wrapText(s.pdf, terbilang, s.contentWidth())
	tailHeight := totalsHeight + 2 + float64(len(terbilangLines))*bodyLineHeight

	s.pdf.Ln(6)
	s.ensureSpace(header.height() + tailHeight)
	s.writeRow(widths, header, columnAligns())

	for i, r := range rows {
		needed := r.height()
		if i == len(rows)-1 {
			needed += tailHeight
		}
		if s.ensureSpace(needed) {
			s.writeRow(widths, header, columnAligns())
		}
		s.writeRow(widths, r, columnAligns())
	}

	if len(rows) == 0 && s.ensureSpace(tailHeight) {
		s.writeRow(widths, header, columnAligns())
	}
	for _, t := range totals {
		s.writeRow(totalWidths, t, []string{"R", "R"})
	}

	s.pdf.Ln(2)
	s.pdf.SetFont(fontFamily, "", bodyFontSize-1)
	for _, line := range terbilangLines {
		s.pdf.CellFormat(0, bodyLineHeight, line, "", 1, "L", false, 0, "")
	}
}

func headerCells() []string {
	cells := make([]string, len(columns))
	for i, c := range columns {
		cells[i] = c.header
	}

	return cells
}

func columnAligns() []string {
	aligns := make([]string, len(columns))
	for i, c := range columns {
		aligns[i] = c.align
	}

	return aligns
}

func (s *slip) writeRow(widths []float64, r tableRow, aligns []string) {
	pdf := s.pdf
	h := r.height()
	y := pdf.GetY()

	pdf.SetFont(fontFamily, r.style, tableFontSize)
	pdf.SetLineWidth(0.2)
	pdf.SetFillColor(headerFill[0], headerFill[1], headerFill[2])

	style := "D"
	if r.fill {
		style = "FD"
	}

	x := pageMarginSide
	for i := range r.cells {
		pdf.Rect(x, y, widths[i], h, style)
		for j, line := range r.cells[i] {
			pdf.SetXY(x+tablePadding, y+tablePadding+float64(j)*tableLineHeight)
			pdf.CellFormat(widths[i]-2*tablePadding, tableLineHeight, line, "", 0, aligns[i], false, 0, "")
		}
		x += widths[i]
	}

	pdf.SetXY(pageMarginSide, y+h)
}

// writeExtraInfos writes Informasi Tambahan, when the snapshot has any.
func (s *slip) writeExtraInfos() {
	if len(s.doc.ExtraInfos) == 0 {
		return
	}

	pdf := s.pdf
	valueX := pageMarginSide + extraInfoLabelWidth + fieldColonWidth
	valueW := s.contentWidth() - extraInfoLabelWidth - fieldColonWidth
	const headingHeight = 8.0

	pdf.Ln(6)
	for i, info := range s.doc.ExtraInfos {
		pdf.SetFont(fontFamily, "", bodyFontSize-1)
		labels := wrapText(pdf, info.Title, extraInfoLabelWidth-2)
		values := wrapText(pdf, info.Description, valueW)
		height := float64(max(len(labels), len(values))) * bodyLineHeight

		if i == 0 {
			// Keep the heading with the first line.
			s.ensureSpace(headingHeight + height)
			pdf.SetFont(fontFamily, "B", bodyFontSize+1)
			pdf.CellFormat(0, headingHeight, "Informasi Tambahan", "", 1, "L", false, 0, "")
		} else {
			s.ensureSpace(height)
		}

		pdf.SetFont(fontFamily, "", bodyFontSize-1)
		top := pdf.GetY()
		for j, line := range labels {
			pdf.SetXY(pageMarginSide, top+float64(j)*bodyLineHeight)
			pdf.CellFormat(extraInfoLabelWidth, bodyLineHeight, line, "", 0, "L", false, 0, "")
		}
		pdf.SetXY(pageMarginSide+extraInfoLabelWidth, top)
		pdf.CellFormat(fieldColonWidth, bodyLineHeight, ":", "", 0, "L", false, 0, "")
		for j, line := range values {
			pdf.SetXY(valueX, top+float64(j)*bodyLineHeight)
			pdf.CellFormat(valueW, bodyLineHeight, line, "", 0, "L", false, 0, "")
		}
		pdf.SetXY(pageMarginSide, top+height)
	}
}

// writeFooter writes a rule and the two footer lines at the bottom of the
// page.
func (s *slip) writeFooter() {
	pdf := s.pdf
	half := s.contentWidth() / 2

	pdf.SetY(-16)
	pdf.SetDrawColor(153, 153, 153)
	pdf.SetLineWidth(0.2)
	y := pdf.GetY()
	pdf.Line(pageMarginSide, y, pageMarginSide+s.contentWidth(), y)
	pdf.SetDrawColor(0, 0, 0)

	pdf.SetXY(pageMarginSide, y+2)
	pdf.SetFont(fontFamily, "", footerFontSize)
	pdf.SetTextColor(68, 68, 68)
	pdf.CellFormat(half, 4.5, footerGenerated, "", 0, "L", false, 0, "")
	pdf.CellFormat(half, 4.5, footerConfidential, "", 0, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

// wrapText splits text into lines no wider than width in the current font,
// breaking at spaces, and inside words only when a word is too long.
func wrapText(pdf *fpdf.Fpdf, text string, width float64) []string {
	var lines []string

	for _, paragraph := range strings.Split(text, "\n") {
		current := ""
		for _, word := range strings.Fields(paragraph) {
			candidate := word
			if current != "" {
				candidate = current + " " + word
			}

			if pdf.GetStringWidth(candidate) <= width {
				current = candidate
				continue
			}

			if current != "" {
				lines = append(lines, current)
				current = ""
			}

			for pdf.GetStringWidth(word) > width {
				cut := fittingPrefix(pdf, word, width)
				lines = append(lines, word[:cut])
				word = word[cut:]
			}
			current = word
		}

		if current != "" {
			lines = append(lines, current)
		}
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

// fittingPrefix returns the byte length of the longest prefix of word that
// fits in width, but at least one rune.
func fittingPrefix(pdf *fpdf.Fpdf, word string, width float64) int {
	cut := 0
	for i, r := range word {
		end := i + utf8.RuneLen(r)
		if cut > 0 && pdf.GetStringWidth(word[:end]) > width {
			break
		}
		cut = end
	}

	return cut
}

// pdfSafeDocument returns the document with every printed text made safe for
// fpdf.
func pdfSafeDocument(doc Document) Document {
	doc.EmployeeName = pdfSafe(doc.EmployeeName)

	doc.Letterhead.Name = pdfSafe(doc.Letterhead.Name)
	doc.Letterhead.LicenseNumber = pdfSafe(doc.Letterhead.LicenseNumber)
	doc.Letterhead.Address = pdfSafe(doc.Letterhead.Address)
	doc.Letterhead.Email = pdfSafe(doc.Letterhead.Email)
	doc.Letterhead.Phone = pdfSafe(doc.Letterhead.Phone)

	components := make([]Component, len(doc.Components))
	for i, c := range doc.Components {
		c.Description = pdfSafe(c.Description)
		components[i] = c
	}
	doc.Components = components

	infos := make([]ExtraInfo, len(doc.ExtraInfos))
	for i, info := range doc.ExtraInfos {
		info.Title = pdfSafe(info.Title)
		info.Description = pdfSafe(info.Description)
		infos[i] = info
	}
	doc.ExtraInfos = infos

	return doc
}

// pdfSafe removes the characters fpdf cannot write: those outside the Basic
// Multilingual Plane (e.g. emoji), which make it panic, and control
// characters other than newlines.
func pdfSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r > 0xFFFF || r == utf8.RuneError || (unicode.IsControl(r) && r != '\n') {
			return -1
		}
		return r
	}, s)
}
