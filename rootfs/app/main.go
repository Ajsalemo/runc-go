package main

import (
	"crypto/tls"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	controllers "go-tcp-server/controllers"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

func handleRequest(conn net.Conn, server string) {
	// close conn
	defer conn.Close()
	// write data to response
	time := time.Now().Format(time.ANSIC)
	res := "TCP server " + server + ": Recieved connection at: " + time
	zap.L().Info("TCP server " + server + ": Recieved connection at: " + time)
	conn.Write([]byte(res))

}

func tcpServerOne() {
	HOST := "0.0.0.0"
	PORT := "8443"
	TYPE := "tcp"
	SERVER := "[1]"

	cert, err := tls.LoadX509KeyPair("./certs/cert.pem", "./certs/key.pem")
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	zap.L().Info("TCP server " + SERVER + " is listening on TLS port " + PORT)
	listen, err := tls.Listen(TYPE, HOST+":"+PORT, config)
	if err != nil {
		zap.L().Fatal(err.Error())
	}
	// close listener
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			zap.L().Fatal(err.Error())
		}
		go handleRequest(conn, SERVER)
	}
}

func tcpServerTwo() {
	HOST := "0.0.0.0"
	PORT := "8444"
	TYPE := "tcp"
	SERVER := "[2]"

	cert, err := tls.LoadX509KeyPair("./certs/cert.pem", "./certs/key.pem")
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	zap.L().Info("TCP server " + SERVER + " is listening on TLS port " + PORT)
	listen, err := tls.Listen(TYPE, HOST+":"+PORT, config)
	if err != nil {
		zap.L().Fatal(err.Error())
	}
	// close listener
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			zap.L().Fatal(err.Error())
		}
		go handleRequest(conn, SERVER)
	}
}

func main() {
	app := fiber.New()
	app.Get("/", controllers.IndexController)
	// Notify the application of the below signals to be handled on shutdown
	s := make(chan os.Signal, 1)
	signal.Notify(s,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	// Goroutine to clean up prior to shutting down
	go func() {
		sig := <-s
		switch sig {
		case os.Interrupt:
			zap.L().Warn("CTRL+C / os.Interrupt recieved, shutting down the application..")
			app.Shutdown()
		case syscall.SIGTERM:
			zap.L().Warn("SIGTERM recieved.., shutting down the application..")
			app.Shutdown()
		case syscall.SIGQUIT:
			zap.L().Warn("SIGQUIT recieved.., shutting down the application..")
			app.Shutdown()
		case syscall.SIGINT:
			zap.L().Warn("SIGINT recieved.., shutting down the application..")
			app.Shutdown()
		}
	}()
	// Start multiple tcp servers
	go tcpServerOne()
	go tcpServerTwo()

	zap.L().Info("Fiber (HTTP) server is running on port 3000")
	fiberErr := app.Listen(":3000")

	if fiberErr != nil {
		zap.L().Fatal(fiberErr.Error())
	}
}
