package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gofmt/itool/idevice"
	"github.com/gofmt/itool/idevice/forward"
	"github.com/gookit/gcli/v3"
)

var DebugCmd = &gcli.Command{
	Name: "debug",
	Desc: "启动 debugserver 调试目标应用",
	Config: func(c *gcli.Command) {
		c.IntOpt(&rport, "rport", "r", 22, "设备SSH端口")
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		app, err := GetSelectApp(device)
		if err != nil {
			return err
		}

		execPath := filepath.Join(app.Path, app.CFBundleExecutable)

		itoolPath := "/itool"
		ok, err := iDirExsit(itoolPath)
		if err != nil {
			return err
		}
		if !ok {
			_, err = shellRun(rport, fmt.Sprintf(`mkdir -p %s`, itoolPath))
			if err != nil {
				return err
			}
		}

		ok, err = iFileExsit("/Developer/usr/bin/debugserver")
		if err != nil {
			return err
		}
		if !ok {
			c.Errorln("/Developer/usr/bin/debugserver not exist. please connect idevice to Xcode")
			c.Errorln("also you can get all iOS DeviceSupport file at https://github.com/iGhibli/iOS-DeviceSupport")
			return nil
		}

		_, _ = shellRun(rport, `killall -9 debugserver 2> /dev/null`)

		debugserverPath := itoolPath + "/debugserver"
		ok, _ = iFileExsit(debugserverPath)
		if !ok {
			_, err = shellRun(rport, fmt.Sprintf(`cat > /%s/ent.xml << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.backboardd.debugapplications</key>
    <true/>
    <key>com.apple.backboardd.launchapplications</key>
    <true/>
    <key>com.apple.diagnosticd.diagnostic</key>
    <true/>
    <key>com.apple.frontboard.debugapplications</key>
    <true/>
    <key>com.apple.frontboard.launchapplications</key>
    <true/>
    <key>com.apple.security.network.client</key>
    <true/>
    <key>com.apple.security.network.server</key>
    <true/>
    <key>com.apple.springboard.debugapplications</key>
    <true/>
    <key>com.apple.system-task-ports</key>
    <true/>
    <key>com.apple.private.logging.diagnostic</key>
    <true/>
    <key>com.apple.private.memorystatus</key>
    <true/>
    <key>com.apple.private.cs.debugger</key>
    <true/>
    <key>com.apple.private.security.container-required</key>
    <false/>
    <key>get-task-allow</key>
    <true/>
    <key>platform-application</key>
    <true/>
    <key>run-unsigned-code</key>
    <true/>
    <key>task_for_pid-allow</key>
    <true/>
</dict>
</plist> 
EOF`, itoolPath))
			if err != nil {
				return err
			}

			_, err = shellRun(rport, fmt.Sprintf(`cp /Developer/usr/bin/debugserver %[1]s;\
            cd %s;ldid -Sent.xml %[1]s;chmod +x  %[1]s;`, debugserverPath, itoolPath))
			if err != nil {
				return err
			}
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer cancel()

		go func() {
			defer cancel()

			c.Println(fmt.Sprintf("%s debugging...", app.CFBundleDisplayName))
			dbgcmd := fmt.Sprintf(`%s 127.0.0.1:1234 %s -x backboard`, debugserverPath, execPath)
			c.Println(fmt.Sprintf("debug params: %s", dbgcmd))
			_, err = shellRun(rport, dbgcmd)
			if err != nil {
				c.Errorln(err)
				return
			}

			c.Println("shell run exited.")
		}()

		go func() {
			time.Sleep(time.Second)
			if err := forward.Start(ctx, device.SerialNumber, 1234, 1234, func(s string, err error) {
				if err != nil {
					c.Errorln(err)
					os.Exit(-1)
				}
			}); err != nil {
				c.Errorln(err)
				os.Exit(-1)
			}
		}()

		<-ctx.Done()

		c.Println(fmt.Sprintf("%s debugging has exited.", app.CFBundleDisplayName))

		return nil
	},
}
