## iOS 越狱工具箱
* 集成 imobiledevice，无需额外安装，对设备进行文件、应用安装卸载等管理；
* 集成 frida，无需额外安装，执行脚本，应用砸壳；
* 集成 ssh 相关便捷操作，支持 dropbear 的 scp；
* 集成设备端口转发功能，无需额外安装 iproxy；
* 支持一键启动 debugserver 调试目标应用；
* 集成网络抓包功能，针对单个用于实现抓包，可TLS解密(实验性)；

# 命令列表
```shell
Itool (版本: v1.0.1-beta)
-----------------------------------------------------
apps         显示设备应用列表
debug        启动 debugserver 调试目标应用
device       选择默认设备
file         设备文件管理[/private/var/mobile/Media]
forward      转发设备端口到本机端口
frida        执行Frida脚本,应用砸壳
info         显示设备信息
install      安装应用到设备
pcap         对设备应用进行网络抓包, 抓包结束后执行 wireshark.sh
restart      重启设备(设备重启后需要重新越狱)
scp          通过SSH传递文件
scp2         通过SSH传递文件, 兼容 dropbear
screenshot   设备截屏
shell        创建SSH交互环境,需要设备越狱
shutdown     关闭设备(设备重启后需要重新越狱)
syslog       显示设备日志
uninstall    卸载设备应用
help         显示帮助信息
```

## 交流
QQ群: 280090
TG群: https://t.me/+vK3cWQ9p9B85OTg1

## 感谢
* https://github.com/4ch12dy/issh
