package monitoring

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/adetunjii/google-sheets-connector/pkg/utils/httputils"
	"github.com/prometheus/client_golang/prometheus"
)

// metrics related metadata
type Options struct {
	ID                 string
	Name               string
	Version            string
	MetricsLabelPrefix string
}

type Option func(*Options)

type metricsDefinition struct {
	opsCounter            *prometheus.CounterVec
	responseStatusCounter *prometheus.CounterVec
	timeCounterSummary    *prometheus.SummaryVec
	timeCounterHistogram  *prometheus.HistogramVec
}

var metrics metricsDefinition
var mu sync.Mutex

type MetricsHandler struct {
	options Options
}

func (m *MetricsHandler) registerMetrics(metricsLabelPrefix string) {
	mu.Lock()
	defer mu.Unlock()

	if metrics.opsCounter == nil {
		metrics.opsCounter = initOpsCounterVec(
			"http_requests_total",
			"Total http requests by endpoints",
			metricsLabelPrefix,
		)
	}

	if metrics.responseStatusCounter == nil {
		metrics.responseStatusCounter = initResponseStatusCounterVec(
			"response_status",
			"Response status by endpoints",
			metricsLabelPrefix,
		)
	}

	if metrics.timeCounterHistogram == nil {
		metrics.timeCounterHistogram = initHistogramVec(
			"requests_latency_seconds",
			"Requests latency in seconds by endpoints",
			metricsLabelPrefix,
		)
	}

	if metrics.timeCounterSummary == nil {
		metrics.timeCounterSummary = initSummaryVec(
			"requests_duration_seconds",
			"Requests duration in seconds by endpoints",
			metricsLabelPrefix,
		)
	}

	// register each of the collectors and check for errors
	for _, c := range []prometheus.Collector{metrics.opsCounter, metrics.responseStatusCounter, metrics.timeCounterHistogram, metrics.timeCounterSummary} {
		if err := prometheus.DefaultRegisterer.Register(c); err != nil {
			promErr := prometheus.AlreadyRegisteredError{}
			if errors.As(err, &promErr) {
				//TODO: use app logger
				log.Printf("prometheus collector already registered", err)
			}
		}
	}
}

func (m *MetricsHandler) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		endpoint := request.URL.Path

		timer := prometheus.NewTimer(
			prometheus.ObserverFunc(func(v float64) {
				metrics.timeCounterSummary.WithLabelValues(m.options.ID, m.options.Name, m.options.Version, endpoint).Observe(v)
				metrics.timeCounterHistogram.WithLabelValues(m.options.ID, m.options.Name, m.options.Version, endpoint).Observe(v)
			}),
		)
		defer timer.ObserveDuration()

		w := httputils.NewResponseWriter(rw)
		next.ServeHTTP(w, request)
		statusCode := w.StatusCode

		// metrics counter per response_status_code
		metrics.responseStatusCounter.WithLabelValues(m.options.ID, m.options.Name, m.options.Version, endpoint, strconv.Itoa(statusCode)).Inc()

		// metrics counter for total number of requests
		metrics.opsCounter.WithLabelValues(m.options.ID, m.options.Name, m.options.Version, endpoint).Inc()

	})
}

func initMetrics(opts []Option) *MetricsHandler {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	m := &MetricsHandler{
		options: options,
	}

	m.registerMetrics(options.MetricsLabelPrefix)
	return m
}

func NewMetricsWrapper(opts ...Option) *MetricsHandler {
	handler := initMetrics(opts)
	return handler
}

func ServiceID(id string) Option {
	return func(opts *Options) {
		opts.ID = id
	}
}

func ServiceName(name string) Option {
	return func(opts *Options) {
		opts.Name = name
	}
}

func ServiceVersion(version string) Option {
	return func(opts *Options) {
		opts.Version = version
	}
}

func ServiceMetricsLabelPrefix(metricsLabelPrefix string) Option {
	return func(opts *Options) {
		opts.MetricsLabelPrefix = metricsLabelPrefix
	}
}

func initOpsCounterVec(name, help, metricsLabelPrefix string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},

		[]string{
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "id"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "name"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "version"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "endpoint"),
		},
	)
}

func initResponseStatusCounterVec(name, help, metricsLabelPrefix string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},

		[]string{
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "id"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "name"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "version"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "endpoint"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "response_status"),
		},
	)
}

func initSummaryVec(name, help, metricsLabelPrefix string) *prometheus.SummaryVec {
	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: name,
			Help: help,
		},
		[]string{
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "id"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "name"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "version"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "endpoint"),
		},
	)
}

func initHistogramVec(name, help, metricsLabelPrefix string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: name,
			Help: help,
		},
		[]string{
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "id"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "name"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "version"),
			fmt.Sprintf("%s_%s", metricsLabelPrefix, "endpoint"),
		},
	)
}
