package main

import (
	"fmt"
	"flag"
	"os"
	"os/exec"
	"io"
	"net"
	"strings"
	"io/ioutil"
)

const (
	DEFAULTBUFFERLEN = 1024
)

var (
	listen = false
	command = false
	upload = false
	execute = ""
	target = ""
	uploadDestination = ""
	port = 0
)

func init() {
	flag.BoolVar(&listen, "l", false, "listen on [host]:[port] for incoming connections")
	flag.BoolVar(&command, "c", false, "initialize a command shell")
	flag.StringVar(&execute, "e", "", "execute the given file upon receiving a connection")
	flag.StringVar(&uploadDestination, "u", "", "upon receiving connection upload a file and write to [destination]")
	flag.StringVar(&target, "t", "0.0.0.0", "target_host")
	flag.IntVar(&port, "p", 0, "port")
}
func usage() {
	s := `
	Netcat Replacement
        Usage: bhpnet.py -t target_host -p port
        -l                - listen on [host]:[port] for incoming connections
        -e file_to_run   - execute the given file upon receiving a connection
        -c                - initialize a command shell
        -u destination    - upon receiving connection upload a file and write to [destination]

        Examples: 
        bhpnet.py -t 192.168.0.1 -p 5555 -l -c
        bhpnet.py -t 192.168.0.1 -p 5555 -l -u=c:\\target.exe
        bhpnet.py -t 192.168.0.1 -p 5555 -l -e=\"cat /etc/passwd\"
        echo 'ABCDEFGHI' | ./bhpnet.py -t 192.168.11.12 -p 135	
	`
	fmt.Println(s)
	os.Exit(0)
}
func readSomething(reader io.Reader, minLength int) []byte {
	buf := make([]byte, minLength)
	allbuf := make([]byte, 0)
	totallen:=0
	for {
		n, err := reader.Read(buf)
		fmt.Println("read :",n,totallen)
		if err != nil {
			break
		}
		totallen+=n
		allbuf = append(allbuf, buf[0:n]...)
		if n < minLength {
			break
		}
	}
	return allbuf
}
// if we don't listen we are a client....make it so.
func clientSender(rw io.ReadWriter) {

	client, err := net.Dial("tcp", fmt.Sprintf("%s:%d", target, port))
	if err != nil {
		fmt.Printf("%s", err)
		return
	}
	defer client.Close()
	buffer := make([]byte, DEFAULTBUFFERLEN)
	if command {
		//execute a shell
		fmt.Println("execute shell")
		for {
			result := readSomething(client, DEFAULTBUFFERLEN)
			rw.Write(result)
			cmd := readSomething(rw, DEFAULTBUFFERLEN)
			client.Write(cmd)
		}

	} else {
		var totallen=0
		//send file
		for {
			//if we detect input from stdin send it
			// if not we are going to wait for the user to punch some in
			n, err := rw.Read(buffer)
			if n <= 0 || err == io.EOF {
				break;
			}
			//fmt.Println("ioread:",n,string(buffer[:n]))
			totallen+=n
			client.Write(buffer[0:n])
		}
		// if not call close, reciver's read will not finish ,go cannot shutdown(READ)
		// client.Close()
		//fmt.Println("send file complete:",totallen)
		//result := readSomething(client, DEFAULTBUFFERLEN)
		//fmt.Println(string(result))

	}

}
func runCommand(command string) []byte {
	command = strings.TrimRight(command, "\r\n")
	allarg := strings.Split(command, " ")
	args := make([]string, 1)
	args[0] = "/c"
	args = append(args, allarg...)
	cmd := exec.Command("cmd", args...)
	out, _ := cmd.Output()
	//if err != nil {
	//	fmt.Println("error")
	//	return []byte(err.Error())
	//}
	return out
}
func clientHandler(client net.Conn) {
	fmt.Println(client.RemoteAddr(), " arrived")
	defer client.Close()
	//check for upload
	if len(uploadDestination) > 0 {
		//read in all of the bytes and write to our destination
		allbuf := readSomething(client, DEFAULTBUFFERLEN)
		fmt.Println("read all file:",len(allbuf))
		err := ioutil.WriteFile(uploadDestination, allbuf, os.ModePerm)
		if err != nil {
			client.Write([]byte(fmt.Sprintf("Failed to save file to %s\r\n", uploadDestination)))
		} else {
			client.Write([]byte(fmt.Sprintf("Successfully to save file to %s\r\n", uploadDestination)))
		}
	}
	if len(execute) > 0 {
		output := runCommand(execute)
		client.Write(output)
	}
	if command {
		cmdbuf := make([]byte, DEFAULTBUFFERLEN)
		for {
			client.Write([]byte("<BHP:#> "))
			fmt.Println("write bhp")
			command := make([]byte, 0)
			for {
				n, err := client.Read(cmdbuf)
				if err != nil || n <= 0 {
					return
				}
				fmt.Println(string(cmdbuf[0:n]))
				command = append(command, cmdbuf[0:n]...)
				if strings.Index(string(command), "\n") > 0 {
					break
				}
			}
			fmt.Println("runcommand", string(command))
			output := runCommand(string(command))
			client.Write(output)

		}
	}
}

func serverLoop() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", target, port))
	if err != nil {
		fmt.Printf("listen on port %d error\n", port)
		return
	}
	fmt.Printf("Listenging on %s:%d\n", target, port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("accept error:", err)
			break
		}
		go clientHandler(conn)
	}

}
func main() {
	if false {
		out := runCommand("date /?\r\n")
		fmt.Println(string(out))
	}
	if len(os.Args) < 2 {
		usage()
	}
	flag.Parse()
	if !listen && len(target) > 0 && port > 0 {
		clientSender(os.Stdin)
	}
	if listen {
		serverLoop()
	}
}
