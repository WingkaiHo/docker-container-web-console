package main

import (
	"bytes"
	"flag"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

var port = flag.String("port", "8080", "Port for server")
var host = flag.String("host", "unix:///var/run/docker.sock", "Docker host for example unix://var/run/docker.sock or tcp://127.0.0.1:2375")

func main() {
	flag.Parse()
	http.Handle("/exec/", websocket.Handler(ExecContainer))
	http.Handle("/", http.FileServer(http.Dir("./")))
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		glog.Errorln(err)
	}
}

func ExecContainer(ws *websocket.Conn) {
	glog.Infof("Request: %s %s", ws.RemoteAddr().String(), ws.Request().URL.String())
	container := ws.Request().URL.Path[len("/exec/"):]
	if container == "" {
		ws.Write([]byte("Container does not exist"))
		return
	}

	dockerClient, err := docker.NewClient(*host)
	if err != nil {
		glog.Errorf("Create the docker client fail %s", err.Error())
		return
	}

	execRes, err := dockerClient.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"/bin/bash"},
		Container:    container,
	})

	if err != nil {
		glog.Errorln(err)
		return
	}

	if err := hijack(*host, "POST", "/exec/"+execRes.ID+"/start", true, ws, ws, ws, nil, nil); err != nil {
		panic(err)
	}

	glog.Infoln("Create console conntion tty from: " + ws.RemoteAddr().String() + " successful.")
	spew.Dump(ws)
}

func hijack(addr, method, path string, setRawTerminal bool, in io.ReadCloser, stdout, stderr io.Writer, started chan io.Closer, data interface{}) error {
	var network string
	params := bytes.NewBufferString("{\"Detach\": false, \"Tty\": true}")
	req, err := http.NewRequest(method, path, params)
	if err != nil {
		glog.Infoln(err.Error())
		return err
	}

	if strings.HasPrefix(addr, "tcp:") {
		network = "tcp"
		addr = addr[4:]
	} else if strings.HasPrefix(addr, "unix:") {
		network = "unix"
		addr = addr[5:]
	}

	req.Header.Set("User-Agent", "Docker-Client")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "tcp")
	req.Host = addr

	dial, err := net.Dial(network, addr)
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
		glog.Errorln(err.Error())
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
