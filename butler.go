package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoisie/mustache"
	"github.com/jasonlvhit/gocron"
	"github.com/udhos/equalfile"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version                 = "v0.5.3"
	PrometheusConfig        = "prometheus.yml"
	AdditionalConfig        = "alerts/commonalerts.yml,alerts/tenant.yml"
	PrometheusRootDirectory = "/opt/prometheus"
	PrometheusHost          string
	ConfigUrl               string
	Files                   ConfigFiles
	LastRun                 time.Time
	HttpTimeout             int
	Subs                    MustacheSubs
	RequiredSubKeys         = []string{"ethos-cluster-id"}

	// Prometheus metrics
	ButlerConfigValid    *prometheus.GaugeVec
	ButlerContactSuccess *prometheus.GaugeVec
	ButlerContactTime    *prometheus.GaugeVec
	ButlerReloadSuccess  prometheus.Gauge
	ButlerReloadTime     prometheus.Gauge
	ButlerRenderSuccess  prometheus.Gauge
	ButlerRenderTime     prometheus.Gauge
	ButlerWriteSuccess   *prometheus.GaugeVec
	ButlerWriteTime      *prometheus.GaugeVec
)

// butlerHeader and butlerFooter represent the strings that need to be matched
// against in the configuration files. If these entries do not exist in the
// downloaded file, then we cannot be assured that these files are legitimate
// configurations.
const (
	butlerHeader = "#butlerstart"
	butlerFooter = "#butlerend"
)

// FAILURE and SUCCESS are float64 enumerations which are used to set the
// success or failure flags for the prometheus check gauges
//
// These need to be outside of the previous const due to them being an
// enumeration, and putting them in the previous const will mess up the
// ordering.
const (
	FAILURE float64 = 0 + iota
	SUCCESS
)

// ConfigFiles is the structure for holding the array of configuration
// files which are passed into butler from the CLI. It is also displayed
// as output to the monitor health check in json output.
type ConfigFiles struct {
	Files []string `json:"additional_config"`
}

// MustacheSubs is the structure for holding the key=value pairs of mustache
// substitutions which have to be handled within the prometheus.yml. It is also
// displayed as output to the monitor health check in json output.
type MustacheSubs struct {
	Subs map[string]string `json:"mustache_subs"`
}

// Monitor is the empty structure to be used for starting up the monitor
// health check and prometheus metrics http endpoints.
type Monitor struct {
}

// NewMonitor returns a Monitor structure which is used to bring up the
// monitor health check and prometheus metrics http endpoints.
func NewMonitor() *Monitor {
	return &Monitor{}
}

// MonitorOutput is the structure which holds the formatting which is output
// to the health check monitor. When /health-check is hit, it returns this
// structure, which is then Marshal'd to json and provided back to the end
// user
type MonitorOutput struct {
	ClusterID        string `json:"cluster_id"`
	ConfigURL        string `json:"config_url"`
	PrometheusHost   string `json:"prometheus_host"`
	PrometheusConfig string `json:"prometheus_config"`
	ConfigFiles
	MustacheSubs
	LastRun time.Time `json:"last_run"`
	Version string    `json:"version"`
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
	mOut := MonitorOutput{ClusterID: Subs.Subs["ethos-cluster-id"],
		ConfigURL:        ConfigUrl,
		PrometheusHost:   PrometheusHost,
		PrometheusConfig: PrometheusConfig,
		ConfigFiles:      Files,
		MustacheSubs:     Subs,
		LastRun:          LastRun,
		Version:          version}
	resp, err := json.Marshal(mOut)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Could not Marshal JSON, but I promise I'm up!")
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(resp))
}

// GetPrometheusPaths returns a slice/array of full paths to the prometheus
// configuration files. For example /opt/prometheus/promethes.yml versus
// just the filename which is passed by the command line.
func GetPrometheusPaths() []string {
	var paths []string
	for _, file := range Files.Files {
		path := fmt.Sprintf("%s/%s", PrometheusRootDirectory, file)
		paths = append(paths, path)
	}
	return paths
}

// GetPrometheusLabels returns a slice/array of only the filenames. This is for
// use with the prometheus monitors where we want to identify which files
// are being worked with for the metrics being exported to prometheus
func GetPrometheusLabels() []string {
	var labels []string
	for _, file := range Files.Files {
		label := path.Base(file)
		labels = append(labels, label)
	}
	return labels
}

