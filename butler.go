package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"git.corp.adobe.com/TechOps-IAO/butler/config"

	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version                 = "v1.0.0"
	PrometheusConfig        = "prometheus.yml"
	PrometheusConfigStatic  = "prometheus.yml"
	AdditionalConfig        = "alerts/commonalerts.yml,alerts/tenant.yml"
	PrometheusRootDirectory = "/opt/prometheus"
	PrometheusHost          string
	ButlerConfigInterval    int
	ButlerConfigUrl         string
	ConfigCache             map[string][]byte
	AllConfigFiles          []string
	PrometheusConfigFiles   []string
	AdditionalConfigFiles   []string
	MustacheSubs            map[string]string
	HttpTimeout             int
	HttpRetries             int
	HttpRetryWaitMin        int
	HttpRetryWaitMax        int
)

// Monitor is the empty structure to be used for starting up the monitor
// health check and prometheus metrics http endpoints.
type Monitor struct {
	Config *config.ButlerConfig
}

// NewMonitor returns a Monitor structure which is used to bring up the
// monitor health check and prometheus metrics http endpoints.
func NewMonitor(bc *config.ButlerConfig) *Monitor {
	return &Monitor{Config: bc}
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
	http.HandleFunc("/health-check", m.MonitorHandler)
	http.Handle("/metrics", promhttp.Handler())
	server := &http.Server{}
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error creating listener: %s", err.Error())
	}
	go server.Serve(listener)
}

// MonitorHandler is the handler function for the /health-check monitor
// endpoint. It displays the JSON Marshal'd output of all the various
// configuration options that buter gets started with, and some run time
// information
func (m *Monitor) MonitorHandler(w http.ResponseWriter, r *http.Request) {
	mOut := MonitorOutput{ConfigPath: m.Config.GetPath(),
		ConfigScheme:     m.Config.Scheme,
		RetrieveInterval: m.Config.Interval,
		LogLevel:         m.Config.GetLogLevel(),
		ConfigSettings:   *m.Config.Config,
		Version:          version}
	resp, err := json.Marshal(mOut)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Could not Marshal JSON, but I promise I'm up!")
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(resp))
}

func SetLogLevel(l string) log.Level {
	switch strings.ToLower(l) {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	case "panic":
		return log.PanicLevel
	default:
		log.Warn(fmt.Sprintf("Unknown log level \"%s\". Defaulting to %s", l, log.InfoLevel))
		return log.InfoLevel
	}
}

func main() {
	var (
		err                    error
		versionFlag            = flag.Bool("version", false, "Print version information.")
		configPath             = flag.String("config.path", "", "Full remote path to butler configuration file (eg: URI without scheme://).")
		configScheme           = flag.String("config.scheme", "http", "Scheme used to download the butler configuration file. Currently supported schemes: http, https.")
		configInterval         = flag.Int("config.retrieve-interval", 300, "The interval, in seconds, to retrieve new butler configuration files.")
		configHttpTimeout      = flag.Int("http.timeout", 10, "The http timeout, in seconds, for GET requests to obtain the butler configuration file.")
		configHttpRetries      = flag.Int("http.retries", 4, "The number of http retries for GET requests to obtain the butler configuration files")
		configHttpRetryWaitMin = flag.Int("http.retry_wait_min", 5, "The minimum amount of time to wait before attemping to retry the http config get operation.")
		configHttpRetryWaitMax = flag.Int("http.retry_wait_max", 10, "The maximum amount of time to wait before attemping to retry the http config get operation.")
		configLogLevel         = flag.String("log.level", "info", "The butler log level. Log levels are: debug, info, warn, error, fatal, panic.")
	)
	flag.Parse()
	log.SetLevel(SetLogLevel(*configLogLevel))
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	if *versionFlag {
		fmt.Fprintf(os.Stdout, "butler %s\n", version)
		os.Exit(0)
	}

	if *configPath == "" {
		log.Fatal("You must provide a -config.path for a path to the butler configuration.")
	}

	log.Infof("Starting butler version %s", version)

	bc := config.NewButlerConfig()
	bc.SetLogLevel(SetLogLevel(*configLogLevel))
	bc.SetScheme(*configScheme)
	bc.SetPath(*configPath)

	// Set the HTTP Timeout
	log.Debugf("main(): setting HttpTimeout to %d", *configHttpTimeout)
	bc.SetTimeout(*configHttpTimeout)

	// Set the HTTP Retries Counter
	log.Debugf("main(): setting HttpRetries to %d", *configHttpRetries)
	bc.SetRetries(*configHttpRetries)

	// Set the HTTP Holdoff Values
	log.Debugf("main(): setting RetryWaitMin[%d] and RetryWaitMax[%d]", *configHttpRetryWaitMin, *configHttpRetryWaitMax)
	bc.SetRetryWaitMin(*configHttpRetryWaitMin)
	bc.SetRetryWaitMax(*configHttpRetryWaitMax)

	// Set the butler configuration retrieval interval
	log.Debugf("main(): setting ConfigInterval to %d", *configInterval)
	bc.SetInterval(*configInterval)

	if err = bc.Init(); err != nil {
		log.Fatalf("Cannot initialize butler config. err=%s", err.Error())
	}

	// Start up the monitor web server
	monitor := NewMonitor(bc)
	monitor.Start()

	// Do initial grab of butler configuration file.
	// Going to do this in an endless loop until we initially
	// grab a configuration file.
	for {
		log.Debugf("main(): running first bc.Handler()")
		err = bc.Handler()
		if err != nil {
			log.Infof("Cannot retrieve butler configuration. err=%s", err.Error())
			log.Infof("Sleeping 5 seconds.")
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	sched := gocron.NewScheduler()
	log.Debugf("main(): starting scheduler...")

	log.Debugf("main(): running butler configuration scheduler every %d seconds", bc.GetInterval())
	sched.Every(uint64(bc.GetInterval())).Seconds().Do(bc.Handler)

	log.Debugf("main(): giving scheduler to butler.")
	bc.SetScheduler(sched)

	log.Debugf("main(): doing initial run of butler configuration management handler")
	bc.RunCMHandler()

	<-sched.Start()
}
