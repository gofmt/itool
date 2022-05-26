package syslog

import (
	"io"
	"os"
	"testing"

	"github.com/gofmt/itool/idevice"
)

func TestSyslog(t *testing.T) {
	conn, err := idevice.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer func(conn *idevice.Conn) {
		_ = conn.Close()
	}(conn)

	devices, err := conn.ListDevices()
	if err != nil {
		t.Fatal(err)
	}

	device := devices[0]
	r, err := Syslog(device.SerialNumber)
	if err != nil {
		t.Fatal(err)
	}

	io.Copy(os.Stdout, r)
}
