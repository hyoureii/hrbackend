package lib

import (
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"
)

const openapp = "https://hr.hyourei.xyz/openapp/scan"

func GenerateAttendanceQr(typ AttendanceClaimType, exp time.Time, id string) ([]byte, error) {
	jwt, err := GenerateAttendanceJWTWithID(typ, exp, id)
	if err != nil {
		return nil, err
	}

	qr, err := qrcode.New(fmt.Sprintf("%s#%s", openapp, jwt), qrcode.Medium)
	if err != nil {
		return nil, err
	}

	qrBytes, err := qr.PNG(512)
	if err != nil {
		return nil, err
	}

	return qrBytes, nil
}
