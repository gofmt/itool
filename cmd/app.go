package cmd

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/gofmt/itool/idevice"
	"github.com/gofmt/itool/idevice/installation"
	"github.com/gookit/gcli/v3"
)

var AppCmd = &gcli.Command{
	Name: "apps",
	Desc: "显示设备应用列表",
	Config: func(c *gcli.Command) {
		c.AddArg("name", "应用名称，用于过滤")
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := installation.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *installation.Client) {
			_ = cli.Close()
		}(cli)

		apps, err := cli.InstalledApps()
		if err != nil {
			return err
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 0, 1, ' ', 0)
		_, _ = fmt.Fprintln(w, "--------------------------------------------------------------")
		for i, app := range apps {
			if c.Arg("name").HasValue() && c.Arg("name").String() != app.CFBundleDisplayName {
				continue
			}

			_, _ = fmt.Fprintln(w, fmt.Sprintf("Number\t: %d", i))
			_, _ = fmt.Fprintln(w, fmt.Sprintf("Name\t: %s", app.CFBundleDisplayName))
			_, _ = fmt.Fprintln(w, fmt.Sprintf("BundleId\t: %s", app.CFBundleIdentifier))
			_, _ = fmt.Fprintln(w, fmt.Sprintf("Version\t: %s", app.CFBundleShortVersionString))
			_, _ = fmt.Fprintln(w, fmt.Sprintf("Executable\t: %s", app.CFBundleExecutable))
			_, _ = fmt.Fprintln(w, fmt.Sprintf("Container\t: %s", app.Container))
			_, _ = fmt.Fprintln(w, fmt.Sprintf("Path\t: %s", app.Path))
			_, _ = fmt.Fprintln(w, "--------------------------------------------------------------")
		}

		return w.Flush()
	},
}

var InstallCmd = &gcli.Command{
	Name:     "install",
	Desc:     "安装应用到设备",
	Examples: "{$binName} {$cmd} /path/example.ipa",
	Config: func(c *gcli.Command) {
		c.AddArg("ipa", "ipa 文件路径", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := installation.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *installation.Client) {
			_ = cli.Close()
		}(cli)

		ipa := c.Arg("ipa").String()
		f, err := os.Open(ipa)
		if err != nil {
			return err
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		return cli.CopyAndInstall(ipa, nil)
	},
}

var UninstallCmd = &gcli.Command{
	Name: "uninstall",
	Desc: "卸载设备应用",
	Config: func(c *gcli.Command) {
		c.AddArg("bundleID", "应用 BundleID", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		cli, err := installation.NewClient(device.SerialNumber)
		if err != nil {
			return err
		}
		defer func(cli *installation.Client) {
			_ = cli.Close()
		}(cli)

		return cli.Uninstall(c.Arg("bundleID").String(), nil)
	},
}

func GetAppDisplayName(device *idevice.DeviceAttachment, bundleID string) (string, error) {
	appCli, err := installation.NewClient(device.SerialNumber)
	if err != nil {
		return "", err
	}
	defer func(appCli *installation.Client) {
		_ = appCli.Close()
	}(appCli)

	return appCli.LookupDisplayName(bundleID)
}

func GetAppWithBundleID(device *idevice.DeviceAttachment, bundleID string) (*installation.AppInfo, error) {
	appCli, err := installation.NewClient(device.SerialNumber)
	if err != nil {
		return nil, err
	}
	defer func(appCli *installation.Client) {
		_ = appCli.Close()
	}(appCli)

	apps, err := appCli.InstalledApps()
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.CFBundleIdentifier == bundleID {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("app bundleID %s not found", bundleID)
}

func GetSelectApp(device *idevice.DeviceAttachment) (*installation.AppInfo, error) {
	appCli, err := installation.NewClient(device.SerialNumber)
	if err != nil {
		return nil, err
	}
	defer func(appCli *installation.Client) {
		_ = appCli.Close()
	}(appCli)

	apps, err := appCli.InstalledApps()
	if err != nil {
		return nil, err
	}

	fmt.Println("应用列表：")
	fmt.Println("--------------------------------------------------------------")
	for i, app := range apps {
		if app.CFBundleDisplayName != "" {
			fmt.Println(i, "\t|", app.CFBundleDisplayName, "["+app.CFBundleIdentifier+"]["+app.CFBundleExecutable+"]")
		}
	}

	fmt.Println("--------------------------------------------------------------")
	fmt.Println("输入应用编号：")
	var input string
	_, err = fmt.Scan(&input)
	if err != nil {
		return nil, err
	}

	idx, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("'%s' 不是正确的应用ID\n", input)
	}

	if idx > len(apps)-1 {
		return nil, fmt.Errorf("'%d' 应用ID不存在\n", idx)
	}

	return &apps[idx], nil
}
