package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofmt/itool/idevice"
	"github.com/gofmt/itool/idevice/forward"
	"github.com/gookit/gcli/v3"
)

var ForwardCmd = &gcli.Command{
	Name: "forward",
	Desc: "转发设备端口到本机端口",
	Config: func(c *gcli.Command) {
		c.AddArg("lport", "本机端口", true)
		c.AddArg("rport", "设备端口", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT)
		defer cancel()

		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		lport := c.Arg("lport").Int()
		rport := c.Arg("rport").Int()

		if err := forward.Start(ctx, device.SerialNumber, lport, rport, func(s string, err error) {
			if err != nil {
				c.Errorln(err)
				os.Exit(-1)
			}
		}); err != nil {
			return err
		}

		<-ctx.Done()

		return nil
	},
}
