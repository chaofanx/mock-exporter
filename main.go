package main

import (
	"fmt"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"mock-exporter/tool"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/user"
	"sync"
	"sync/atomic"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/common/version"
)

// handler wraps an unfiltered http.Handler but uses a filtered handler,
// created on the fly, if filtering is requested. Create instances with
// newHandler.
type handler struct {
	unfilteredHandler http.Handler
	// exporterMetricsRegistry is a separate registry for the metrics about
	// the exporter itself.
	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
	logger                  log.Logger
}

func newHandler(includeExporterMetrics bool, maxRequests int, logger log.Logger) *handler {
	h := &handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		includeExporterMetrics:  includeExporterMetrics,
		maxRequests:             maxRequests,
		logger:                  logger,
	}
	if h.includeExporterMetrics {
		h.exporterMetricsRegistry.MustRegister(
			promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
			promcollectors.NewGoCollector(),
		)
	}
	return h
}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	level.Debug(h.logger).Log("msg", "collect query:", "filters", filters)

	if len(filters) == 0 {
		// No filters, use the prepared unfiltered handler.
		h.unfilteredHandler.ServeHTTP(w, r)
		return
	}
	// To serve filtered metrics, we create a filtering handler on the fly.
	filteredHandler, err := h.innerHandler(filters...)
	if err != nil {
		level.Warn(h.logger).Log("msg", "Couldn't create filtered metrics handler:", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create filtered metrics handler: %s", err)))
		return
	}
	filteredHandler.ServeHTTP(w, r)
}

// innerHandler is used to create both the one unfiltered http.Handler to be
// wrapped by the outer handler and also the filtered handlers created on the
// fly. The former is accomplished by calling innerHandler without any arguments
// (in which case it will log all the collectors enabled via command-line
// flags).
func (h *handler) innerHandler(filters ...string) (http.Handler, error) {

	//r := prometheus.NewRegistry()
	//r.MustRegister(version.NewCollector("node_exporter"))
	//if err := r.Register(nc); err != nil {
	//	return nil, fmt.Errorf("couldn't register node collector: %s", err)
	//}

	//var handler http.Handler
	//if h.includeExporterMetrics {
	//	handler = promhttp.HandlerFor(
	//		prometheus.Gatherers{h.exporterMetricsRegistry, r},
	//		promhttp.HandlerOpts{
	//			ErrorLog:            stdlog.New(log.NewStdlibAdapter(level.Error(h.logger)), "", 0),
	//			ErrorHandling:       promhttp.ContinueOnError,
	//			MaxRequestsInFlight: h.maxRequests,
	//			Registry:            h.exporterMetricsRegistry,
	//		},
	//	)
	//	// Note that we have to use h.exporterMetricsRegistry here to
	//	// use the same promhttp metrics for all expositions.
	//	handler = promhttp.InstrumentMetricHandler(
	//		h.exporterMetricsRegistry, handler,
	//	)
	//} else {
	//	handler = promhttp.HandlerFor(
	//		r,
	//		promhttp.HandlerOpts{
	//			ErrorLog:            stdlog.New(log.NewStdlibAdapter(level.Error(h.logger)), "", 0),
	//			ErrorHandling:       promhttp.ContinueOnError,
	//			MaxRequestsInFlight: h.maxRequests,
	//		},
	//	)
	//}
	//
	//return handler, nil
	return nil, nil
}

func startWebServer(port int, path *string) error {
	addr := fmt.Sprintf(":%d", port)

	// Check if the port is already in use
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	fmt.Printf("Starting server on port %d\n", port)

	// Your web server logic here
	http.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from port %d!", port)
	})

	// Start the web server
	err = http.Serve(listener, nil)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var (
		metricsPath = kingpin.Flag(
			"path",
			"Path under which to expose metrics.",
		).Short('p').Default("/metrics").String()

		//mocks = kingpin.Flag(
		//	"mock",
		//	"Sample prom file (.prom) that requires mocking",
		//).Short('m').Required().Strings()

		portStart = kingpin.Flag(
			"web.port",
			"The starting value of the port",
		).Default("10000").Int()

		portLength = kingpin.Flag(
			"web.length",
			"The length of the port range (starting from the starting value)",
		).Default("50").Int()
	)

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("mock_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting mock_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		level.Warn(logger).Log("msg", "Node Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
	}

	var wg sync.WaitGroup
	wg.Add(*portLength)

	var portUsed int32 = 0
	for i := 0; i < *portLength; i++ {
		port := *portStart + i
		tool.PortCheck(port)
		go func(p int) {
			fmt.Println(p, "current")
			defer wg.Done()
			err := startWebServer(p, metricsPath)
			if err != nil {
				fmt.Printf("Error starting server on port %d: %v\n", p, err)
			}
			atomic.AddInt32(&portUsed, 1)
		}(port)
	}

	wg.Wait()

	level.Info(logger).Log("msg", fmt.Sprintf("%d ports have started listening and the application started successfully", portUsed))
}
