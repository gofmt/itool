package idevice

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetDefaultDevice() (*DeviceAttachment, error) {
	conn, err := NewConn()
	if err != nil {
		return nil, err
	}
	defer func(conn *Conn) {
		_ = conn.Close()
	}(conn)

	devices, err := conn.ListDevices()
	if err != nil {
		return nil, err
	}

	if len(devices) < 1 {
		return nil, errors.New("not found device")
	}

	homeDir, _ := os.UserHomeDir()
	bs, err := ioutil.ReadFile(filepath.Join(homeDir, ".itool"))
	if err != nil {
		return devices[0], nil
	}

	for _, device := range devices {
		if device.SerialNumber == string(bs) {
			return device, nil
		}
	}

	return devices[0], nil
}
