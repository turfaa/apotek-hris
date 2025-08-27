package attendance

type PayableType string

const (
	// PayableTypeWorking means the employee is working and will be paid with their shift fee.
	PayableTypeWorking PayableType = "working"

	// PayableTypeBenefit means the employee is not working but will be paid with their shift fee as their benefit.
	PayableTypeBenefit PayableType = "benefit"

	// PayableTypeNone means the employee is not working and will not be paid.
	PayableTypeNone PayableType = "none"
)

func (p PayableType) IsValid() bool {
	switch p {
	case PayableTypeWorking, PayableTypeBenefit, PayableTypeNone:
		return true
	default:
		return false
	}
}

func PayableTypes() []PayableType {
	return []PayableType{
		PayableTypeWorking,
		PayableTypeBenefit,
		PayableTypeNone,
	}
}
