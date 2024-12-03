package templates

import (
	_ "embed"
	"html/template"
)

//go:embed work_log_for_patient.html
var workLogForPatientTemplate string

var WorkLogForPatient = template.Must(template.New("work_log_for_patient.html").Parse(workLogForPatientTemplate))

type WorkLogForPatientData struct {
	PatientName  string
	Date         string
	Place        string
	EmployeeName string
	Units        []WorkLogUnitForPatientData
	Notes        string
}

type WorkLogUnitForPatientData struct {
	WorkType    string
	WorkOutcome string
	OutcomeUnit string
	Notes       string
}
