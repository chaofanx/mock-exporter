package main

import (
	"fmt"
	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"net"
	"net/http"
	"os"
	"os/user"
	"sync"
	"sync/atomic"
	"time"
)

func startWebServer(port int, path, mock *string) error {
	addr := fmt.Sprintf(":%d", port)

	// Check if the port is already in use
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc(*path, GetHandle(mock))

	// Start the web server
	err = http.Serve(listener, mux)
	if err != nil {
		return err
	}

	return nil
}

func ReadFile(name string) (string, error) {
	file, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func GetHandle(mock *string) func(w http.ResponseWriter, r *http.Request) {
	// read file to string
	file, err := ReadFile(*mock)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, file)
	}
}

func main() {
	var (
		metricsPath = kingpin.Flag(
			"path",
			"Path under which to expose metrics.",
		).Short('p').Default("/metrics").String()

		mock = kingpin.Flag(
			"mock",
			"Sample prom file (.prom) that requires mocking",
		).Short('m').Required().String()

		portStart = kingpin.Flag(
			"web.port",
			"The starting value of the port",
		).Default("10000").Int()

		portLength = kingpin.Flag(
			"web.length",
			"The length of the port range (starting from the starting value. If any port is occupied, it will be skipped.)",
		).Default("50").Int()
	)

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	u, _ := user.Current()
	version.BuildUser = u.Name
	version.BuildDate = time.Now().Format("2006-01-02 15:04:05")
	version.Branch = "master"
	kingpin.Version(version.Print("mock_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting mock_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())
	if curUser, err := user.Current(); err == nil && curUser.Uid == "0" {
		level.Warn(logger).Log("msg", "Node Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
	}

	var wg sync.WaitGroup
	wg.Add(*portLength)

	var portUsed int32 = 0
	for i := 0; i < *portLength; i++ {
		port := *portStart + i
		go func(p int) {
			defer wg.Done()
			err := startWebServer(p, metricsPath, mock)
			if err != nil {
				fmt.Printf("Error starting server on port %d: %v\n", p, err)
			}
			atomic.AddInt32(&portUsed, 1)
		}(port)
	}

	wg.Wait()

	level.Info(logger).Log("msg", fmt.Sprintf("%d ports have started listening and the application started successfully", portUsed))
}
