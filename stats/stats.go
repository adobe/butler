package stats

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Prometheus metrics
	ButlerConfigValid       *prometheus.GaugeVec
	ButlerContactSuccess    *prometheus.GaugeVec
	ButlerContactTime       *prometheus.GaugeVec
	ButlerKnownGoodCached   prometheus.Gauge
	ButlerKnownGoodRestored prometheus.Gauge
	ButlerReloadSuccess     prometheus.Gauge
	ButlerReloadTime        prometheus.Gauge
	ButlerRenderSuccess     prometheus.Gauge
	ButlerRenderTime        prometheus.Gauge
	ButlerWriteSuccess      *prometheus.GaugeVec
	ButlerWriteTime         *prometheus.GaugeVec
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

func SetButlerReloadVal(res float64) {
	if ButlerReloadSuccess == nil {
		ButlerReloadSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "butler_localconfig_reload_success",
			Help: "Did butler successfully reload prometheus",
		})
		prometheus.MustRegister(ButlerReloadSuccess)

	}

	if ButlerReloadTime == nil {
		ButlerReloadTime = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "butler_localconfig_reload_time",
			Help: "Time that butler successfully reload prometheus",
		})
		prometheus.MustRegister(ButlerReloadTime)

	}

	if res == SUCCESS {
		ButlerReloadSuccess.Set(SUCCESS)
		ButlerReloadTime.SetToCurrentTime()
	} else {
		ButlerReloadSuccess.Set(FAILURE)
	}
}

func SetButlerRenderVal(res float64) {
	if ButlerRenderSuccess == nil {
		ButlerRenderSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "butler_localconfig_render_success",
			Help: "Did butler successfully render the prometheus.yml",
		})
		prometheus.MustRegister(ButlerRenderSuccess)
	}

	if ButlerRenderTime == nil {
		ButlerRenderTime = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "butler_localconfig_render_time",
			Help: "Time that butler successfully rendered the prometheus.yml",
		})
		prometheus.MustRegister(ButlerRenderTime)
	}

	if res == SUCCESS {
		ButlerRenderSuccess.Set(SUCCESS)
		ButlerRenderTime.SetToCurrentTime()
	} else {
		ButlerRenderSuccess.Set(FAILURE)
	}
}

func SetButlerWriteVal(res float64, label string) {
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

	if res == SUCCESS {
		ButlerWriteSuccess.With(prometheus.Labels{"config_file": label}).Set(SUCCESS)
		ButlerWriteTime.With(prometheus.Labels{"config_file": label}).SetToCurrentTime()
	} else {
		ButlerWriteSuccess.With(prometheus.Labels{"config_file": label}).Set(FAILURE)
	}
}

func SetButlerConfigVal(res float64, label string) {
	if ButlerConfigValid == nil {
		ButlerConfigValid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_config_valid",
			Help: "Is the butler configuration valid",
		}, []string{"config_file"})
		prometheus.MustRegister(ButlerConfigValid)
	}

	if res == SUCCESS {
		ButlerConfigValid.With(prometheus.Labels{"config_file": label}).Set(SUCCESS)
	} else {
		ButlerConfigValid.With(prometheus.Labels{"config_file": label}).Set(FAILURE)
	}
}
func SetButlerContactVal(res float64, label string) {
	if ButlerContactSuccess == nil {
		ButlerContactSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_contact_success",
			Help: "Did butler succesfully contact the remote repository",
		}, []string{"config_file"})
		prometheus.MustRegister(ButlerContactSuccess)

	}

	if ButlerContactTime == nil {
		ButlerContactTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_contact_time",
			Help: "Time that butler succesfully contacted the remote repository",
		}, []string{"config_file"})
		prometheus.MustRegister(ButlerContactTime)
	}

	if res == SUCCESS {
		ButlerContactSuccess.With(prometheus.Labels{"config_file": label}).Set(SUCCESS)
		ButlerContactTime.With(prometheus.Labels{"config_file": label}).SetToCurrentTime()
	} else {
		ButlerContactSuccess.With(prometheus.Labels{"config_file": label}).Set(FAILURE)
	}
}

func SetButlerKnownGoodCachedVal(res float64) {
	if ButlerKnownGoodCached == nil {
		ButlerKnownGoodCached = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "butler_lastknowngood_cached",
			Help: "Did butler cache the known good configuration",
		})
		prometheus.MustRegister(ButlerKnownGoodCached)

	}
}

func SetButlerKnownGoodRestoredVal(res float64) {
	if ButlerKnownGoodRestored == nil {
		ButlerKnownGoodRestored = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "butler_lastknowngood_restored",
			Help: "Did butler restore the known good configuration",
		})
		prometheus.MustRegister(ButlerKnownGoodRestored)

	}

	if res == SUCCESS {
	} else {
	}
}
