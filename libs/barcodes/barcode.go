package barcodes

import qrcode "github.com/skip2/go-qrcode"

func QRPNG(value string, size int) ([]byte, error) {
	img, err := qrcode.Encode(value, qrcode.Medium, size)
	if err != nil {
		return nil, err
	}
	return img, nil
}
