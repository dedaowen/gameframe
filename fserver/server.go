package fserver

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dedaowen/gameframe/fnet"
	"github.com/dedaowen/gameframe/iface"
	"github.com/dedaowen/gameframe/logger"
	"github.com/dedaowen/gameframe/timer"
	"github.com/dedaowen/gameframe/utils"
	"github.com/gorilla/websocket"
)

func init() {
	utils.GlobalObject.Protoc = fnet.NewProtocol()
	// --------------------------------------------init log start
	utils.ReSettingLog()
	// --------------------------------------------init log end
}

type Server struct {
	Name          string
	IPVersion     string
	IP            string
	Port          int
	MaxConn       int
	GenNum        *utils.UUIDGenerator
	connectionMgr iface.Iconnectionmgr
	Protoc        iface.IServerProtocol
}

func NewServer() iface.Iserver {
	s := &Server{
		Name:          utils.GlobalObject.Name,
		IPVersion:     "tcp4",
		IP:            "0.0.0.0",
		Port:          utils.GlobalObject.TcpPort,
		MaxConn:       utils.GlobalObject.MaxConn,
		connectionMgr: fnet.NewConnectionMgr(),
		Protoc:        utils.GlobalObject.Protoc,
		GenNum:        utils.NewUUIDGenerator(""),
	}
	utils.GlobalObject.TcpServer = s

	return s
}

func NewTcpServer(name string, version string, ip string, port int, maxConn int, protoc iface.IServerProtocol) iface.Iserver {
	s := &Server{
		Name:          name,
		IPVersion:     version,
		IP:            ip,
		Port:          port,
		MaxConn:       maxConn,
		connectionMgr: fnet.NewConnectionMgr(),
		Protoc:        protoc,
		GenNum:        utils.NewUUIDGenerator(""),
	}
	utils.GlobalObject.TcpServer = s

	return s
}

func (this *Server) handleConnection(conn net.Conn) {
	if conn, ok := conn.(*net.TCPConn); ok {
		conn.SetNoDelay(true)
		conn.SetKeepAlive(true)
	}
	// conn.SetDeadline(time.Now().Add(time.Minute * 2))
	var fconn *fnet.Connection
	if this.Protoc == nil {
		fconn = fnet.NewConnection(conn, this.GenNum.GetUint32(), utils.GlobalObject.Protoc)
	} else {
		fconn = fnet.NewConnection(conn, this.GenNum.GetUint32(), this.Protoc)
	}
	fconn.SetProperty(fnet.XINGO_CONN_PROPERTY_NAME, this.Name)
	fconn.Start()
}

func (this *Server) Start() {
	utils.GlobalObject.TcpServers[this.Name] = this
	go func() {
		this.Protoc.InitWorker(utils.GlobalObject.PoolSize)
		if this.IPVersion == "ws" {
			this.startWs()
		} else {
			this.startTCP()
		}
	}()
}
func (this *Server) startTCP() {
	tcpAddr, err := net.ResolveTCPAddr(this.IPVersion, fmt.Sprintf("%s:%d", this.IP, this.Port))
	if err != nil {
		logger.Fatal("ResolveTCPAddr err: ", err)
		return
	}
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logger.Error(err)
	}
	logger.Info(fmt.Sprintf("start server %s...", this.Name))
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			logger.Error(err)
		}
		//max client exceed
		if this.connectionMgr.Len() >= utils.GlobalObject.MaxConn {
			conn.Close()
		} else {
			go this.handleConnection(conn)
		}
	}
}
func (this *Server) startWs() {
	fmt.Println("startws -------", this.IP)
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     utils.GlobalObject.CheckOrigin,
	}
	http.HandleFunc("/"+strings.TrimPrefix(this.IP, "/"), func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error(fmt.Sprintf("Upgrade failure, URI=%s, Error=%s", r.RequestURI, err.Error()))
			return
		}

		c, err := NewWSConn(conn)
		if err != nil {
			logger.Error(fmt.Sprintf("wscon new failure, URI=%s, Error=%s", r.RequestURI, err.Error()))
			return
		}

		this.handleConnection(c)
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%d", this.Port), nil); err != nil {
		logger.Fatal(err)
	} else {
		logger.Info(fmt.Sprintf("start wsserver %s...", this.Name))
	}
}

func (this *Server) GetConnectionMgr() iface.Iconnectionmgr {
	return this.connectionMgr
}

func (this *Server) GetConnectionQueue() chan interface{} {
	return nil
}

func (this *Server) Stop() {
	logger.Info("stop xingo server ", this.Name)
	if utils.GlobalObject.OnServerStop != nil {
		utils.GlobalObject.OnServerStop()
	}
}

func (this *Server) AddRouter(router interface{}) {
	logger.Info("AddRouter")
	utils.GlobalObject.Protoc.GetMsgHandle().AddRouter(router)
}

func (this *Server) CallLater(durations time.Duration, f func(v ...interface{}), args ...interface{}) {
	delayTask := timer.NewTimer(durations, f, args)
	delayTask.Run()
}

func (this *Server) CallWhen(ts string, f func(v ...interface{}), args ...interface{}) {
	loc, err_loc := time.LoadLocation("Local")
	if err_loc != nil {
		logger.Error(err_loc)
		return
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", ts, loc)
	now := time.Now()
	if err == nil {
		if now.Before(t) {
			this.CallLater(t.Sub(now), f, args...)
		} else {
			logger.Error("CallWhen time before now")
		}
	} else {
		logger.Error(err)
	}
}

func (this *Server) CallLoop(durations time.Duration, f func(v ...interface{}), args ...interface{}) {
	go func() {
		delayTask := timer.NewTimer(durations, f, args)
		for {
			time.Sleep(delayTask.GetDurations())
			delayTask.GetFunc().Call()
		}
	}()
}

func (this *Server) WaitSignal() {
	signal.Notify(utils.GlobalObject.ProcessSignalChan, os.Kill, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	sig := <-utils.GlobalObject.ProcessSignalChan
	logger.Info(fmt.Sprintf("server exit. signal: [%s]", sig))
	this.Stop()
}

func (this *Server) Serve() {
	this.Start()
	this.WaitSignal()
}
