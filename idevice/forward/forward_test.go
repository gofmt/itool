package forward

import (
	"context"
	"testing"

	"github.com/gofmt/itool/idevice"
)

func TestStart(t *testing.T) {
	device, err := idevice.GetDefaultDevice()
	if err != nil {
		t.Fatal(err)
	}

	if err := Start(context.Background(), device.SerialNumber, 2222, 2222, func(s string, err error) {
		if err != nil {
			t.Fatal(err)
		}

		t.Log(s)
	}); err != nil {
		t.Fatal(err)
	}

	select {}
}
