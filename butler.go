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
	"path/filepath"
	"strings"
	"time"

	"github.com/hoisie/mustache"
	"github.com/jasonlvhit/gocron"
	"github.com/udhos/equalfile"
)

var (
	version                 = "v0.4.0"
	PrometheusConfig        = "prometheus.yml"
	AdditionalConfig        = "alerts/commonalerts.yml,alerts/tenant.yml"
	PrometheusRootDirectory = "/opt/prometheus"
	PrometheusHost          string
	ClusterId               string
	ConfigUrl               string
	Files                   ConfigFiles
	LastRun                 time.Time
	HttpTimeout             int
)

const (
	butlerHeader = "#butlerstart"
	butlerFooter = "#butlerend"
)

type ConfigFiles struct {
	Files []string `json:"additional_config"`
}

type Monitor struct {
}

type MonitorOutput struct {
	ClusterID        string `json:"cluster_id"`
	ConfigURL        string `json:"config_url"`
	PrometheusHost   string `json:"prometheus_host"`
	PrometheusConfig string `json:"prometheus_config"`
	ConfigFiles
	LastRun time.Time `json:"last_run"`
	Version string    `json:"version"`
}

func (m *Monitor) Start() {
	http.HandleFunc("/health-check", m.MonitorHandler)
	server := &http.Server{}
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error creating listener: %s", err.Error())
	}
	go server.Serve(listener)
}

func (m *Monitor) MonitorHandler(w http.ResponseWriter, r *http.Request) {
	mOut := MonitorOutput{ClusterID: ClusterId,
		ConfigURL:        ConfigUrl,
		PrometheusHost:   PrometheusHost,
		PrometheusConfig: PrometheusConfig,
		ConfigFiles:      Files,
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

func GetPrometheusPaths() []string {
	var paths []string
	for _, file := range Files.Files {
		path := fmt.Sprintf("%s/%s", PrometheusRootDirectory, file)
		paths = append(paths, path)
	}
	return paths
}

func GetPCMSUrls() []string {
	var urls []string
	for _, file := range Files.Files {
		u := fmt.Sprintf("%s/%s", ConfigUrl, file)
		urls = append(urls, u)
	}
	return urls
}

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

func RenderPrometheusYaml(f *os.File) {
	out := mustache.RenderFile(f.Name(), map[string]string{"ethos-cluster-id": ClusterId})
	f, err := os.OpenFile(f.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()
	_, err = f.Write([]byte(out))
	if err != nil {
		log.Fatal(err.Error())
	}
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
			continue
		}

		// For the prometheus.yml we have to do some mustache
		// cluster-id replacement on downloaded file
		if GetPrometheusPaths()[i] == fmt.Sprintf("%s/%s", PrometheusRootDirectory, PrometheusConfig) {
			RenderPrometheusYaml(f)
			// Going to need to rewrite the destination filename for the file comparison
			// Probably a better way to do this
			Files.Files[i] = fmt.Sprintf("prometheus.yml")
		}

		// Let's ensure that the files starts with #butlerstart and
		// ends with #butlerend. If they do not, then we will assume
		// we did not get a correct configuration, or there is an issue
		// with the upstream
		err := ValidateButlerConfig(f)
		if err != nil {
			log.Printf("%s for %s.\n", err.Error(), GetPrometheusPaths()[i])
			continue
		}

		// Let's compare the source and destination files
		cmp := equalfile.New(nil, equalfile.Options{})
		equal, err := cmp.CompareFile(f.Name(), GetPrometheusPaths()[i])
		if !equal {
			log.Printf("Found difference in \"%s.\"  Updating.", GetPrometheusPaths()[i])
			err = CopyFile(f.Name(), GetPrometheusPaths()[i])
			if err != nil {
				log.Fatal(err.Error())
			}

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
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf(err.Error())
		} else {
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

func NewMonitor() *Monitor {
	return &Monitor{}
}

func main() {
	var (
		err                        error
		versionFlag                = flag.Bool("version", false, "Print version information.")
		configUrlFlag              = flag.String("config.url", "", "The base url to grab prometheus configuration files")
		configClusterIdFlag        = flag.String("config.cluster-id", "", "The ethos cluster identifier.")
		configPrometheusConfigFlag = flag.String("config.prometheus-config", PrometheusConfig, "The prometheus configuration file.")
		configAdditionalConfigFlag = flag.String("config.additional-config", AdditionalConfig, "The prometheus configuration files to grab in comma separated format.")
		configSchedulerIntFlag     = flag.Int("config.scheduler-interval", 300, "The interval, in seconds, to run the scheduler.")
		configPrometheusHost       = flag.String("config.prometheus-host", os.Getenv("HOST"), "The prometheus host to reload.")
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

	if *configClusterIdFlag == "" {
		log.Fatal("You must provide a -config.cluster-id")
	} else {
		ClusterId = *configClusterIdFlag
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

	monitor := NewMonitor()
	monitor.Start()

	// Do one run of PCMSHandler() then start off the scheduler
	PCMSHandler()

	sched := gocron.NewScheduler()
	sched.Every(uint64(*configSchedulerIntFlag)).Seconds().Do(PCMSHandler)
	<-sched.Start()
}
