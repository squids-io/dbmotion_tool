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
	err := cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println("tunnel for " + *ip + ":" + *port +" on " + p + " is created." )
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