// GetFloatTimeNow returns a float64 value of Unix time since the Epoch. This is
// typically in uint32 format; however, prometheus Gauge's require their input
// to be a float64
func GetFloatTimeNow() float64 {
	return float64(time.Now().Unix())
}

// GetPCMSUrls returns a slice/array of complete URLs to the locations where
// the butler managed configuration files need to be downloaded from.
func GetPCMSUrls() []string {
	var urls []string
	for _, file := range Files.Files {
		u := fmt.Sprintf("%s/%s", ConfigUrl, file)
		urls = append(urls, u)
	}
	return urls
}

// DownloadPCMSFile returns a pointer to an os.File object which is the result
// of creating a temporary file, and downloading a prometheus configuration
// file to it. If there is an error, nil is returned instead of the os.File
// pointer
func DownloadPCMSFile(u string) *os.File {
	tmpFile, err := ioutil.TempFile("/tmp", "pcmsfile")
	if err != nil {
		log.Fatal(err)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(HttpTimeout) * time.Second}

	response, err := httpClient.Get(u)

	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		log.Printf("Could not download from %s, err=%s\n", u, err.Error())
		tmpFile = nil
		return tmpFile
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		log.Printf("Did not receive 200 response code for %s. code=%d\n", u, response.StatusCode)
		tmpFile = nil
		return tmpFile
	}

	_, err = io.Copy(tmpFile, response.Body)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		log.Printf("Could not download from %s, err=%s\n", u, err.Error())
		tmpFile = nil
		return tmpFile
	}
	return tmpFile
}

// CopyFile copies the src path string to the dst path string. If there is an
// error, an error is returned, otherwise nil is returned.
func CopyFile(src string, dst string) error {
	var (
		err error
		in  *os.File
		out *os.File
	)

	// open source
	in, err = os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// open destination
	if _, err = os.Stat(dst); err != nil {
		out, err = os.Create(dst)
	} else {
		out, err = os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0644)
	}
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

// RenderPrometheusYaml takes a pointer to an os.File object. It reads the file
// attempts to parse the mustache
func RenderPrometheusYaml(f *os.File) error {
	tmpl, err := mustache.ParseFile(f.Name())
	if err != nil {
		return err
	}

	out := tmpl.Render(Subs.Subs)

	f, err = os.OpenFile(f.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(out))
	if err != nil {
		return err
	}
	return nil
}

func ValidateButlerConfig(f *os.File) error {
	var (
		configLine    string
		isFirstLine   bool
		isValidHeader bool
		isValidFooter bool
	)
	isFirstLine = true
	isValidHeader = true
	isValidFooter = true

	file, err := os.Open(f.Name())
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		configLine = scanner.Text()
		// Check that the header is valid
		if isFirstLine {
			if configLine != butlerHeader {
				isValidHeader = false
			}
			isFirstLine = false
		}
	}
	// Check that the footer is valid
	if configLine != butlerFooter {
		if configLine != butlerFooter {
			isValidFooter = false
		}
	}

	if !isValidHeader && !isValidFooter {
		return errors.New("Invalid butler header and footer")
	} else if !isValidHeader {
		return errors.New("Invalid butler header")
	} else if !isValidFooter {
		return errors.New("Invalid butler footer")
	} else {
		return nil
	}
}

