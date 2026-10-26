package helpers

import (
	"encoding/json"
	"errors"
	"strings"
)

type StudentQR struct {
	InstitutionCode string `json:"institution_code"`
	Barcode         string `json:"barcode"`
}

// ParseStudentQR accepts the JSON payload printed on student QR cards, or a raw barcode value.
func ParseStudentQR(value string) (StudentQR, error) {
	var qr StudentQR
	value = strings.TrimSpace(value)
	if err := json.Unmarshal([]byte(value), &qr); err == nil && strings.TrimSpace(qr.Barcode) != "" {
		qr.Barcode = strings.TrimSpace(qr.Barcode)
		qr.InstitutionCode = strings.TrimSpace(qr.InstitutionCode)
		return qr, nil
	}
	if value != "" && !strings.HasPrefix(value, "{") {
		return StudentQR{Barcode: value}, nil
	}
	return StudentQR{}, errors.New("invalid QR code")
}
