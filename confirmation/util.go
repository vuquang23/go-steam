package confirmation

import (
	"crypto/md5"
	"fmt"
)

func getDeviceID(accountName string, password string) string {
	sum := md5.Sum([]byte(accountName + password))
	deviceID := fmt.Sprintf(
		"android:%x-%x-%x-%x-%x",
		sum[:2], sum[2:4], sum[4:6], sum[6:8], sum[8:10],
	)

	return deviceID
}