func PCMSHandler() {
	IsModified := false
	log.Println("Processing PCMS Files.")

	// Check to see if the files currently exist. If the docker path is properly mounted from the prometheus
	// container, then we should see those files.  Error out if we cannot see those files.
	for _, file := range GetPrometheusPaths() {
		dir := filepath.Dir(file)
		if _, err := os.Stat(dir); err != nil {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				log.Fatalf(err.Error())
			}
			log.Printf("Created directory \"%s\"", dir)
		}
	}

	for i, u := range GetPCMSUrls() {
		f := DownloadPCMSFile(u)
		if f == nil {
			ButlerContactSuccess.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(FAILURE)
			continue
		} else {
			ButlerContactSuccess.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(SUCCESS)
			ButlerContactTime.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(GetFloatTimeNow())
		}

		// For the prometheus.yml we have to do some mustache replacement on downloaded file
		if GetPrometheusPaths()[i] == fmt.Sprintf("%s/%s", PrometheusRootDirectory, PrometheusConfig) {
			err := RenderPrometheusYaml(f)
			if err != nil {
				ButlerReloadSuccess.Set(FAILURE)
			} else {
				ButlerReloadSuccess.Set(SUCCESS)
				ButlerReloadTime.Set(GetFloatTimeNow())
			}
			// Going to need to rewrite the destination filename for the file comparison
			// Probably a better way to do this
			Files.Files[i] = fmt.Sprintf("prometheus.yml")
		}

		// Let's ensure that the files starts with #butlerstart and
		// ends with #butlerend. If they do not, then we will assume
		// we did not get a correct configuration, or there is an issue
		// with the upstream
		if err := ValidateButlerConfig(f); err != nil {
			log.Printf("%s for %s.\n", err.Error(), GetPrometheusPaths()[i])
			ButlerConfigValid.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(FAILURE)
			continue
		} else {
			ButlerConfigValid.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(SUCCESS)
		}

		// Let's compare the source and destination files
		cmp := equalfile.New(nil, equalfile.Options{})
		equal, err := cmp.CompareFile(f.Name(), GetPrometheusPaths()[i])
		if !equal {
			log.Printf("Found difference in \"%s.\"  Updating.", GetPrometheusPaths()[i])
			err = CopyFile(f.Name(), GetPrometheusPaths()[i])
			if err != nil {
				ButlerWriteSuccess.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(FAILURE)
				log.Printf(err.Error())
			}
			ButlerWriteSuccess.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(SUCCESS)
			ButlerWriteTime.With(prometheus.Labels{"config_file": GetPrometheusLabels()[i]}).Set(GetFloatTimeNow())

			IsModified = true
		}
		os.Remove(f.Name())

		if GetPrometheusPaths()[i] == fmt.Sprintf("%s/prometheus.yml", PrometheusRootDirectory) {
			// Now put things back to how they originally were...
			Files.Files[i] = PrometheusConfig
		}
	}

	if IsModified {
		log.Printf("Reloading prometheus.")
		// curl -v -X POST $HOST:9090/-/reload
		promUrl := fmt.Sprintf("http://%s:9090/-/reload", PrometheusHost)
		client := &http.Client{
			Timeout: time.Duration(HttpTimeout) * time.Second}
		req, err := http.NewRequest("POST", promUrl, nil)
		if err != nil {
			log.Printf(err.Error())
			ButlerReloadSuccess.Set(FAILURE)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf(err.Error())
			ButlerReloadSuccess.Set(FAILURE)
		} else {
			ButlerReloadSuccess.Set(SUCCESS)
			ButlerReloadTime.Set(GetFloatTimeNow())
			log.Printf("resp=%#v\n", resp)
		}
	} else {
		log.Printf("Found no differences in PCMS files.")
	}
	LastRun = time.Now()
}

func ParseConfigFiles(file *ConfigFiles, configFiles string) error {
	files := strings.Split(configFiles, ",")
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		file.Files = append(file.Files, f)
	}
	return nil
}

func ParseMustacheSubs(subs *MustacheSubs, configSubs string) error {
	pairs := strings.Split(configSubs, ",")
	for _, p := range pairs {
		p = strings.TrimSpace(p)
		keyvalpairs := strings.Split(p, "=")
		if len(keyvalpairs) != 2 {
			log.Printf("ParseMustacheSubs(): invalid key value pair \"%s\"... ignoring.", keyvalpairs)
			continue
		}
		key := strings.TrimSpace(keyvalpairs[0])
		val := strings.TrimSpace(keyvalpairs[1])
		subs.Subs[key] = val
	}
	// validate against RequiredSubKeys
	if !ValidateMustacheSubs(subs) {
		return errors.New(fmt.Sprintf("could not validate required mustache subs. check your config. required subs=%s.", RequiredSubKeys))
	}
	return nil
}

func ValidateMustacheSubs(subs *MustacheSubs) bool {
	var (
		subEntries map[string]bool
	)
	subEntries = make(map[string]bool)

	// set the default return value to false
	for _, vs := range RequiredSubKeys {
		subEntries[vs] = false
	}

	// range over the subs and see if the keys match the required list of substitution keys
	for k, _ := range subs.Subs {
		if _, ok := subEntries[k]; ok {
			subEntries[k] = true
		}
	}

	// If any of the sub keys are false, then something is missing
	for _, v := range subEntries {
		if v == false {
			return false
		}
	}
	return true
}

