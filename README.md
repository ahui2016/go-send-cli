# go-send-cli

go-send 的终端命令行工具，配合 go-send 使用。


## 安装方法

```sh
$ git clone https://github.com/ahui2016/go-send-cli.git
$ cd go-send-cli
$ go install
```


## 使用方法

- 执行命令 `go-send-cli` (在 Windows 里则是 go-send-cli.exe) 即可接收消息
- 执行命令 `go-send-cli -text "abc def"` 即可发送内容为 abc def 的消息


## 默认安装位置

- Linux 里通常是 /home/your-user-name/go/bin
- Windows 里通常是 C:\Users\your-user-name\go\bin
- 在安装前可以用命令 `go env -w GOBIN=path/to/go/bin` 指定安装位置
- 如果安装后运行 go-send-cli 找不到程序，请将安装位置添加到系统 path 中


## 与 go-send 配合

- 如果 go-send-cli 与 go-send 安装在同一台机器里，直接运行 go-send-cli 即可自动获取 go-send 的密码和网址。
- 如果 go-send-cli 与 go-send 安装在不同的机器中，则需要执行以下命令进行设置
  ```
  $ go-send-cli -pass 密码 -addr 网址
  ```


## demo 演示

由于我搭了一个 go-send 的演示站，因此你不需要自己搭建 go-send 即可体验 go-send-cli 的命令行操作。

- 安装 go-send-cli, 安装方法如上所示
- 执行以下命令
  ```
  $ go-send-cli -pass abc -addr http://send.ai42.xyz
  ```
- 然后就可以收发消息了，使用方法见上文。
