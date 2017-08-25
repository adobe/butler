package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	//"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hoisie/mustache"
	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
	"github.com/udhos/equalfile"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version                 = "v0.6.7"
	PrometheusConfig        = "prometheus.yml"
	PrometheusConfigStatic  = "prometheus.yml"
	AdditionalConfig        = "alerts/commonalerts.yml,alerts/tenant.yml"
	PrometheusRootDirectory = "/opt/prometheus"
	PrometheusHost          string
	ConfigUrl               string
	ConfigCache             map[string][]byte
	AllConfigFiles          []string
	PrometheusConfigFiles   []string
	AdditionalConfigFiles   []string
	MustacheSubs            map[string]string
	LastRun                 time.Time
	HttpTimeout             int
	HttpRetries             int
	HttpRetryWaitMin        = 50
	HttpRetryWaitMax        = 75
	RequiredSubKeys         = []string{"ethos-cluster-id"}
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
	ClusterID             string            `json:"cluster_id"`
	ConfigURL             string            `json:"config_url"`
	PrometheusHost        string            `json:"prometheus_host"`
	PrometheusConfigFiles []string          `json:"prometheus_config_files"`
	AdditionalConfigFiles []string          `json:"additional_config_files"`
	MustacheSubs          map[string]string `json:"mustache_subs"`
	LastRun               time.Time         `json:"last_run"`
	Version               string            `json:"version"`
}

type PrometheusFileMap struct {
	TmpFile string
	Success bool
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
	mOut := MonitorOutput{ClusterID: MustacheSubs["ethos-cluster-id"],
		ConfigURL:             ConfigUrl,
		PrometheusHost:        PrometheusHost,
		PrometheusConfigFiles: PrometheusConfigFiles,
		AdditionalConfigFiles: AdditionalConfigFiles,
		MustacheSubs:          MustacheSubs,
		LastRun:               LastRun,
		Version:               version}
	resp, err := json.Marshal(mOut)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Could not Marshal JSON, but I promise I'm up!")
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(resp))
}

// GetPrometheusPaths returns a slice/array of full paths to the prometheus
// configuration files. For example /opt/prometheus/prometheus.yml versus
// just the filename which is passed by the command line.
func GetPrometheusPaths(entries []string) []string {
	var paths []string
	for _, file := range entries {
		path := fmt.Sprintf("%s/%s", PrometheusRootDirectory, file)
		paths = append(paths, path)
	}
	return paths
}

// GetPrometheusPath returns a string containing the full path to the prometheus
// configuration file. For example if file is passed in from the command line
// is prometheus.yml, the response could be /opt/prometheus/prometheus.yml
func GetPrometheusPath(file string) string {
	for {
		if strings.HasPrefix(file, "/") {
			file = TrimPrefix(file, "/")
		} else {
			break
		}
	}
	return fmt.Sprintf("%s/%s", PrometheusRootDirectory, file)
}

// GetPrometheusLabels returns a slice/array of only the filenames. This is for
// use with the prometheus monitors where we want to identify which files
// are being worked with for the metrics being exported to prometheus
func GetPrometheusLabels(entries []string) []string {
	var labels []string
	for _, file := range entries {
		label := path.Base(file)
		labels = append(labels, label)
	}
	return labels
}

// GetPrometheusLabel returns a string containing only the filename, without
// path information. This is for use with prometheus monitors where we want to
// identify which files are being worked with for mthe metrics being exported
// to prometheus
func GetPrometheusLabel(entry string) string {
	return path.Base(entry)
}

// TrimSuffix returns a sub string of the string provided, as the first argument,
// with the suffix, second argument, removed from the beginning, if the string ends
// with that suffix.
func TrimSuffix(s string, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

// TrimPrefix returns a sub string of the string provided, as the first argument,
// with the prefix, second argument, removed from the beginning, if the string begins
// with that prefix.
func TrimPrefix(s string, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		s = s[len(prefix):]
	}
	return s
}

