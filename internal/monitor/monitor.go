package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/adobe/butler/internal/alog"
	"github.com/adobe/butler/internal/config"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// NewMonitor returns a Monitor structure which is used to bring up the
// monitor health check and prometheus metrics http endpoints.
func NewMonitor() *Monitor {
	return &Monitor{}
}

func (m *Monitor) WithOpts(opts *MonitorOpts) *Monitor {
	m.config = opts.Config
	m.version = opts.Version
	return m
}

// Monitor is the empty structure to be used for starting up the monitor
// health check and prometheus metrics http endpoints.
type Monitor struct {
	config  *config.ButlerConfig
	mux     *http.ServeMux
	server  *http.Server
	version string
}

type MonitorOpts struct {
	Version string
	Config  *config.ButlerConfig
}

// MonitorOutput is the structure which holds the formatting which is output
// to the health check monitor. When /health-check is hit, it returns this
// structure, which is then Marshal'd to json and provided back to the end
// user
type MonitorOutput struct {
	ConfigPath       string                `json:"config-path"`
	ConfigScheme     string                `json:"config-scheme"`
	RetrieveInterval int                   `json:"retrieve-interval"`
	LogLevel         log.Level             `json:"log-level"`
	ConfigSettings   config.ConfigSettings `json:"config-settings"`
	Version          string                `json:"version"`
}

// Start turns up the http server for monitoring butler.
func (m *Monitor) Start() {
	var (
		err      error
		listener net.Listener
		mux      *http.ServeMux
		server   *http.Server
	)
	if m.mux == nil {
		mux = http.DefaultServeMux
		mux.HandleFunc("/health-check", m.MonitorHandler)
		mux.Handle("/metrics", promhttp.Handler())
		m.mux = mux
	}

	if m.config.Config.Globals.EnableHttpLog {
		loggingHandler := alog.NewApacheLoggingHandler(mux, m.config)
		server = &http.Server{
			Handler: loggingHandler,
		}
	} else {
		server = &http.Server{}
	}
	m.server = server
	if m.config.Config.Globals.HttpProto == "https" {
		cer, err := tls.LoadX509KeyPair(m.config.Config.Globals.HttpTlsCert, m.config.Config.Globals.HttpTlsKey)
		if err != nil {
			log.Fatalf("Error loading ssl certificate/key data: %s", err.Error())
		}
		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		listener, err = tls.Listen("tcp", fmt.Sprintf(":%v", m.config.Config.Globals.HttpPort), config)
		if err != nil {
			log.Fatalf("Error creating listener: %s", err.Error())
		}
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%v", m.config.Config.Globals.HttpPort))

		if err != nil {
			log.Fatalf("Error creating listener: %s", err.Error())
		}
	}
	go server.Serve(listener)
}

func (m *Monitor) Stop() error {
	timeout := 5
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	if m.server != nil {
		err := m.server.Shutdown(ctx)

		if err != nil {
			return err
		} else {
			m.server = nil
		}
	}
	return nil
}
func (m *Monitor) Update(bc *config.ButlerConfig) {
	m.config = bc
	m.Stop()
	m.Start()
}

// MonitorHandler is the handler function for the /health-check monitor
// endpoint. It displays the JSON Marshal'd output of all the various
// configuration options that buter gets started with, and some run time
// information
func (m *Monitor) MonitorHandler(w http.ResponseWriter, r *http.Request) {
	mOut := MonitorOutput{ConfigPath: m.config.GetPath(),
		ConfigScheme:     m.config.Url.Scheme,
		RetrieveInterval: m.config.Interval,
		LogLevel:         m.config.GetLogLevel(),
		ConfigSettings:   *m.config.Config,
		Version:          m.version}
	resp, err := json.Marshal(mOut)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Could not Marshal JSON, but I promise I'm up!")
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(resp))
}
