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

// NewMonitor returns a Monitor object which is used to bring up the
// monitor health check and prometheus metrics http endpoints.
func NewMonitor() *Monitor {
	return &Monitor{}
}

// WithOpts returns the Monitor object and sets the object values
// to the options which were passed in.
func (m *Monitor) WithOpts(opts *Opts) *Monitor {
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

// Opts is an object which stores the Monitor object's configuration details
// It expects a butler version which will be used for the monitor output,
// and the butler configuration.
type Opts struct {
	Version string
	Config  *config.ButlerConfig
}

// Output is the structure which holds the formatting which is output
// to the health check monitor. When /health-check is hit, it returns this
// structure, which is then Marshal'd to json and provided back to the end
// user
type Output struct {
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
		mux.HandleFunc("/health-check", m.Handler)
		mux.Handle("/metrics", promhttp.Handler())
		m.mux = mux
	}

	if m.config.Config.Globals.EnableHTTPLog {
		loggingHandler := alog.NewApacheLoggingHandler(mux, m.config)
		server = &http.Server{
			Handler: loggingHandler,
		}
	} else {
		server = &http.Server{}
	}
	m.server = server
	if m.config.Config.Globals.HTTPProto == "https" {
		cer, err := tls.LoadX509KeyPair(m.config.Config.Globals.HTTPTLSCert, m.config.Config.Globals.HTTPTLSKey)
		if err != nil {
			log.Fatalf("Error loading ssl certificate/key data: %s", err.Error())
		}
		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		listener, err = tls.Listen("tcp", fmt.Sprintf(":%v", m.config.Config.Globals.HTTPPort), config)
		if err != nil {
			log.Fatalf("Error creating listener: %s", err.Error())
		}
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%v", m.config.Config.Globals.HTTPPort))

		if err != nil {
			log.Fatalf("Error creating listener: %s", err.Error())
		}
	}
	go server.Serve(listener)
}

// Stop is to shut down the butler webserver used for the monitor and health
// checking. This is really for testing purposes only.
func (m *Monitor) Stop() error {
	timeout := 5
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	if m.server != nil {
		err := m.server.Shutdown(ctx)

		if err != nil {
			return err
		}
	}
	m.server = nil
	return nil
}

// Update is used to update the butler configuration for the webserver.
// It takes a butler configuration as the argumnt. It then set s the
// Monitor.config with the argument. Finally "restarts" the webserver.
// This is realy for testing purposes only.
func (m *Monitor) Update(bc *config.ButlerConfig) {
	m.config = bc
	m.Stop()
	m.Start()
}

// Handler is the handler function for the /health-check monitor
// endpoint. It displays the JSON Marshal'd output of all the various
// configuration options that buter gets started with, and some run time
// information
func (m *Monitor) Handler(w http.ResponseWriter, r *http.Request) {
	mOut := Output{ConfigPath: m.config.GetPath(),
		ConfigScheme:     m.config.URL.Scheme,
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
