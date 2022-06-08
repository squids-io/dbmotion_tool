package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"io/ioutil"
)

var ip = flag.String("h", "", "ip will be connected")
var port = flag.String("p", "", "port will be connected")
var action = flag.String("t", "", "check/set/create/close")

func GetListenPort() map[string]int {
	ports := make(map[string]int)
	cmd := exec.Command("ssh", "-p", "22222", "-i", "squids-ali-dbmotion.pem", "dbmotion@dbmotion.squids.cn",
		"netstat", "-apn", "|", "grep", "LISTEN", "|", "grep", "^tcp", "|", "awk", "'{print $4}'")
	out, _ := cmd.CombinedOutput()
	rs := string(out)
	r := strings.Split(rs, "\n")
	for i := 0; i < len(r); i++ {
		if !strings.Contains(r[i], ":") {
			continue
		}
		if string(r[i][0]) == ":" { // ipv6
			if _, ok := ports[r[i][2:]]; ok {
				continue
			} else {
				ports[r[i][2:]] = 1
			}
		} else {
			f := strings.Split(r[i], ":")
			if _, ok := ports[f[1]]; ok {
				continue
			} else {
				ports[f[1]] = 1
			}
		}

	}
	return ports
}

func SetSsh() {
	fmt.Println("Check and set sshd.")
	cmd := exec.Command("bash", "-c", "cat /etc/ssh/sshd_config | grep -n GatewayPorts | awk -F : '{print $1}'")
	out, _ := cmd.CombinedOutput()
	line := string(out)

	if line != "" {
		fmt.Println("Found GatewayPosts, delete it")
		cmd = exec.Command("sed", "-i", "/GatewayPorts/d", "/etc/ssh/sshd_config")
		err := cmd.Run()
		if err != nil {
			fmt.Println("Set sshd failed: " + err.Error())
			fmt.Println("You can set 'GatewayPorts yes' and restart sshd by manual")
			os.Exit(1)
		}
	} else {
		fmt.Println("GatewayPorts not found, add it")
	}
	cmd = exec.Command("sed", "-i", `$a\GatewayPorts yes`, "/etc/ssh/sshd_config")
	/*
	   } else {
		fmt.Println("Found GatewayPorts, set to yes")
		c := fmt.Sprintf(`'%sc GatewayPorts yes'`, line)
		cmd = exec.Command("sed", "-i", c, "/etc/ssh/sshd_config")
	}*/
	err := cmd.Run()
	if err != nil {
		fmt.Println("Set sshd failed: " + err.Error())
		fmt.Println("You can set 'GatewayPorts yes' and restart sshd by manual")
		os.Exit(1)
	}
	fmt.Println("Restart sshd")
	cmd = exec.Command("systemctl", "restart", "sshd")
	err = cmd.Run()
	if err != nil {
		fmt.Println("Restart sshd failed: "+ err.Error())
		os.Exit(1)
	}
}

func closeTunnel() {
	cmd := exec.Command("bash", "-c", "ps x | grep ssh | grep qngfNTR | grep " + *port +" | grep -v grep | awk '{print $1}'")
	out, _ := cmd.CombinedOutput()
	p := strings.Trim(string(out),"\n")
	if p == "" {
		fmt.Println("tunnel for port " + *port + " is not found, nothing to do.")
		os.Exit(0)
	} else {
		fmt.Println("tunnel for port " + *port + " is found, close it")
		cmd = exec.Command("kill", "-9", p)
		err := cmd.Run()
		if err != nil {
			fmt.Println("close tunnel failed: " + err.Error())
			os.Exit(1)
		}
		fmt.Println("tunnel is closed.")
	}
}

func sshInvalid() bool {
	cmd := exec.Command("bash", "-c", "cat /etc/ssh/sshd_config | grep GatewayPorts | awk -F : '{print $1}'")
	out, _ := cmd.CombinedOutput()
	line := string(out)
	invalid := true
	if line == "" {
		fmt.Println("GatewayPorts not found")
		return true
	} else {
		lines := strings.Split(line,"\n")
		for i:=0; i<len(lines);i++ {
			if lines[i] == "" {
				continue
			}
			l := strings.Trim(lines[i], " ")
			if string(l[0]) == "#" {
				continue
			}

			f := strings.Split(l, " ")
			if f[len(f)-1] == "yes" {
				invalid = false
				fmt.Println("ssh config is ok: " + l)
				break
			}
		}
	}

	return invalid
}

func createTunnel() {
	var p string

	_, err := os.Stat("squids-ali-dbmotion.pem")
	if err != nil && os.IsNotExist(err) {
		genPem()
		defer rmPem()
	}

	ports := GetListenPort()

	rand.Seed(time.Now().UnixNano())
	for {
		p = strconv.Itoa(rand.Intn(65535))
		if _, ok := ports[p]; ok {
			continue
		} else {
			break
		}
	}
	msg := fmt.Sprintf("create tunnel for %s:%s on %s", *ip, *port, p)
	fmt.Println(msg)
	addr := fmt.Sprintf("%s:%s:%s", p, *ip, *port)
	cmd := exec.Command("ssh", "-qngfNTR", addr, "dbmotion@dbmotion.squids.cn", "-i", 
		"squids-ali-dbmotion.pem", "-p", "22222", "-o", "ServerAliveInterval=300")
	err = cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println("tunnel for " + *ip + ":" + *port +" on " + p + " is created." )
}

func rmPem() {
	os.Remove("squids-ali-dbmotion.pem")
}

const (
	pemContent string = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`
)

func genPem() {
	if err := ioutil.WriteFile("squids-ali-dbmotion.pem", []byte(pemContent), 0600); err != nil {
		fmt.Println("Can not generate pem file: " + err.Error())
		os.Exit(1)
	}
}

func isRoot() bool {
	cmd := exec.Command("whoami")
	out, _ := cmd.CombinedOutput()
	u := strings.Trim(string(out), "\n")
	if u != "root" {
		return false
	}
	return true
}

func main() {

	flag.Parse()

	if *action == "set" {
		if !isRoot() {
			fmt.Println("root is need to set")
			os.Exit(1)
		}
		SetSsh()
	} else if *action == "check" {
		if !isRoot() {
			fmt.Println("root is need to check")
			os.Exit(1)
		}
		if sshInvalid() {
			fmt.Println("GatewayPorts check failed")
			os.Exit(1)
		} else {
			fmt.Println("GatewayPorts check ok")
		}
	}else if *action == "create" {
		if *port == "" {
			fmt.Println("port is not set.")
			os.Exit(1)
		}
		if *ip == "" {
			fmt.Println("ip is not set.")
		}
		if sshInvalid() {
			fmt.Println("GatewayPorts must be yes.")
			os.Exit(1)
		}
		createTunnel()
	} else if *action == "close" {
		if *port == "" {
			fmt.Println("port is not set.")
			os.Exit(1)
		}
		closeTunnel()
	} else {
		fmt.Println("action must be set one of init/create/close.")
		os.Exit(1)
	}
}
