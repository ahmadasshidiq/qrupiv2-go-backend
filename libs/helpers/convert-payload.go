package helpers

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func intFromPayload(payload map[string]interface{}, key string, fallback int) int {
	if val, ok := payload[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			i, _ := strconv.Atoi(v)
			return i
		}
	}
	return fallback
}

func strFromPayload(payload map[string]interface{}, key string, fallback string) string {
	if val, ok := payload[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return fallback
}

func excelDateToTime(excelDate float64) time.Time {
	baseDate := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	duration := time.Duration(excelDate * 24 * float64(time.Hour))
	return baseDate.Add(duration)
}

func ScanRowsToMap(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})

		for i, col := range cols {
			val := values[i]

			switch v := val.(type) {

			// NULL → "-"
			case nil:
				row[col] = "-"

			// []byte → string
			case []byte:
				row[col] = string(v)

			// time.Time → formatted string
			case time.Time:
				row[col] = v.Format("2006-01-02 15:04:05")

			// float64 → kemungkinan Excel date
			case float64:
				lowerCol := strings.ToLower(col)

				if strings.Contains(lowerCol, "date") ||
					strings.Contains(lowerCol, "created") ||
					strings.Contains(lowerCol, "time") {

					t := excelDateToTime(v)
					row[col] = t.Format("2006-01-02 15:04:05")
				} else {
					row[col] = v
				}

			// default
			default:
				row[col] = v
			}
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func Singular(word string) string {
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "s") {
		return word[:len(word)-1]
	}
	return word
}

func ChooseUUID(value, fallback uuid.UUID) uuid.UUID {
	if value == uuid.Nil {
		return fallback
	}
	return value
}

func ChooseString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func GenerateInstitutionCode(name string) string {
	// Pisah nama berdasarkan spasi
	words := strings.Fields(name)

	// Ambil huruf pertama dari setiap kata
	code := ""
	for _, w := range words {
		if len(w) > 0 {
			code += strings.ToUpper(string(w[0]))
		}
	}

	// Buat angka acak 4 digit
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(9000) + 1000 // menghasilkan 1000–9999

	return code + "-" + strconv.Itoa(num)
}

// slugify singkat untuk ubah nama group jadi bentuk kode
func slugify(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := re.ReplaceAllString(input, "")
	if len(slug) > 8 {
		slug = slug[:8] // batasi panjang
	}
	return strings.ToUpper(slug)
}

// GenerateQRCode dengan nama grup, tipe, dan tanggal
func GenerateQRCode(groupName, qrType string) string {
	b := make([]byte, 3)
	_, err := rand.Read(b)
	if err != nil {
		return "QR-000000"
	}

	shortGroup := slugify(groupName)
	shortType := strings.ToUpper(qrType[:3])  // ATT atau QUI
	datePart := time.Now().Format("20060102") // YYYYMMDD
	randomPart := hex.EncodeToString(b)

	return fmt.Sprintf("QR-%s-%s-%s-%s", shortGroup, shortType, datePart, randomPart)
}

func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3 // radius bumi dalam meter
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c // hasil meter
}
