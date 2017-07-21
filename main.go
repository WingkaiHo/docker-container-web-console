package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

var port = flag.String("port", "8080", "Port for server")
var host = flag.String("host", "127.0.0.1:2375", "Docker host")

func main() {
	flag.Parse()
	http.Handle("/exec/", websocket.Handler(ExecContainer))
	http.Handle("/", http.FileServer(http.Dir("./")))
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		glog.Errorln(err)
	}
}

func ExecContainer(ws *websocket.Conn) {
	container := ws.Request().URL.Path[len("/exec/"):]
	fmt.Println(container)
	if container == "" {
		ws.Write([]byte("Container does not exist"))
		return
	}
	type stuff struct {
		Id string
	}
	var s stuff
	params := bytes.NewBufferString("{\"AttachStdin\":true,\"AttachStdout\":true,\"AttachStderr\":true,\"Tty\":true,\"Cmd\":[\"/bin/bash\"]}")
	resp, err := http.Post("http://"+*host+"/containers/"+container+"/exec", "application/json", params)
	if err != nil {
		glog.Errorln(err)
		panic(err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorln(err)
		panic(err)
	}
	json.Unmarshal([]byte(data), &s)
	glog.Infoln("request id: " + s.Id)
	if err := hijack(*host, "POST", "/exec/"+s.Id+"/start", true, ws, ws, ws, nil, nil); err != nil {
		panic(err)
	}

	glog.Infoln("Create console conntion tty from: " + ws.RemoteAddr().String() + " successful.")
	spew.Dump(ws)
}

func hijack(addr, method, path string, setRawTerminal bool, in io.ReadCloser, stdout, stderr io.Writer, started chan io.Closer, data interface{}) error {
	params := bytes.NewBufferString("{\"Detach\": false, \"Tty\": true}")
	req, err := http.NewRequest(method, path, params)
	fmt.Println(req)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Docker-Client")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "tcp")
	req.Host = addr

	dial, err := net.Dial("tcp", addr)
	// When we set up a TCP connection for hijack, there could be long periods
	// of inactivity (a long running command with no output) that in certain
	// network setups may cause ECONNTIMEOUT, leaving the client in an unknown
	// state. Setting TCP KeepAlive on the socket connection will prohibit
	// ECONNTIMEOUT unless the socket connection truly is broken
	if tcpConn, ok := dial.(*net.TCPConn); ok {
		glog.Infof("Setup up a tcp conection for hijack ok")
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	if err != nil {
		glog.Error()
		return err
	}
	clientconn := httputil.NewClientConn(dial, nil)
	defer clientconn.Close()

	// Server hijacks the connection, error 'connection closed' expected
	clientconn.Do(req)

	rwc, br := clientconn.Hijack()
	defer rwc.Close()

	if started != nil {
		started <- rwc
	}

	var receiveStdout chan error

	if stdout != nil || stderr != nil {
		go func() (err error) {
			if setRawTerminal && stdout != nil {
				// read data from docker container bash and send to browser terminal
				_, err = io.Copy(stdout, br)
			}
			return err
		}()
	}
	fmt.Println("a")
	go func() error {
		if in != nil {
			// read data from browser and send to docker container bash
			io.Copy(rwc, in)
		}

		// browser clouse terminal, and send the exit code to docker container bash
		exitCode := []byte{'e', 'x', 'i', 't', '\n'}
		rwc.Write(exitCode)
		if conn, ok := rwc.(interface {
			CloseWrite() error
		}); ok {
			if err := conn.CloseWrite(); err != nil {
			}
		}
		return nil
	}()
	if stdout != nil || stderr != nil {
		if err := <-receiveStdout; err != nil {
			return err
		}
	}
	spew.Dump(br)
	go func() {
		for {
			glog.Info(br)
			spew.Dump(br)
		}
	}()

	return nil
}
