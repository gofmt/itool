package cmd

import (
	"fmt"
	"os"

	"github.com/gofmt/itool/idevice"
	"github.com/gofmt/itool/idevice/afc"
	"github.com/gookit/gcli/v3"
)

var FileCmd = &gcli.Command{
	Name: "file",
	Desc: "设备文件管理[/private/var/mobile/Media]",
	Subs: []*gcli.Command{
		MkdirCmd,
		RmdirCmd,
		RemoveCmd,
		PushCmd,
		PullCmd,
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := afc.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *afc.Client) {
			_ = cli.Close()
		}(cli)

		fs, err := cli.ReadDir("/")
		if err != nil {
			return err
		}

		for _, f := range fs {
			if f == "." || f == ".." {
				continue
			}
			fmt.Println(f)
		}

		return nil
	},
}

var MkdirCmd = &gcli.Command{
	Name: "mkdir",
	Desc: "创建目录[/private/var/mobile/Media]",
	Config: func(c *gcli.Command) {
		c.AddArg("name", "目录名称", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := afc.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *afc.Client) {
			_ = cli.Close()
		}(cli)

		return cli.MakeDir(c.Arg("name").String())
	},
}

var RmdirCmd = &gcli.Command{
	Name: "rmdir",
	Desc: "删除目录[/private/var/mobile/Media]",
	Config: func(c *gcli.Command) {
		c.AddArg("name", "目录名称", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := afc.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *afc.Client) {
			_ = cli.Close()
		}(cli)

		return cli.RemoveAll(c.Arg("name").String())
	},
}

var RemoveCmd = &gcli.Command{
	Name: "remove",
	Desc: "删除文件[/private/var/mobile/Media]",
	Config: func(c *gcli.Command) {
		c.AddArg("path", "文件路径", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := afc.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *afc.Client) {
			_ = cli.Close()
		}(cli)

		return cli.RemovePath(c.Arg("path").String())
	},
}

var PushCmd = &gcli.Command{
	Name: "push",
	Desc: "推送文件或目录到设备[/private/var/mobile/Media]",
	Config: func(c *gcli.Command) {
		c.AddArg("src", "本机路径", true)
		c.AddArg("dst", "设备路径", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := afc.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *afc.Client) {
			_ = cli.Close()
		}(cli)

		return cli.CopyToDevice(c.Arg("dst").String(), c.Arg("src").String(), func(dst, src string, info os.FileInfo) {
			fmt.Println(src, "->", dst)
		})
	},
}

var PullCmd = &gcli.Command{
	Name: "pull",
	Desc: "从设备拉取文件或目录到本机[/private/var/mobile/Media]",
	Config: func(c *gcli.Command) {
		c.AddArg("src", "设备路径", true)
		c.AddArg("dst", "本机路径", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		return pull(device, c.Arg("dst").String(), c.Arg("src").String())
	},
}

func pull(device *idevice.DeviceAttachment, src, dst string) error {
	cli, err := afc.NewClient(device.SerialNumber)
	if err != nil {
		return err
	}
	defer func(cli *afc.Client) {
		_ = cli.Close()
	}(cli)

	return cli.CopyFromDevice(dst, src, func(dst, src string,
		info os.FileInfo) {
		fmt.Println(dst, "<-", "/private/var/mobile/Media/"+src)
	})
}

func push(device *idevice.DeviceAttachment, src, dst string) error {
	cli, err := afc.NewClient(device.SerialNumber)
	if err != nil {
		return err
	}
	defer func(cli *afc.Client) {
		_ = cli.Close()
	}(cli)

	return cli.CopyToDevice(dst, src, func(dst, src string, info os.FileInfo) {
		fmt.Println(src, "->", "/private/var/mobile/Media/"+dst)
	})
}
