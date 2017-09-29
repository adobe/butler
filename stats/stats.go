package stats

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	//log "github.com/sirupsen/logrus"
)

var (
	// Prometheus metrics
	ButlerConfigValid       *prometheus.GaugeVec
	ButlerContactSuccess    *prometheus.GaugeVec
	ButlerContactTime       *prometheus.GaugeVec
	ButlerKnownGoodCached   *prometheus.GaugeVec
	ButlerKnownGoodRestored *prometheus.GaugeVec
	ButlerReloadSuccess     *prometheus.GaugeVec
	ButlerReloadTime        *prometheus.GaugeVec
	ButlerRenderSuccess     *prometheus.GaugeVec
	ButlerRenderTime        *prometheus.GaugeVec
	ButlerWriteSuccess      *prometheus.GaugeVec
	ButlerWriteTime         *prometheus.GaugeVec
	statsMutex              *sync.Mutex
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

func init() {
	statsMutex = &sync.Mutex{}
}

func SetButlerReloadVal(res float64, label string) {
	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerReloadSuccess == nil {
		ButlerReloadSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_localconfig_reload_success",
			Help: "Did butler successfully reload prometheus",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerReloadSuccess)

	}

	if ButlerReloadTime == nil {
		ButlerReloadTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_localconfig_reload_time",
			Help: "Time that butler successfully reload prometheus",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerReloadTime)

	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerReloadSuccess.With(prometheus.Labels{"manager": label}).Set(SUCCESS)
		ButlerReloadTime.With(prometheus.Labels{"manager": label}).SetToCurrentTime()
	} else {
		ButlerReloadSuccess.With(prometheus.Labels{"manager": label}).Set(FAILURE)
	}
}

func SetButlerRenderVal(res float64, repo string, file string) {
	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerRenderSuccess == nil {
		ButlerRenderSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_localconfig_render_success",
			Help: "Did butler successfully render the prometheus.yml",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerRenderSuccess)
	}

	if ButlerRenderTime == nil {
		ButlerRenderTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_localconfig_render_time",
			Help: "Time that butler successfully rendered the prometheus.yml",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerRenderTime)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerRenderSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(SUCCESS)
		ButlerRenderTime.With(prometheus.Labels{"config_file": file, "repo": repo}).SetToCurrentTime()
	} else {
		ButlerRenderSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(FAILURE)
	}
}

func SetButlerWriteVal(res float64, label string) {
	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerWriteSuccess == nil {
		ButlerWriteSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_localconfig_write_success",
			Help: "Did butler successfully write the configuration",
		}, []string{"config_file"})
		prometheus.MustRegister(ButlerWriteSuccess)
	}

	if ButlerWriteTime == nil {
		ButlerWriteTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_localconfig_write_time",
			Help: "Time that butler successfully write the configuration",
		}, []string{"config_file"})
		prometheus.MustRegister(ButlerWriteTime)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerWriteSuccess.With(prometheus.Labels{"config_file": label}).Set(SUCCESS)
		ButlerWriteTime.With(prometheus.Labels{"config_file": label}).SetToCurrentTime()
	} else {
		ButlerWriteSuccess.With(prometheus.Labels{"config_file": label}).Set(FAILURE)
	}
}

func SetButlerConfigVal(res float64, repo string, file string) {
	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerConfigValid == nil {
		ButlerConfigValid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_config_valid",
			Help: "Is the butler configuration valid",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerConfigValid)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerConfigValid.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(SUCCESS)
	} else {
		ButlerConfigValid.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(FAILURE)
	}
}
func SetButlerContactVal(res float64, repo string, file string) {
	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerContactSuccess == nil {
		ButlerContactSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_contact_success",
			Help: "Did butler succesfully contact the remote repository",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerContactSuccess)

	}

	if ButlerContactTime == nil {
		ButlerContactTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_contact_time",
			Help: "Time that butler succesfully contacted the remote repository",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerContactTime)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerContactSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(SUCCESS)
		ButlerContactTime.With(prometheus.Labels{"config_file": file, "repo": repo}).SetToCurrentTime()
	} else {
		ButlerContactSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(FAILURE)
	}
}

func SetButlerKnownGoodCachedVal(res float64, label string) {
	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerKnownGoodCached == nil {
		ButlerKnownGoodCached = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_lastknowngood_cached",
			Help: "Did butler cache the known good configuration",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerKnownGoodCached)

	}
	if res == SUCCESS {
		ButlerKnownGoodCached.With(prometheus.Labels{"manager": label}).Set(SUCCESS)
	} else {
		ButlerKnownGoodCached.With(prometheus.Labels{"manager": label}).Set(FAILURE)
	}
	statsMutex.Unlock()
}

func SetButlerKnownGoodRestoredVal(res float64, label string) {
	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerKnownGoodRestored == nil {
		ButlerKnownGoodRestored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_lastknowngood_restored",
			Help: "Did butler restore the known good configuration",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerKnownGoodRestored)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerKnownGoodRestored.With(prometheus.Labels{"manager": label}).Set(SUCCESS)
	} else {
		ButlerKnownGoodRestored.With(prometheus.Labels{"manager": label}).Set(FAILURE)
	}
}

// GetStatsLabel returns the filename of the provided file in url format.
func GetStatsLabel(file string) string {
	fileSplit := strings.Split(file, "/")
	ret := fileSplit[len(fileSplit)-1]
	return ret
}
