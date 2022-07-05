package main

import (
	"os"

	"github.com/gofmt/itool/cmd"
	"github.com/gookit/gcli/v3"
)

var (
	version = "1.0.1-beta"
)

func main() {
	gcli.DefaultVerb = gcli.VerbQuiet

	app := gcli.NewApp(func(app *gcli.App) {
		app.Version = "v" + version
		app.Desc = "itool"
		app.ExitOnEnd = false
		app.On(gcli.EvtAppInit, func(data ...interface{}) (stop bool) {
			return false
		})
	})

	app.On(gcli.EvtAppRunError, func(data ...interface{}) (stop bool) {
		app.Println(data[1])
		return true
	})

	app.On(gcli.EvtCmdNotFound, func(data ...interface{}) (stop bool) {
		app.Printf(`没有这个命令 "%s"`, data[1])
		app.Println("\n使用 itool --help 查看可用的命令")
		return true
	})

	app.On(gcli.EvtAppCmdNotFound, func(data ...interface{}) (stop bool) {
		return false
	})

	app.On(gcli.EvtCmdRunError, func(data ...interface{}) (stop bool) {
		return false
	})

	app.Add(
		cmd.DeviceCmd,
		cmd.InfoCmd,
		cmd.RestartCmd,
		cmd.ShutdownCmd,
		cmd.SyslogCmd,
		cmd.AppCmd,
		cmd.InstallCmd,
		cmd.UninstallCmd,
		cmd.ShellCmd,
		cmd.ForwardCmd,
		cmd.PcapCmd,
		cmd.DebugCmd,
		cmd.FileCmd,
		cmd.ScreenShotCmd,
		cmd.FridaCmd,
		cmd.ScpCmd,
		cmd.Scp2Cmd,
	)

	os.Exit(app.Run(nil))
}
