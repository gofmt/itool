package cmd

import "C"
import (
	"archive/zip"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/gofmt/itool/frida"
	"github.com/gofmt/itool/idevice"
	"github.com/gookit/gcli/v3"
)

var fridaOut = ""
var bundleID = ""
var FridaCmd = &gcli.Command{
	Name: "frida",
	Desc: "执行Frida脚本,应用砸壳",
	Subs: []*gcli.Command{
		dumpCmd,
	},
	Config: func(c *gcli.Command) {
		c.StrOpt(&fridaOut, "out", "o", "", "数据输出文件路径")
		c.StrOpt(&bundleID, "bundleID", "b", "", "应用ID")
		c.AddArg("path", "frida脚本路径", true)
	},
	Func: func(c *gcli.Command, args []string) error {
		source, err := ioutil.ReadFile(c.Arg("path").String())
		if err != nil {
			return err
		}

		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		if bundleID == "" {
			app, err := GetSelectApp(device)
			if err != nil {
				return err
			}
			bundleID = app.CFBundleIdentifier
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer cancel()

		return frida.Start(ctx, device.SerialNumber, bundleID, string(source), func(s string, bs []byte) {
			if len(strings.TrimSpace(fridaOut)) > 0 {
				_ = ioutil.WriteFile(fridaOut, bs, os.ModePerm)
			}
			fmt.Println(s)
		})
	},
}

//go:embed dump.js
var dumpScript string

var dumpCmd = &gcli.Command{
	Name: "dump",
	Desc: "使用Frida砸壳",
	Config: func(c *gcli.Command) {
		c.StrOpt(&bundleID, "bundleID", "b", "", "应用ID")
		c.AddArg("path", "ipa 输出路径")
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		displayName := ""
		if bundleID == "" {
			app, err := GetSelectApp(device)
			if err != nil {
				return err
			}

			bundleID = app.CFBundleIdentifier
			displayName = app.CFBundleDisplayName
		} else {
			displayName, err = GetAppDisplayName(device, bundleID)
			if err != nil {
				return err
			}
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer cancel()

		c.Printf("Dumping %s ...\n", displayName)

		tempPath := os.TempDir()
		defer func(path string) {
			_ = os.RemoveAll(path)
		}(tempPath)

		payloadPath := filepath.Join(tempPath, "Payload")
		_ = os.MkdirAll(payloadPath, os.ModePerm)
		return frida.Start(ctx, device.SerialNumber, bundleID, dumpScript, func(s string, bs []byte) {
			logMap := make(map[string]interface{})
			if err := json.Unmarshal([]byte(s), &logMap); err != nil {
				c.Errorln(err)
				cancel()
				return
			}

			payload := logMap["payload"]
			atype := logMap["type"]
			if atype == "log" {
				c.Println(payload.(string))
			} else if atype == "send" {
				obj, ok := payload.(map[string]interface{})
				if !ok {
					cancel()
					return
				}

				appPath, ok := obj["app"].(string)
				if ok {
					baseName := filepath.Base(appPath)
					if err := pull(device, baseName, payloadPath); err != nil {
						c.Errorln(err)
						cancel()
						return
					}
				}

				dumpPath, ok := obj["dump"].(string)
				if ok {
					binPath := obj["path"].(string)
					idx := strings.LastIndex(binPath, ".app/")
					baseName := filepath.Base(binPath[:idx+4])
					realPath := filepath.Join(payloadPath, baseName, binPath[idx+4:])
					if err := pull(device, filepath.Base(dumpPath), realPath); err != nil {
						c.Errorln(err)
						cancel()
						return
					}
					_ = os.Chmod(realPath, 0655)
				}

				done, ok := obj["done"].(string)
				if ok && done == "ok" {
					ipaPath := displayName + ".ipa"
					curPath, _ := os.Getwd()
					outFullPath := filepath.Join(curPath, ipaPath)
					if c.Arg("path").HasValue() {
						ipaPath = c.Arg("path").String()
						outFullPath = ipaPath
					}
					if err := compress(payloadPath, ipaPath); err != nil {
						c.Errorln(err)
					}
					c.Printf("dump %s to path: %s\n", displayName, outFullPath)
					cancel()
				}
			} else if atype == "error" {
				c.Errorln(s)
				cancel()
			}
		})
	},
}

func compress(srcPath, zipPath string) error {
	_ = os.RemoveAll(zipPath)
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	archive := zip.NewWriter(f)
	defer func(archive *zip.Writer) {
		_ = archive.Close()
	}(archive)

	if err := filepath.Walk(srcPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = "Payload/" + strings.TrimPrefix(path, srcPath)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		wr, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			sf, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func(sf *os.File) {
				_ = sf.Close()
			}(sf)

			_, _ = io.Copy(wr, sf)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
