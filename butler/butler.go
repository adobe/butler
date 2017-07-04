package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/hoisie/mustache"
	"github.com/jasonlvhit/gocron"
	"github.com/udhos/equalfile"
)

var (
	version                 = "v0.1.0"
	jsonFiles               = `{"files": ["prometheus.yml", "alerts/commonalerts.yml", "alerts/tenant.yml"]}`
	PrometheusRootDirectory = "/opt/prometheus"
	PrometheusHost          string
	ClusterId               string
	ConfigUrl               string
	Files                   ConfigFiles
)

type ConfigFiles struct {
	Files []string `json:files"`
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

	response, err := http.Get(u)
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
		//log.Printf("creating file \"%s\"", dst)
		out, err = os.Create(dst)
	} else {
		//log.Printf("opening file \"%s\"", dst)
		out, err = os.OpenFile(dst, os.O_WRONLY, 0644)
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
	f, err := os.OpenFile(f.Name(), os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()
	_, err = f.Write([]byte(out))
	if err != nil {
		log.Fatal(err.Error())
	}
}

func taskHandler() {
	IsModified := false
	log.Println("Processing PCMS Files.")

	for i, u := range GetPCMSUrls() {
		f := DownloadPCMSFile(u)
		if f == nil {
			continue
		}

		// For the prometheus.yml we have to do some mustache
		// cluster-id replacement on downloaded file
		if GetPrometheusPaths()[i] == fmt.Sprintf("%s/prometheus.yml", PrometheusRootDirectory) {
			RenderPrometheusYaml(f)
		}

		cmp := equalfile.New(nil, equalfile.Options{})
		equal, err := cmp.CompareFile(f.Name(), GetPrometheusPaths()[i])
		if !equal {
			log.Printf("Found difference in \"%s\"... Updating.", GetPrometheusPaths()[i])
			err = CopyFile(f.Name(), GetPrometheusPaths()[i])
			if err != nil {
				log.Fatal(err.Error())
			}

			IsModified = true
		}
		os.Remove(f.Name())
	}
	if IsModified {
		log.Printf("Reloading prometheus.")
		// curl -v -X POST $HOST:9090/-/reload
		promUrl := fmt.Sprintf("http://%s:9090/-/reload", PrometheusHost)
		client := &http.Client{}
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
}

func main() {
	var (
		err                   error
		versionFlag           = flag.Bool("version", false, "Print version information.")
		configUrlFlag         = flag.String("config.url", "", "The base url to grab prometheus configuration files")
		configClusterIdFlag   = flag.String("config.cluster-id", "", "The ethos cluster identifier.")
		configFilesJsonFlag   = flag.String("config.files", jsonFiles, "The prometheus configuration files to grab.")
		configScheduleIntFlag = flag.Int("config.schedule-interval", 5, "The interval, in minutes, to run the schedule.")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Fprintln(os.Stdout, version)
		os.Exit(0)
	}

	// Grab the prometheus host
	PrometheusHost = os.Getenv("HOST")
	if PrometheusHost == "" {
		log.Fatal("Cannot retrieve HOST environment variable")
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

	Files = ConfigFiles{}
	if *configFilesJsonFlag == "" {
		err := json.Unmarshal([]byte(jsonFiles), &Files)
		if err != nil {
			log.Fatalf(err.Error())
		}
	} else {
		err := json.Unmarshal([]byte(*configFilesJsonFlag), &Files)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	if _, err = url.ParseRequestURI(ConfigUrl); err != nil {
		log.Fatalf("Cannot parse ConfigUrl=%s", ConfigUrl)
	}

	// Check that we can connect to the url
	if _, err = http.Get(ConfigUrl); err != nil {
		log.Fatalf("Cannot connect to \"%s\", err=%s", ConfigUrl, err.Error())
	}

	// Check to see if the files currently exist. If the docker path is properly mounted from the prometheus
	// container, then we should see those files.  Error out if we cannot see those files.
	for _, file := range GetPrometheusPaths() {
		if _, err = os.Stat(file); err != nil {
			log.Fatalf("Cannot find file \"%s\". Is the directory properly mounted to docker?", file)
		}
	}

	sched := gocron.NewScheduler()
	sched.Every(uint64(*configScheduleIntFlag)).Minutes().Do(taskHandler)
	<-sched.Start()
}
