package salary

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/turfaa/apotek-hris/internal/salary/payslip"
	"github.com/turfaa/apotek-hris/pkg/timex"
)

// ErrNoSnapshots is returned when there are no snapshots to put in a ZIP of
// slips.
var ErrNoSnapshots = errors.New("no salary snapshots for this month")

// RenderSlip renders the pay slip PDF of a snapshot. It returns
// payslip.ErrNotConfigured when the letterhead is not set, and
// sql.ErrNoRows when the snapshot does not exist.
func (s *Service) RenderSlip(ctx context.Context, snapshotID int64) (payslip.File, error) {
	if !s.letterhead.Configured() {
		return payslip.File{}, payslip.ErrNotConfigured
	}

	snapshot, err := s.db.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return payslip.File{}, fmt.Errorf("get snapshot %d: %w", snapshotID, err)
	}

	name, err := s.employeeName(ctx, snapshot.EmployeeID)
	if err != nil {
		return payslip.File{}, err
	}

	return s.renderSlip(snapshot, name)
}

// RenderSlipsZip renders the slips of every snapshot of a month, of one
// employee when employeeID is set, into a ZIP archive. It returns
// ErrNoSnapshots when there are none.
func (s *Service) RenderSlipsZip(ctx context.Context, month timex.Month, employeeID *int64) (payslip.File, error) {
	if !s.letterhead.Configured() {
		return payslip.File{}, payslip.ErrNotConfigured
	}

	snapshots, err := s.db.GetSnapshots(ctx, GetSnapshotsRequest{EmployeeID: employeeID, Month: &month})
	if err != nil {
		return payslip.File{}, fmt.Errorf("get snapshots: %w", err)
	}
	if len(snapshots) == 0 {
		return payslip.File{}, ErrNoSnapshots
	}

	// Oldest first, so a later duplicate of an employee's month is the one
	// named "(2)".
	slices.SortFunc(snapshots, func(a, b Snapshot) int { return cmp.Compare(a.ID, b.ID) })

	names, err := s.employeeNames(ctx, snapshots)
	if err != nil {
		return payslip.File{}, err
	}

	files := make([]payslip.File, len(snapshots))
	for i, snapshot := range snapshots {
		files[i], err = s.renderSlip(snapshot, names[snapshot.EmployeeID])
		if err != nil {
			return payslip.File{}, err
		}
	}

	// The archive's entries are dated by the newest snapshot, so the same
	// snapshots give the same archive.
	var newest time.Time
	for _, snapshot := range snapshots {
		if snapshot.CreatedAt.After(newest) {
			newest = snapshot.CreatedAt
		}
	}

	content, err := payslip.Zip(files, newest)
	if err != nil {
		return payslip.File{}, fmt.Errorf("zip slips: %w", err)
	}

	return payslip.File{Filename: payslip.ZipFilename(month.Year, month.Month), Content: content}, nil
}

func (s *Service) renderSlip(snapshot Snapshot, employeeName string) (payslip.File, error) {
	doc := payslip.Document{
		Letterhead:       s.letterhead,
		SnapshotID:       snapshot.ID,
		EmployeeName:     employeeName,
		Year:             snapshot.Month.Year,
		Month:            snapshot.Month.Month,
		CreatedAt:        snapshot.CreatedAt,
		Components:       make([]payslip.Component, len(snapshot.Salary.Components)),
		TotalWithoutDebt: snapshot.Salary.TotalWithoutDebt(),
		Total:            snapshot.Salary.Total(),
		ExtraInfos:       make([]payslip.ExtraInfo, len(snapshot.Salary.ExtraInfos)),
	}

	for i, c := range snapshot.Salary.Components {
		doc.Components[i] = payslip.Component{
			Description: c.Description,
			Multiplier:  c.Multiplier,
			Amount:      c.Amount,
			Total:       c.Total(),
		}
	}

	for i, info := range snapshot.Salary.ExtraInfos {
		doc.ExtraInfos[i] = payslip.ExtraInfo{Title: info.Title, Description: info.Description}
	}

	content, err := payslip.Render(doc)
	if err != nil {
		return payslip.File{}, fmt.Errorf("render slip of snapshot %d: %w", snapshot.ID, err)
	}

	return payslip.File{
		Filename: payslip.SlipFilename(snapshot.Month.Year, snapshot.Month.Month, employeeName),
		Content:  content,
	}, nil
}

// employeeName returns the employee's current name, or "Karyawan #id" when
// the employee no longer exists, so a slip can always be rendered.
func (s *Service) employeeName(ctx context.Context, employeeID int64) (string, error) {
	employee, err := s.hrisService.GetEmployee(ctx, employeeID)
	if errors.Is(err, sql.ErrNoRows) {
		return unknownEmployeeName(employeeID), nil
	}
	if err != nil {
		return "", fmt.Errorf("get employee %d: %w", employeeID, err)
	}

	return employee.Name, nil
}

// employeeNames returns the names of the snapshots' employees by id, with
// "Karyawan #id" for those that no longer exist.
func (s *Service) employeeNames(ctx context.Context, snapshots []Snapshot) (map[int64]string, error) {
	var ids []int64
	for _, snapshot := range snapshots {
		if !slices.Contains(ids, snapshot.EmployeeID) {
			ids = append(ids, snapshot.EmployeeID)
		}
	}

	employees, err := s.hrisService.GetEmployeesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get employees: %w", err)
	}

	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		names[id] = unknownEmployeeName(id)
	}
	for _, employee := range employees {
		names[employee.ID] = employee.Name
	}

	return names, nil
}

func unknownEmployeeName(id int64) string {
	return fmt.Sprintf("Karyawan #%d", id)
}
