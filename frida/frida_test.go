package frida

import (
	"context"
	"os"
	"os/signal"
	"testing"

	"github.com/gofmt/itool/idevice"
)

func TestStart(t *testing.T) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	device, err := idevice.GetDefaultDevice()
	if err != nil {
		t.Fatal(err)
	}

	if err := Start(ctx, device.SerialNumber, "com.elong.app", "console.log('hello world!')", func(s string, bs []byte) {
		t.Log(s)
		cancel()
	}); err != nil {
		t.Fatal(err)
	}
}
