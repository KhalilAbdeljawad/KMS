package ssh

import (
	"KMSV2/helpers"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

const port = ":22"

func Run() {
	client, err := DialWithPasswd("ip", "user", "password")
	if err != nil {
		handleErr(err)
	}
	defer client.Close()

	helpers.Dump(client)
	//return
	out, err := client.Cmd("cat /tmp/system.cfg").Output()
	if err != nil {
		handleErr(err)
	}
	fmt.Println(string(out))

	// default terminal
	if err := client.Terminal(nil).Start(); err != nil {
		handleErr(err)
	}

	// with a terminal config

	config := TerminalConfig{
		Term:   "xterm",
		Height: 40,
		Weight: 80,
		Modes: ssh.TerminalModes{
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		},
	}
	if err := client.Terminal(&config).Start(); err != nil {
		handleErr(err)
	}
}

func handleErr(err error) bool {
	if err != nil {
		fmt.Printf("\nError: %v", err)
		return false
	} else {
		return true
	}
}

func ConnectAndRun(ip, username, password, command string) string {
	client, err := DialWithPasswd(ip+port, username, password)
	if err != nil {
		handleErr(err)
		return ""
	}
	defer client.Close()

	out, err := client.Cmd(command).Output()
	if err != nil {
		handleErr(err)
	}
	return (string(out))
}

func GetUbntConfig(ip, username, password string) string {
	return ConnectAndRun(ip, username, password, "cat /tmp/system.cfg")
}

func (client *Client) GetUbntConfig() string {
	str, _ := client.Cmd("cat /tmp/system.cfg").Output()
	return string(str)
}

func DownloadFile(sshFilePath, localFileName string) {

}

func (client *Client) UploadFile(srcFile, destFile, destPath string) bool {
	session, err := client.client.NewSession()
	defer session.Close()
	println(client.client.Conn.User())
	ret := handleErr(err)
	file, err := os.Open(srcFile)
	defer file.Close()
	stat, _ := file.Stat()
	ret = handleErr(err)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		hostIn, err := session.StdinPipe()
		ret = handleErr(err)
		if !ret {
			//return ret
		}
		defer hostIn.Close()
		println(destFile)
		println(stat.Size())
		fmt.Fprintf(hostIn, "C0664 %d %s\n", stat.Size(), destFile)
		n, err := io.Copy(hostIn, file)
		ret = handleErr(err)
		if !ret {
			//return ret
		}
		println(n)
		l, err := fmt.Fprint(hostIn, "\x00")
		ret = handleErr(err)
		if !ret {
			//return ret
		}
		println(l)
		wg.Done()
	}()

	err = session.Run("/usr/bin/scp -t " + destPath)
	client.Cmd("cfgmtd -f /tmp/system.cfg -w ").Run()

	client.Cmd("reboot").Run()
	return handleErr(err)
	wg.Wait()
	return true
}

func (client *Client) UploadText(text, destFile, destPath string) bool {
	srcFile := helpers.RandStringRunes(10) + ".txt"

	helpers.WriteStrToFile(srcFile, text)
	println(srcFile)
	var b bool = true
	println("start uploading..")
	client.UploadFile(srcFile, destFile, destPath)

	return b
}

func (client *Client) ConfigByLine(line, destFile, destPath string) bool {
	configText := client.GetUbntConfig()
	configArray := strings.Split(configText, "\n")
	configObject := strings.Split(line, "=")[0]
	newConfig := ""
	for k := range configArray {
		if strings.Contains(configArray[k], configObject) {
			configArray[k] = line
		}
		newConfig += configArray[k] + "\n"
	}

	srcFile := helpers.RandStringRunes(10) + ".txt"

	helpers.WriteStrToFile(srcFile, newConfig)
	println(srcFile)

	var b bool = true
	println("start uploading..")

	b = client.UploadFile(srcFile, destFile, destPath)

	os.Remove(srcFile)
	return b
}

func UploadFileSCP(srcFile, destFile string) bool {
	print(srcFile, " ", destFile)
	clientConfig, _ := auth.PasswordKey("user", "password", ssh.InsecureIgnoreHostKey())

	// For other authentication methods see ssh.ClientConfig and ssh.AuthMethod

	// Create a new SCP client
	client := scp.NewClient("10.100.37.208:22", &clientConfig)

	// Connect to the remote server
	err := client.Connect()
	if err != nil {
		fmt.Println("Couldn't establish a connection to the remote server ", err)
		return false
	}

	// Open a file
	f, err := os.Open(srcFile)
	handleErr(err)
	// Close client connection after the file has been copied
	defer client.Close()

	// Close the file after it has been copied
	defer f.Close()

	// Finaly, copy the file over
	// Usage: CopyFile(fileReader, remotePath, permission)

	err = client.CopyFile(f, destFile, "0655")
	handleErr(err)
	if err != nil {
		fmt.Println("Error while copying file ", err)
		return false
	}
	return true
}

func GetCambConfig(ip, username, password string) string {
	return ConnectAndRun(ip, username, password, "config show dump")
}
