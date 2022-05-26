package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/gofmt/itool/idevice"
	"github.com/gofmt/itool/idevice/diagnostics"
	"github.com/gofmt/itool/idevice/lockdownd"
	"github.com/gofmt/itool/idevice/screenshotr"
	"github.com/gofmt/itool/idevice/syslog"
	"github.com/gookit/color"
	"github.com/gookit/gcli/v3"
)

var DeviceCmd = &gcli.Command{
	Name: "device",
	Desc: "选择默认设备",
	Func: func(c *gcli.Command, args []string) error {
		conn, err := idevice.NewConn()
		if err != nil {
			return err
		}
		defer func(conn *idevice.Conn) {
			_ = conn.Close()
		}(conn)

		devices, err := conn.ListDevices()
		if err != nil {
			return err
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 0, 1, ' ', 0)
		_, _ = fmt.Fprintln(w, "--------------------------------------------------------------")

		for i, device := range devices {
			cli, err := lockdownd.NewClient(device.SerialNumber)
			if err != nil {
				return err
			}

			info, err := cli.GetValues()
			if err != nil {
				_ = cli.Close()
				return err
			}

			_ = cli.Close()

			_, _ = fmt.Fprintln(w, "- Number\t: "+strconv.Itoa(i))
			_, _ = fmt.Fprintln(w, "- UDID\t: "+info.UniqueDeviceID)
			_, _ = fmt.Fprintln(w, "- DeviceName\t: "+info.DeviceName)
			_, _ = fmt.Fprintln(w, "- ProductName\t: "+info.ProductName)
			_, _ = fmt.Fprintln(w, "- ProductType\t: "+info.ProductType)
			_, _ = fmt.Fprintln(w, "- ProductVersion\t: "+info.ProductVersion)
			_, _ = fmt.Fprintln(w, "- CPUArchitecture\t: "+info.CPUArchitecture)
			_, _ = fmt.Fprintln(w, "- BuildVersion\t: "+info.BuildVersion)
			_, _ = fmt.Fprintln(w, "- WiFiAddress\t: "+info.WiFiAddress)
			_, _ = fmt.Fprintln(w, "- DeviceColor\t: "+info.DeviceColor)
			_, _ = fmt.Fprintln(w, "- HardwareModel\t: "+info.HardwareModel)
			_, _ = fmt.Fprintln(w, "- UniqueChipID\t: "+fmt.Sprintf("%d", info.UniqueChipID))
			_, _ = fmt.Fprintln(w, "--------------------------------------------------------------")

		}

		_ = w.Flush()

		fmt.Println("输入设备编号并设置为默认设备：")
		var input string
		_, err = fmt.Scan(&input)
		if err != nil {
			return err
		}

		idx, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("'%s' 不是正确的设备编号\n", input)
		}

		if idx > len(devices)-1 {
			return fmt.Errorf("'%d' 设备编号不存在\n", idx)
		}

		homeDir, _ := os.UserHomeDir()
		device := devices[idx]
		return ioutil.WriteFile(filepath.Join(homeDir, ".itool"), []byte(device.SerialNumber), os.ModePerm)
	},
}

var InfoCmd = &gcli.Command{
	Name: "info",
	Desc: "显示设备信息",
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := lockdownd.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *lockdownd.Client) {
			_ = cli.Close()
		}(cli)

		info, err := cli.GetValues()
		if err != nil {
			return err
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 0, 1, ' ', 0)

		_, _ = fmt.Fprintln(w, "- UDID\t: "+info.UniqueDeviceID)
		_, _ = fmt.Fprintln(w, "- DeviceName\t: "+info.DeviceName)
		_, _ = fmt.Fprintln(w, "- ProductName\t: "+info.ProductName)
		_, _ = fmt.Fprintln(w, "- ProductType\t: "+info.ProductType)
		_, _ = fmt.Fprintln(w, "- ProductVersion\t: "+info.ProductVersion)
		_, _ = fmt.Fprintln(w, "- CPUArchitecture\t: "+info.CPUArchitecture)
		_, _ = fmt.Fprintln(w, "- BuildVersion\t: "+info.BuildVersion)
		_, _ = fmt.Fprintln(w, "- WiFiAddress\t: "+info.WiFiAddress)
		_, _ = fmt.Fprintln(w, "- DeviceColor\t: "+info.DeviceColor)
		_, _ = fmt.Fprintln(w, "- HardwareModel\t: "+info.HardwareModel)
		_, _ = fmt.Fprintln(w, "- UniqueChipID\t: "+fmt.Sprintf("%d", info.UniqueChipID))

		_ = w.Flush()

		return nil
	},
}

