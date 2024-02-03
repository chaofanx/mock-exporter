package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"log"
	"mock-exporter/common"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"sync/atomic"
	"syscall"
	"time"
)

func startWebServer(port int, path, mock *string, chaos *float64) (func(server *http.Server), error) {
	addr := fmt.Sprintf(":%d", port)

	// Check if the port is already in use
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc(*path, getHandle(mock, chaos))
	// Start the web server
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Server failed to start: %v\n", err)
	}
	return func(server *http.Server) {
		if err := server.Shutdown(context.Background()); err != nil {
			fmt.Printf("Server shutdown error: %v\n", err)
		}
	}, nil
}

func getLabelNames(labels []*dto.LabelPair) []string {
	var result []string
	for _, label := range labels {
		result = append(result, *label.Name)
	}
	return result
}

func getLabelValues(labels []*dto.LabelPair) []string {
	var result []string
	for _, label := range labels {
		result = append(result, *label.Value)
	}
	return result
}

// getHandle returns a handler function that serves the metrics from the given file
// The file should be in prometheus text format
func getHandle(mock *string, chaos *float64) func(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()
	file, err := os.Open(*mock)
	var parser expfmt.TextParser
	metrics, err := parser.TextToMetricFamilies(bufio.NewReader(file))
	if err != nil {
		log.Fatalf("Failed to decode metrics: %v", err)
	}

	for _, metric := range metrics {
		metricType := *metric.Type
		switch metricType {
		case dto.MetricType_COUNTER:
			if len(metric.Metric) < 1 {
				fmt.Printf("No metric values for %v\n", *metric.Name)
				continue
			}
			counter := prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: *metric.Name,
				Help: *metric.Help,
			}, getLabelNames(metric.Metric[0].Label))
			registry.MustRegister(counter)
			for _, metricValue := range metric.Metric {
				counter.WithLabelValues(getLabelValues(metricValue.Label)...).Add(*metricValue.Counter.Value)
				var counterValue = metricValue
				go func() {
					// Generate random numbers using normal distribution
					for {
						vibration := common.RandVibrate(1, *chaos)
						if vibration < 0 {
							vibration = 0
						}
						counter.WithLabelValues(getLabelValues(counterValue.Label)...).Add(vibration)
						time.Sleep(time.Duration(1000) * time.Millisecond)
					}
				}()
			}
		case dto.MetricType_GAUGE:
			if len(metric.Metric) < 1 {
				fmt.Printf("No metric values for %v\n", *metric.Name)
				continue
			}
			gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: *metric.Name,
				Help: *metric.Help,
			}, getLabelNames(metric.Metric[0].Label))
			registry.MustRegister(gauge)
			for _, metricValue := range metric.Metric {
				gauge.WithLabelValues(getLabelValues(metricValue.Label)...).Set(*metricValue.Gauge.Value)
				var gaugeValue = metricValue
				go func() {
					// Generate random numbers using normal distribution
					for {
						vibration := common.RandVibrate(*gaugeValue.Gauge.Value, *chaos)
						gauge.WithLabelValues(getLabelValues(gaugeValue.Label)...).Set(vibration)
						time.Sleep(time.Duration(1000) * time.Millisecond)
					}
				}()
			}
		case dto.MetricType_SUMMARY:
			if len(metric.Metric) < 1 {
				fmt.Printf("No metric values for %v\n", *metric.Name)
				continue
			}
			summary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
				Name: *metric.Name,
				Help: *metric.Help,
			}, getLabelNames(metric.Metric[0].Label))
			registry.MustRegister(summary)
			for _, metricValue := range metric.Metric {
				summary.WithLabelValues(getLabelValues(metricValue.Label)...).Observe(*metricValue.Summary.SampleSum)
			}
		case dto.MetricType_HISTOGRAM:
			if len(metric.Metric) < 1 {
				fmt.Printf("No metric values for %v\n", *metric.Name)
				continue
			}
			histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: *metric.Name,
				Help: *metric.Help,
			}, getLabelNames(metric.Metric[0].Label))
			registry.MustRegister(histogram)
			for _, metricValue := range metric.Metric {
				histogram.WithLabelValues(getLabelValues(metricValue.Label)...).Observe(*metricValue.Histogram.SampleSum)
			}
		default:
			fmt.Printf("Unknown metric type: %v\n", metricType)
		}
	}

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	handler := promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		})
	return func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
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

		chaos = kingpin.Flag(
			"chaos",
			"Chaos coefficient",
		).Short('c').Default("1.0").Float64()
	)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
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

	level.Info(logger).Log("msg", fmt.Sprintf("Current chaos is %f", *chaos))

	var portUsed int32 = 0
	for i := 0; i < *portLength; i++ {
		port := *portStart + i
		go func(p int) {
			_, err := startWebServer(p, metricsPath, mock, chaos)
			if err != nil {
				fmt.Printf("Error starting server on port %d: %v\n", p, err)
			}
			atomic.AddInt32(&portUsed, 1)
		}(port)
	}

	level.Info(logger).Log("msg", fmt.Sprintf("%d ports have started listening and the application started successfully", portUsed))

	<-stopChan

	level.Info(logger).Log("msg", "Received SIGINT/SIGTERM, exiting gracefully...")

}