// GetPCMSUrls returns a slice/array of complete URLs to the locations where
// the butler managed configuration files need to be downloaded from.
func GetPCMSUrls(entries []string) []string {
	var urls []string
	for _, File := range entries {
		// Let's santize the input a little bit and strip off trailing and prefixed
		// slashes "/" from the ConfigUrl and the File
		for {
			if strings.HasSuffix(ConfigUrl, "/") {
				ConfigUrl = TrimSuffix(ConfigUrl, "/")
			} else {
				break
			}
		}
		for {
			if strings.HasPrefix(File, "/") {
				File = TrimPrefix(File, "/")
			} else {
				break
			}
		}
		u := fmt.Sprintf("%s/%s", ConfigUrl, File)
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

	httpClient := retryablehttp.NewClient()
	httpClient.HTTPClient.Timeout = time.Duration(HttpTimeout) * time.Second
	httpClient.RetryMax = HttpRetries
	httpClient.RetryWaitMin = time.Duration(HttpRetryWaitMin) * time.Millisecond
	httpClient.RetryWaitMax = time.Duration(HttpRetryWaitMax) * time.Millisecond
	// I really don't care about any of the debug output that comes
	// from retryablehttp output
	httpClient.Logger.SetFlags(0)
	httpClient.Logger.SetOutput(ioutil.Discard)

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
		log.Printf("Did not receive 200 response code for %s. code=%s\n", u, response.StatusCode)
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

// CacheConfigs stores the currently known good prometheus configurations
// into memory
func CacheConfigs() {
	log.Printf("Storing known good Prometheus configurations to cache.\n")
	ConfigCache = make(map[string][]byte)
	for _, file := range AllConfigFiles {
		out, err := ioutil.ReadFile(GetPrometheusPath(file))
		if err != nil {
			log.Printf("Could not store %s to cache. err=%s.\n", GetPrometheusPath(file), err.Error())
		} else {
			ConfigCache[GetPrometheusPath(file)] = out
		}
	}
	log.Printf("Done storing known good Prometheus configurations to cache.\n")
}

// RestoreCachedConfigs restores the currently cached prometheus configurations
// back to disk
func RestoreCachedConfigs() {
	// If we do not have a good configuration cache, then there's nothing for us to do.
	if ConfigCache == nil {
		log.Printf("No current known good Prometheus configurations in cache. Cleaning configuration...\n")
		for _, file := range AllConfigFiles {
			fileName := GetPrometheusPath(file)
			log.Printf("Removing bad Prometheus configuration file %s.", fileName)
			os.Remove(fileName)
		}
		log.Printf("Done cleaning broken configuration. Returning...")
		SetButlerKnownGoodRestoredVal(FAILURE)
		return
	}

	log.Printf("Restoring known good Prometheus configurations from cache.\n")
	for _, file := range AllConfigFiles {
		fileName := GetPrometheusPath(file)
		fileData := ConfigCache[fileName]

		f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Could not open %s for writing! err=%s.\n", fileName, err.Error())
			continue
		} else {
			count, err := f.Write(fileData)
			if err != nil {
				log.Printf("Could write to %s! err=%s.\n", fileName, err.Error())
				continue
			} else {
				f.Close()
				log.Printf("Wrote %d bytes for %s.\n", count, fileName)
			}
		}
	}
	log.Printf("Done restoring known good Prometheus configurations from cache.\n")
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

	out := tmpl.Render(MustacheSubs)

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

// ValidateButlerConfig takes a pointer to an os.File object. It scans over the
// file and ensures that it begins with the proper header, and ends with the
// proper footer. If it does not begin or end with the proper header/footer,
// then an error is returned. If the file passes the checks, a nil is returned.
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

// CheckPaths checks takes a slice of full paths to a file, and checks to see
// if the underlying directory exists. If the path does not exist, it will
// create a new directory.
func CheckPaths(Files []string) bool {
	// Check to see if the files currently exist. If the docker path is properly mounted from the prometheus
	// container, then we should see those files.  Error out if we cannot see those files.
	for _, file := range GetPrometheusPaths(Files) {
		dir := filepath.Dir(file)
		if _, err := os.Stat(dir); err != nil {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				log.Fatalf(err.Error())
			}
			log.Printf("Created directory \"%s\"", dir)
		}
	}
	// Check to see if there are any additional files that need to be cleaned up
	// that aren't part of our current file list.
	err := filepath.Walk(PrometheusRootDirectory, PathCleanup)
	// If there is an error, then true signifies that we need to reload the prometheus config
	// otherwise false means that there have been no changes to the path structure (eg: removal
	// of unneeded config files)
	if err != nil {
		return true
	} else {
		return false
	}
}

// PathCleanup
func PathCleanup(path string, f os.FileInfo, err error) error {
	log.Debugf("PathCleanup(): entering")
	var (
		Found bool
	)
	Found = false

	// We don't have to do anything with a directory
	if f.Mode().IsDir() {
		log.Debugf("PathCleanup(): %s is a directory... returning nil", f.Name())
		return nil
	}

	for _, file := range AllConfigFiles {
		if path == GetPrometheusPath(file) {
			Found = true
		}
	}

	if !Found {
		message := fmt.Sprintf("Found unknown file \"%s\". deleting...", path)
		log.Debugf("PathCleanup(): Found unknown file \"%s\". deleting...", path)
		os.Remove(path)
		return errors.New(message)
	}
	return nil
}

func ProcessAdditionalConfigFiles(Files []string, c chan bool) {
	var (
		ModifiedFileMap map[string]bool
	)

	IsModified := false
	ModifiedFileMap = make(map[string]bool)

	// Process the prometheus.yml configuration files
	for i, u := range GetPCMSUrls(Files) {
		// Grab the remote file into a local temp file
		f := DownloadPCMSFile(u)
		if f == nil {
			SetButlerContactVal(FAILURE, GetPrometheusLabels(Files)[i])
			continue
		} else {
			SetButlerContactVal(SUCCESS, GetPrometheusLabels(Files)[i])
		}

		// Let's ensure that the files starts with #butlerstart and
		// ends with #butlerend. If they do not, then we will assume
		// we did not get a correct configuration, or there is an issue
		// with the upstream
		if err := ValidateButlerConfig(f); err != nil {
			log.Printf("%s for %s.\n", err.Error(), GetPrometheusPaths(Files)[i])
			SetButlerConfigVal(FAILURE, GetPrometheusLabels(Files)[i])
			continue
		} else {
			SetButlerConfigVal(SUCCESS, GetPrometheusLabels(Files)[i])
		}

		ModifiedFileMap[f.Name()] = CompareAndCopy(f.Name(), GetPrometheusPaths(Files)[i])

		// Clean up the temp file
		os.Remove(f.Name())
	}

	// Check for file modification differences
	for _, v := range ModifiedFileMap {
		if v {
			IsModified = true
		}
	}

	// Update the channel
	c <- IsModified
}

// ProcessPrometheusConfigFiles
func ProcessPrometheusConfigFiles(Files []string, c chan bool) {
	var (
		TmpFiles     []string
		LegitFileMap map[string]PrometheusFileMap
		IsModified   bool
		RenderFile   bool
	)

	IsModified = false
	RenderFile = true
	LegitFileMap = make(map[string]PrometheusFileMap)

	// Create a temporary file for the merged prometheus configurations
	TmpMergedFile, err := ioutil.TempFile("/tmp", "pcmsfile")
	if err != nil {
		log.Fatal(err)
	}

	// Process the prometheus.yml configuration files
	for i, u := range GetPCMSUrls(Files) {
		FileMap := PrometheusFileMap{}

		// Grab the remote file into a local temp file
		f := DownloadPCMSFile(u)
		if f == nil {
			SetButlerContactVal(FAILURE, GetPrometheusLabels(Files)[i])
			FileMap.Success = false
			RenderFile = false
			continue
		} else {
			SetButlerContactVal(SUCCESS, GetPrometheusLabels(Files)[i])
			FileMap.Success = true
		}

		FileMap.TmpFile = f.Name()

		// Let's ensure that the files starts with #butlerstart and
		// ends with #butlerend. If they do not, then we will assume
		// we did not get a correct configuration, or there is an issue
		// with the upstream
		if err := ValidateButlerConfig(f); err != nil {
			log.Printf("%s for %s.\n", err.Error(), GetPrometheusPaths(Files)[i])
			SetButlerConfigVal(FAILURE, GetPrometheusLabels(Files)[i])
			RenderFile = false
			FileMap.Success = false
			continue
		} else {
			SetButlerConfigVal(SUCCESS, GetPrometheusLabels(Files)[i])
			FileMap.Success = true
		}

		// For the prometheus.yml we have to do some mustache replacement on downloaded file
		err := RenderPrometheusYaml(f)
		if err != nil {
			log.Printf("%s for %s.\n", err.Error(), GetPrometheusPaths(Files)[i])
			SetButlerRenderVal(FAILURE)
			SetButlerConfigVal(FAILURE, GetPrometheusLabels(Files)[i])
			RenderFile = false
			FileMap.Success = false
			continue
		} else {
			SetButlerRenderVal(SUCCESS)
			SetButlerConfigVal(SUCCESS, GetPrometheusLabels(Files)[i])
			FileMap.Success = true
		}

		// going to want to keep tabs on TmpFiles, and remove all of them at the end.
		// remember that we want to merge all the downloaded files, so why remove them right now
		TmpFiles = append(TmpFiles, f.Name())
		LegitFileMap[GetPrometheusLabels(Files)[i]] = FileMap
	}

	// Need to verify whether or not we got all the prometheus configuration
	// files. If not, then we should not try to process them.
	for _, v := range LegitFileMap {
		if !v.Success {
			RenderFile = false
		}
	}

	// Let's process and merge the prometheus files
	if RenderFile {
		out, err := os.OpenFile(TmpMergedFile.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Could not process and merge new %s err=%s.", PrometheusConfigStatic, err.Error())
			SetButlerConfigVal(FAILURE, PrometheusConfigStatic)
			// just giving up at this point
			// Clean up the temporary files
			for _, file := range TmpFiles {
				os.Remove(file)
			}
			// Clean up Prometheus temp file
			os.Remove(TmpMergedFile.Name())
			c <- false
			return
		} else {
			for i, _ := range GetPCMSUrls(Files) {
				file := GetPrometheusLabels(Files)[i]
				in, err := os.Open(LegitFileMap[file].TmpFile)
				if err != nil {
					log.Printf("Could not process and merge new %s err=%s.", PrometheusConfigStatic, err.Error())
					SetButlerConfigVal(FAILURE, GetPrometheusLabels(Files)[i])
					// just giving up at this point, as well...
					// Clean up the temporary files
					for _, file := range TmpFiles {
						os.Remove(file)
					}
					// Clean up Prometheus temp file
					os.Remove(TmpMergedFile.Name())
					c <- false
					return
				}
				_, err = io.Copy(out, in)
				if err != nil {
					log.Printf("Could not process and merge new %s err=%s.", PrometheusConfigStatic, err.Error())
					SetButlerConfigVal(FAILURE, GetPrometheusLabels(Files)[i])
					// just giving up at this point, again...
					// Clean up the temporary files
					for _, file := range TmpFiles {
						os.Remove(file)
					}
					// Clean up Prometheus temp file
					os.Remove(TmpMergedFile.Name())
					c <- false
					return
				}
				in.Close()
			}
		}
		out.Close()
		promFile := fmt.Sprintf("%s/%s", PrometheusRootDirectory, PrometheusConfigStatic)
		IsModified = CompareAndCopy(TmpMergedFile.Name(), promFile)
	} else {
		IsModified = false
	}

	// Clean up the temporary files
	for _, file := range TmpFiles {
		os.Remove(file)
	}

	// Clean up Prometheus temp file
	os.Remove(TmpMergedFile.Name())

	// Update the channel
	c <- IsModified
}

func CompareAndCopy(source string, dest string) bool {
	// Let's compare the source and destination files
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(source, dest)
	if !equal {
		log.Printf("Found difference in \"%s.\"  Updating.", dest)
		err = CopyFile(source, dest)
		if err != nil {
			SetButlerWriteVal(FAILURE, GetPrometheusLabel(dest))
			log.Printf(err.Error())
		}
		SetButlerWriteVal(SUCCESS, GetPrometheusLabel(dest))
		return true
	} else {
		return false
	}
}

func PCMSHandler() {
	c := make(chan bool)
	log.Println("Processing PCMS Files.")

	checkPathModified := CheckPaths(AllConfigFiles)

	go ProcessPrometheusConfigFiles(PrometheusConfigFiles, c)
	go ProcessAdditionalConfigFiles(AdditionalConfigFiles, c)

	promModified, additionalModified := <-c, <-c

	log.Debugf("PCMSHandler(): checkPathModified=%#v", checkPathModified)
	log.Debugf("PCMSHandler(): promModified=%#v", promModified)
	log.Debugf("PCMSHandler(): additionalModified=%#v", additionalModified)

	if checkPathModified || promModified || additionalModified {
		log.Debugf("PCMSHandler(): going to reload prometheus")
		err := ReloadPrometheusHandler()
		log.Debugf("PCMSHandler(): reloaded prometheus. err=%#v", err)
	} else {
		log.Printf("Found no differences in PCMS files.")
	}
	LastRun = time.Now()
}

// PrometheusReloadRetryPolicy overrides go-retryablehttp's DefaultRetryPolicy
// for how it handles retrying http connections. By default if it receives a
// 50X response from the server, it'll retry. We do not want to do that with
// prometheus.
func PrometheusReloadRetryPolicy(resp *http.Response, err error) (bool, error) {
	if err != nil {
		return true, err
	}

	// Here is our policy override. By default it looks for
	// resp.StatusCode >= 500 ...
	if resp.StatusCode == 0 || resp.StatusCode >= 600 {
		return true, nil
	}
	return false, nil
}

func ReloadPrometheusHandler() error {
	var err error
	log.Printf("Reloading prometheus.")
	// curl -v -X POST $HOST:9090/-/reload
	promUrl := fmt.Sprintf("http://%s:9090/-/reload", PrometheusHost)

	client := retryablehttp.NewClient()
	client.CheckRetry = PrometheusReloadRetryPolicy
	client.HTTPClient.Timeout = time.Duration(HttpTimeout) * time.Second
	client.RetryMax = HttpRetries
	client.RetryWaitMin = time.Duration(HttpRetryWaitMin) * time.Millisecond
	client.RetryWaitMax = time.Duration(HttpRetryWaitMax) * time.Millisecond
	// I really don't care about any of the debug output that comes
	// from retryablehttp output
	client.Logger.SetFlags(0)
	client.Logger.SetOutput(ioutil.Discard)

	resp, err := client.Post(promUrl, "application/json", strings.NewReader(`{}`))
	if err != nil {
		log.Printf(err.Error())
		SetButlerReloadVal(FAILURE)
		return err
	}

	if resp.StatusCode == 200 {
		log.Printf("Successfully reloaded prometheus config. http_code=%d.\n", int(resp.StatusCode))
		SetButlerKnownGoodCachedVal(SUCCESS)
		SetButlerKnownGoodRestoredVal(FAILURE)
		SetButlerReloadVal(SUCCESS)
		CacheConfigs()
	} else {
		log.Printf("Received bad response from prometheus server. reverting to last known good config. http_code=%d.\n", int(resp.StatusCode))
		SetButlerKnownGoodCachedVal(FAILURE)
		SetButlerKnownGoodRestoredVal(FAILURE)
		SetButlerReloadVal(FAILURE)
		RestoreCachedConfigs()
	}

	return nil
}

func ParseConfigFiles(configFiles string) []string {
	var FileList []string
	files := strings.Split(configFiles, ",")
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		FileList = append(FileList, f)
	}
	return FileList
}

