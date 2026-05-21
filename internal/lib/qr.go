package lib

import (
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"
)

const openapp = "https://hrconnect.hyourei.xyz/openapp/scan"

func GenerateAttendanceQr() ([]byte, error) {
	jwt, err := GenerateAttendanceJWT(CheckIn, time.Now().Add(time.Second * 30))
	if err != nil {
		return nil, err
	}

	qr, err := qrcode.New(fmt.Sprintf("%s#%s", openapp, jwt), qrcode.Medium)
	if err != nil {
		return nil, err
	}
	return qr.PNG(512)
}
