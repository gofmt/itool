package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gofmt/itool/idevice"
	"github.com/gofmt/itool/idevice/forward"
	"github.com/gookit/gcli/v3"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var ShellCmd = &gcli.Command{
	Name: "shell",
	Desc: "创建SSH交互环境,需要设备越狱",
	Subs: []*gcli.Command{
		RunCmd,
	},
	Config: func(c *gcli.Command) {
		c.AddArg("rport", "设备SSH端口")
	},
	Func: func(c *gcli.Command, args []string) error {
		device, err := idevice.GetDefaultDevice()
		if err != nil {
			return err
		}

		lport, err := GetAvailablePort()
		if err != nil {
			return err
		}

		rport := 22
		arg0 := c.Arg("rport")
		if arg0.HasValue() {
			rport = arg0.Int(22)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := forward.Start(ctx, device.SerialNumber, lport, rport, func(s string, err error) {
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}); err != nil {
			return err
		}

		sship := fmt.Sprintf("127.0.0.1:%d", lport)
		cli, err := newSSHClient(sship)
		if err != nil {
			return err
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

		fd := int(os.Stdin.Fd())
		oldState, err := terminal.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer func(fd int, oldState *terminal.State) {
			_ = terminal.Restore(fd, oldState)
		}(fd, oldState)

		session.Stdout = os.Stdout
		session.Stdin = os.Stdin
		session.Stderr = os.Stderr

		tWidth, tHeight, err := terminal.GetSize(fd)
		if err != nil {
			return err
		}

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		if err := session.RequestPty("xterm-256color", tHeight, tWidth, modes); err != nil {
			return err
		}

		if err := session.Shell(); err != nil {
			return err
		}

		_ = session.Wait()

		return nil
	},
}

var rport = 22

var RunCmd = &gcli.Command{
	Name: "run",
	Desc: "执行设备SHELL命令",
	Config: func(c *gcli.Command) {
		c.AddArg("commands", "SHELL命令列表", true, true)
		c.IntOpt(&rport, "rport", "r", 22, "设备SSH端口")
	},
	Func: func(c *gcli.Command, args []string) error {
		result, err := shellRun(rport, strings.Join(args, " "))
		if err != nil {
			return err
		}

		c.Println(string(result))

		return nil
	},
}

func shellRun(rport int, cmd string) (result []byte, err error) {
	device, err := idevice.GetDefaultDevice()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lport, err := GetAvailablePort()
	if err != nil {
		return nil, err
	}

	if err := forward.Start(ctx, device.SerialNumber, lport, rport, func(s string, err error) {
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}); err != nil {
		return nil, err
	}

	deviceIp := fmt.Sprintf("127.0.0.1:%d", lport)
	cli, err := newSSHClient(deviceIp)
	if err != nil {
		return
	}
	defer func(cli *ssh.Client) {
		_ = cli.Close()
	}(cli)

	session, err := cli.NewSession()
	if err != nil {
		return
	}
	defer func(session *ssh.Session) {
		_ = session.Close()
	}(session)

	buf := new(bytes.Buffer)
	session.Stdout = buf
	session.Stderr = os.Stderr
	if err := session.Run(cmd); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func iFileExsit(path string) (bool, error) {
	result, err := shellRun(rport, fmt.Sprintf(`[ -f %s ] && echo "1" || echo "0"`, path))
	if err != nil {
		return false, err
	}

	return string(result[0]) == "1", nil
}

func iDirExsit(path string) (bool, error) {
	result, err := shellRun(rport, fmt.Sprintf(`[ -d %s ] && echo "1" || echo "0"`, path))
	if err != nil {
		return false, err
	}

	return string(result[0]) == "1", nil
}

func newSSHClient(deviceIp string) (*ssh.Client, error) {
	cfg := ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password("alpine"),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}

	return ssh.Dial("tcp", deviceIp, &cfg)
}

func GetAvailablePort() (int, error) {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", "0.0.0.0"))
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return 0, err
	}
	defer func(listener *net.TCPListener) {
		_ = listener.Close()
	}(listener)

	return listener.Addr().(*net.TCPAddr).Port, nil
}