var ScreenShotCmd = &gcli.Command{
	Name: "screenshot",
	Desc: "设备截屏",
	Config: func(c *gcli.Command) {
		c.AddArg("path", "截屏图片保存路径", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := screenshotr.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *screenshotr.Client) {
			_ = cli.Close()
		}(cli)

		img, err := cli.ScreenshotImage()
		if err != nil {
			return err
		}

		f, err := os.Create(c.Arg("path").String())
		if err != nil {
			return err
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		return png.Encode(f, img)
	},
}

var RestartCmd = &gcli.Command{
	Name: "restart",
	Desc: "重启设备(设备重启后需要重新越狱)",
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := diagnostics.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *diagnostics.Client) {
			_ = cli.Close()
		}(cli)

		return cli.Restart()
	},
}

var ShutdownCmd = &gcli.Command{
	Name: "shutdown",
	Desc: "关闭设备(设备重启后需要重新越狱)",
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := diagnostics.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *diagnostics.Client) {
			_ = cli.Close()
		}(cli)

		return cli.Shutdown()
	},
}

var SyslogCmd = &gcli.Command{
	Name: "syslog",
	Desc: "显示设备日志",
	Config: func(c *gcli.Command) {
		c.AddArg("key", "过滤关键字")
	},
	Func: func(c *gcli.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT)
		defer cancel()

		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		r, err := syslog.Syslog(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(r io.ReadCloser) {
			_ = r.Close()
		}(r)

		fgWhite := color.FgWhite.Render
		fgGreen := color.FgGreen.Render
		fgCyan := color.FgCyan.Render
		fgRed := color.FgRed.Render
		fgLiRed := color.FgLightRed.Render
		fgYellow := color.FgYellow.Render
		fgLiYellow := color.FgLightYellow.Render
		fgMagenta := color.FgMagenta.Render
		white := color.White.Render

		br := bufio.NewReader(r)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:

				}

				bs, err := br.ReadBytes(0)
				if err != nil {
					fmt.Println(err)
					continue
				}

				bs = bytes.TrimRight(bs, "\x0a\x00")
				line := decodeSyslog(bs)
				ss := strings.Split(line, ">: ")
				header := strings.Split(ss[0], " ")
				if c.Arg("key").HasValue() && !strings.Contains(line, c.Arg("key").String()) {
					continue
				}

				t, err := time.Parse(time.Stamp, header[0]+" "+header[1]+" "+header[2])
				if err != nil {
					panic(err)
				}

				level := header[5][1:]
				body := ss[1]
				switch level {
				case "Notice":
					level = fgGreen(level)
				case "Error":
					level = fgRed(level)
					body = fgLiRed(body)
				case "Warning":
					level = fgYellow(level)
					body = fgLiYellow(body)
				case "Debug":
					level = fgMagenta(level)
				default:
					level = white(level)
				}

				fmt.Printf(
					"[%s](%s)[%s]: %s\n",
					fgWhite(t.Format("01-02 15:04:05")),
					// gray(msg.DeviceName),
					fgCyan(header[4]),
					level,
					body,
				)
			}
		}()

		<-ctx.Done()

		return nil
	},
}

func decodeSyslog(bs []byte) string {
	specialChar := bytes.Contains(bs, []byte(`\134`))
	if specialChar {
		bs = bytes.Replace(bs, []byte(`\134`), []byte(""), -1)
	}
	kBackslash := byte(0x5c)
	kM := byte(0x4d)
	kDash := byte(0x2d)
	kCaret := byte(0x5e)

	// Mask for the UTF-8 digit range.
	kNum := byte(0x30)

	var out []byte
	for i := 0; i < len(bs); {

		if (bs[i] != kBackslash) || i > (len(bs)-4) {
			out = append(out, bs[i])
			i = i + 1
		} else {
			if (bs[i+1] == kM) && (bs[i+2] == kCaret) {
				out = append(out, (bs[i+3]&byte(0x7f))+byte(0x40))
			} else if bs[i+1] == kM && bs[i+2] == kDash {
				out = append(out, bs[i+3]|byte(0x80))
			} else if isDigit(bs[i+1:i+3], kNum) {
				out = append(out, decodeOctal(bs[i+1], bs[i+2], bs[i+3]))
			} else {
				out = append(out, bs[0], bs[1], bs[2], bs[3], bs[4])
			}
			i = i + 4
		}
	}
	return string(out)
}

func isDigit(b []byte, kNum byte) bool {
	for _, v := range b {
		if (v & byte(0xf0)) != kNum {
			return false
		}
	}
	return true
}

func decodeOctal(x, y, z byte) byte {
	return (x&byte(0x3))<<byte(6) | (y&byte(0x7))<<byte(3) | z&byte(0x7)
}
