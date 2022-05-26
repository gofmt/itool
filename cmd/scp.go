package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofmt/itool/idevice"
	"github.com/gofmt/itool/idevice/forward"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/progress"
	"golang.org/x/crypto/ssh"
)

var ScpCmd = &gcli.Command{
	Name: "scp",
	Desc: "通过SSH传递文件",
	Config: func(c *gcli.Command) {
		c.IntOpt(&rport, "rport", "r", 22, "设备SSH端口")
		c.AddArg("src", "源路径", true)
		c.AddArg("dst", "目标路径", true)
	},
	Examples: "local file to device: {$binName} {$cmd} /path/example.js :/tmp/example." +
		"js\ndevice file to local: {$binName} {$cmd} :/tmp/example.js /path/example.js",
	Func: func(c *gcli.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("SCP参数错误")
		}

		var (
			remotePath   = ""
			localPath    = ""
			copyToRemote = false
		)

		if strings.HasPrefix(args[0], ":") {
			remotePath = strings.TrimLeft(args[0], ":")
			localPath = args[1]
			copyToRemote = false
		} else if strings.HasPrefix(args[1], ":") {
			remotePath = strings.TrimLeft(args[1], ":")
			localPath = args[0]
			copyToRemote = true
		}

		if len(remotePath) == 0 || len(localPath) == 0 {
			return errors.New("SCP参数错误")
		}

		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		lPort, err := GetAvailablePort()
		if err != nil {
			return err
		}

		if err := forward.Start(context.Background(), device.SerialNumber, lPort, rport, nil); err != nil {
			return err
		}

		deviceIp := fmt.Sprintf("127.0.0.1:%d", lPort)
		cli, err := newSSHClient(deviceIp)
		if err != nil {
			return fmt.Errorf("连接SSH错误：%w", err)
		}
		defer func(cli *ssh.Client) {
			_ = cli.Close()
		}(cli)

		session, err := cli.NewSession()
		if err != nil {
			return err
		}
		defer func(session *ssh.Session) {
			_ = session.Close()
		}(session)

		if copyToRemote {
			return scpTo(session, localPath, remotePath)
		}

		return scpFrom(session, remotePath, localPath)
	},
}

func scpFrom(session *ssh.Session, remote, local string) error {
	done := make(chan bool)
	cherr := make(chan error)
	go func() {
		err := session.Run("/usr/bin/scp -qrf " + remote)
		if err != nil {
			cherr <- err
			return
		}
		done <- true
	}()

	go func() {
		writer, err := session.StdinPipe()
		if err != nil {
			cherr <- err
			return
		}
		defer func(writer io.WriteCloser) {
			_ = writer.Close()
		}(writer)

		reader, err := session.StdoutPipe()
		if err != nil {
			cherr <- err
			return
		}

		_, err = writer.Write([]byte{0})
		if err != nil {
			cherr <- err
			return
		}

		var (
			permMode int
			fileSize int
			fileName string
		)
		_, err = fmt.Fscanf(reader, "C%04o %d %s", &permMode, &fileSize, &fileName)
		if err != nil {
			cherr <- err
			return
		}

		_, err = writer.Write([]byte{0})
		if err != nil {
			cherr <- err
			return
		}

		f, err := os.OpenFile(local, os.O_CREATE|os.O_RDWR, os.FileMode(permMode))
		if err != nil {
			cherr <- err
			return
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		cs := progress.BarStyles[3]
		p := progress.CustomBar(40, cs)
		p.Format = progress.FullBarFormat

		p.Start()

		chunkSize := int64(4096)
		totalSize := int64(fileSize)
		p.MaxSteps = uint(totalSize / chunkSize)
		curSize := int64(0)
		for curSize < totalSize {
			if chunkSize > totalSize-curSize {
				chunkSize = totalSize - curSize
			}
			bs := make([]byte, chunkSize)
			n, err := reader.Read(bs)
			if err != nil {
				cherr <- err
				return
			}
			curSize += int64(n)
			_, err = f.Write(bs[:n])
			if err != nil {
				cherr <- err
				return
			}
			p.Advance()
		}
		p.Finish()

		err = f.Close()
		if err != nil {
			cherr <- err
			return
		}

		_, err = writer.Write([]byte{0})
		if err != nil {
			cherr <- err
			return
		}

	}()

	select {
	case err := <-cherr:
		if err != nil {
			return err
		}
	case <-done:
		return nil
	}

	return nil
}

func scpTo(session *ssh.Session, local, remote string) error {
	done := make(chan bool)
	cherr := make(chan error)
	go func() {
		err := session.Run("/usr/bin/scp -qrt " + remote)
		if err != nil {
			cherr <- err
			return
		}
	}()

	go func() {
		f, err := os.Open(local)
		if err != nil {
			cherr <- err
			return
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		fi, _ := f.Stat()

		w, err := session.StdinPipe()
		if err != nil {
			cherr <- err
			return
		}
		defer func(w io.WriteCloser) {
			_ = w.Close()
		}(w)

		_, err = fmt.Fprintf(w, "C%04o %d %s\n", fi.Mode(), fi.Size(), filepath.Base(remote))
		if err != nil {
			cherr <- err
			return
		}

		cs := progress.BarStyles[3]
		p := progress.CustomBar(40, cs)
		p.Format = progress.FullBarFormat
		p.MaxSteps = uint(fi.Size() / 4096)
		p.Start()

		buf := make([]byte, 4096)
		for {
			switch nr, err := f.Read(buf[:]); true {
			case nr < 0:
				cherr <- fmt.Errorf("cat: error reading: %w", err)
				return
			case nr == 0:
				_, err = w.Write([]byte{0})
				if err != nil {
					cherr <- err
					return
				}

				done <- true
				p.Finish()
				return
			case nr > 0:
				if _, err := w.Write(buf); err != nil {
					cherr <- fmt.Errorf("file write error: %w", err)
					return
				}
				p.Advance()
			}
		}
	}()

	select {
	case err := <-cherr:
		if err != nil {
			return err
		}
	case <-done:
		return nil
	}

	return nil
}