func main() {
	var (
		err                        error
		versionFlag                = flag.Bool("version", false, "Print version information.")
		configUrlFlag              = flag.String("config.url", "", "The base url to grab prometheus configuration files")
		configPrometheusConfigFlag = flag.String("config.prometheus-config", PrometheusConfig, "The prometheus configuration file.")
		configAdditionalConfigFlag = flag.String("config.additional-config", AdditionalConfig, "The prometheus configuration files to grab in comma separated format.")
		configSchedulerIntFlag     = flag.Int("config.scheduler-interval", 300, "The interval, in seconds, to run the scheduler.")
		configPrometheusHost       = flag.String("config.prometheus-host", os.Getenv("HOST"), "The prometheus host to reload.")
		configHttpTimeout          = flag.Int("config.http-timeout-host", 10, "The http timeout, in seconds, for GET requests to gather the configuration files")
		configMustacheSubs         = flag.String("config.mustache-subs", "", "prometheus.yml Mustache Substitutions.")
	)
	flag.Parse()

	log.Printf("Starting butler version %s\n", version)

	if *versionFlag {
		fmt.Fprintln(os.Stdout, version)
		os.Exit(0)
	}

	// Set the HTTP Timeout
	HttpTimeout = *configHttpTimeout

	// Grab the prometheus host
	if *configPrometheusHost == "" {
		log.Fatal("You must provide a -config.prometheus-host, or a HOST environment variable.")
	} else {
		PrometheusHost = *configPrometheusHost
	}

	if *configUrlFlag == "" {
		log.Fatal("You must provide a -config.url")
	} else {
		ConfigUrl = *configUrlFlag
	}

	if *configMustacheSubs == "" {
		log.Fatal("You must provide a -config.mustache-subs")
	} else {
		Subs.Subs = make(map[string]string)
		err := ParseMustacheSubs(&Subs, *configMustacheSubs)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	if *configPrometheusConfigFlag != "" {
		PrometheusConfig = *configPrometheusConfigFlag
	}

	ParseConfigFiles(&Files, *configPrometheusConfigFlag)
	ParseConfigFiles(&Files, *configAdditionalConfigFlag)

	if _, err = url.ParseRequestURI(ConfigUrl); err != nil {
		log.Fatalf("Cannot parse ConfigUrl=%s", ConfigUrl)
	}

	// Check that we can connect to the url
	if _, err = http.Get(ConfigUrl); err != nil {
		log.Fatalf("Cannot connect to \"%s\", err=%s", ConfigUrl, err.Error())
	}

	// Setup the prometheus metric information
	ButlerConfigValid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_config_valid",
		Help: "Is the butler configuration valid",
	}, []string{"config_file"})

	ButlerContactSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_contact_success",
		Help: "Did butler succesfully contact the remote repository",
	}, []string{"config_file"})

	ButlerContactTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_contact_time",
		Help: "Time that butler succesfully contacted the remote repository",
	}, []string{"config_file"})

	ButlerReloadSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "butler_localconfig_reload_success",
		Help: "Did butler successfully reload prometheus",
	})
	// Set to successful initially
	ButlerReloadSuccess.Set(SUCCESS)

	ButlerReloadTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "butler_localconfig_reload_time",
		Help: "Time that butler successfully reload prometheus",
	})
	// Set the initial time to now
	ButlerReloadTime.Set(GetFloatTimeNow())

	ButlerRenderSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "butler_localconfig_render_success",
		Help: "Did butler successfully render the prometheus.yml",
	})
	// Set to successful initially
	ButlerRenderSuccess.Set(SUCCESS)

	ButlerRenderTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "butler_localconfig_render_time",
		Help: "Time that butler successfully rendered the prometheus.yml",
	})
	// Set the initial time to now
	ButlerRenderTime.Set(GetFloatTimeNow())

	ButlerWriteSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_write_success",
		Help: "Did butler successfully write the configuration",
	}, []string{"config_file"})

	ButlerWriteTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_write_time",
		Help: "Time that butler successfully write the configuration",
	}, []string{"config_file"})

	prometheus.MustRegister(ButlerConfigValid)
	prometheus.MustRegister(ButlerContactSuccess)
	prometheus.MustRegister(ButlerContactTime)
	prometheus.MustRegister(ButlerReloadSuccess)
	prometheus.MustRegister(ButlerReloadTime)
	prometheus.MustRegister(ButlerRenderSuccess)
	prometheus.MustRegister(ButlerRenderTime)
	prometheus.MustRegister(ButlerWriteSuccess)
	prometheus.MustRegister(ButlerWriteTime)

	// Start up the monitor web server
	monitor := NewMonitor()
	monitor.Start()

	// Do one run of PCMSHandler() then start off the scheduler
	PCMSHandler()

	sched := gocron.NewScheduler()
	sched.Every(uint64(*configSchedulerIntFlag)).Seconds().Do(PCMSHandler)
	<-sched.Start()
}