func ParseMustacheSubs(Subs map[string]string, configSubs string) error {
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
		Subs[key] = val
	}
	// validate against RequiredSubKeys
	if !ValidateMustacheSubs(Subs) {
		return errors.New(fmt.Sprintf("could not validate required mustache subs. check your config. required subs=%s.", RequiredSubKeys))
	}
	return nil
}

func ValidateMustacheSubs(Subs map[string]string) bool {
	var (
		subEntries map[string]bool
	)
	subEntries = make(map[string]bool)

	// set the default return value to false
	for _, vs := range RequiredSubKeys {
		subEntries[vs] = false
	}

	// range over the subs and see if the keys match the required list of substitution keys
	for k, _ := range Subs {
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
		err                        error
		versionFlag                = flag.Bool("version", false, "Print version information.")
		configUrlFlag              = flag.String("config.url", "", "The base url to grab prometheus configuration files")
		configPrometheusConfigFlag = flag.String("config.prometheus-config", PrometheusConfig, "The prometheus configuration file.")
		configAdditionalConfigFlag = flag.String("config.additional-config", AdditionalConfig, "The prometheus configuration files to grab in comma separated format.")
		configSchedulerIntFlag     = flag.Int("config.scheduler-interval", 300, "The interval, in seconds, to run the scheduler.")
		configPrometheusHost       = flag.String("config.prometheus-host", os.Getenv("HOST"), "The prometheus host to reload.")
		configHttpTimeout          = flag.Int("config.http-timeout-host", 10, "The http timeout, in seconds, for GET requests to gather the configuration files")
		configHttpRetries          = flag.Int("config.http-retries-host", 4, "The number of http retries for GET requests to gather the configuration files")
		configMustacheSubs         = flag.String("config.mustache-subs", "", "prometheus.yml Mustache Substitutions.")
		configLogLevel             = flag.String("log.level", "info", "The butler log level. Log levels are: debug, info, warn, error, fatal, panic.")
	)
	flag.Parse()
	log.SetLevel(SetLogLevel(*configLogLevel))
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	if *versionFlag {
		fmt.Fprintf(os.Stdout, "butler %s\n", version)
		os.Exit(0)
	}

	log.Infof("Starting butler version %s", version)

	// Set the HTTP Timeout
	log.Debugf("main(): setting HttpTimeout to %d", *configHttpTimeout)
	HttpTimeout = *configHttpTimeout

	// Set the HTTP Retries Counter
	log.Debugf("main(): setting HttpRetries to %d", *configHttpRetries)
	HttpRetries = *configHttpRetries

	// Grab the prometheus host
	if *configPrometheusHost == "" {
		log.Fatal("You must provide a -config.prometheus-host, or a HOST environment variable.")
	} else {
		log.Debugf("main(): setting PrometheusHost to %s", *configPrometheusHost)
		PrometheusHost = *configPrometheusHost
	}

	if *configUrlFlag == "" {
		log.Fatal("You must provide a -config.url")
	} else {
		log.Debugf("main(): setting ConfigUrl to %s", *configUrlFlag)
		ConfigUrl = *configUrlFlag
	}

	if *configMustacheSubs == "" {
		log.Fatal("You must provide a -config.mustache-subs")
	} else {
		MustacheSubs = make(map[string]string)
		err := ParseMustacheSubs(MustacheSubs, *configMustacheSubs)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	if *configPrometheusConfigFlag != "" {
		PrometheusConfig = *configPrometheusConfigFlag
	}

	PrometheusConfigFiles = ParseConfigFiles(*configPrometheusConfigFlag)
	AdditionalConfigFiles = ParseConfigFiles(*configAdditionalConfigFlag)

	if _, err = url.ParseRequestURI(ConfigUrl); err != nil {
		log.Fatalf("Cannot parse ConfigUrl=%s", ConfigUrl)
	}

	// Check that we can connect to the url
	if _, err = http.Get(ConfigUrl); err != nil {
		log.Fatalf("Cannot connect to \"%s\", err=%s", ConfigUrl, err.Error())
	}

	// Start up the monitor web server
	monitor := NewMonitor()
	monitor.Start()

	// Get a complete list of files
	AllConfigFiles = append(AllConfigFiles, PrometheusConfigStatic)

	for _, v := range AdditionalConfigFiles {
		AllConfigFiles = append(AllConfigFiles, v)
	}
	// Do one run of PCMSHandler() then start off the scheduler
	PCMSHandler()

	sched := gocron.NewScheduler()
	log.Debugf("main(): starting scheduler for PCMSHandler for every %d seconds", *configSchedulerIntFlag)
	sched.Every(uint64(*configSchedulerIntFlag)).Seconds().Do(PCMSHandler)
	<-sched.Start()
}
