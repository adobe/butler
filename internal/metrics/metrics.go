/*
Copyright 2017-2026 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	//log "github.com/sirupsen/logrus"
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

// Prometheus metrics
var (
	butlerConfigValid       *prometheus.GaugeVec
	butlerContactRetry      *prometheus.GaugeVec
	butlerContactRetryTime  *prometheus.GaugeVec
	butlerContactSuccess    *prometheus.GaugeVec
	butlerContactTime       *prometheus.GaugeVec
	butlerKnownGoodCached   *prometheus.GaugeVec
	butlerKnownGoodRestored *prometheus.GaugeVec
	butlerReloadCount       *prometheus.GaugeVec
	butlerReloadSuccess     *prometheus.GaugeVec
	butlerReloadTime        *prometheus.GaugeVec
	butlerReloaderRetry     *prometheus.GaugeVec
	butlerRemoteRepoSanity  *prometheus.GaugeVec
	butlerRemoteRepoUp      *prometheus.GaugeVec
	butlerRenderSuccess     *prometheus.GaugeVec
	butlerRenderTime        *prometheus.GaugeVec
	butlerRepoInSync        *prometheus.GaugeVec
	butlerWriteSuccess      *prometheus.GaugeVec
	butlerWriteTime         *prometheus.GaugeVec
)

func init() {
	butlerConfigValid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_config_valid",
		Help: "Is the butler configuration valid",
	}, []string{"config_file", "repo"})

	butlerContactRetry = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_contact_retry",
		Help: "Did butler retry contact the remote repository",
	}, []string{"config_file", "repo"})

	butlerContactRetryTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_contact_retry_time",
		Help: "Time that butler retried contact the remote repository",
	}, []string{"config_file", "repo"})

	butlerContactSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_contact_success",
		Help: "Did butler successfully contact the remote repository",
	}, []string{"config_file", "repo"})

	butlerContactTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_contact_time",
		Help: "Time that butler successfully contacted the remote repository",
	}, []string{"config_file", "repo"})

	butlerKnownGoodCached = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_lastknowngood_cached",
		Help: "Did butler cache the known good configuration",
	}, []string{"manager"})

	butlerKnownGoodRestored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_lastknowngood_restored",
		Help: "Did butler restore the known good configuration",
	}, []string{"manager"})

	butlerReloadCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_reload_count",
		Help: "butler reload counter",
	}, []string{"manager"})

	butlerReloadSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_reload_success",
		Help: "Did butler successfully reload prometheus",
	}, []string{"manager"})

	butlerReloadTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_reload_time",
		Help: "Time that butler successfully reload prometheus",
	}, []string{"manager"})

	butlerReloaderRetry = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_manager_reload_retry",
		Help: "How many retries has butler attempted to reload manager",
	}, []string{"manager"})

	butlerRemoteRepoSanity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_sanity",
		Help: "Did all butler managed files pass the sanity checking",
	}, []string{"manager"})

	butlerRemoteRepoUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_remoterepo_up",
		Help: "Have all the files been downloaded by butler",
	}, []string{"manager"})

	butlerRenderSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_render_success",
		Help: "Did butler successfully render the prometheus.yml",
	}, []string{"config_file", "repo"})

	butlerRenderTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_render_time",
		Help: "Time that butler successfully rendered the prometheus.yml",
	}, []string{"config_file", "repo"})

	butlerRepoInSync = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_local_remote_insync",
		Help: "Are the remote and local repo files the same",
	}, []string{"manager"})

	butlerWriteSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_write_success",
		Help: "Did butler successfully write the configuration",
	}, []string{"config_file"})

	butlerWriteTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "butler_localconfig_write_time",
		Help: "Time that butler successfully write the configuration",
	}, []string{"config_file"})

	prometheus.MustRegister(butlerConfigValid)
	prometheus.MustRegister(butlerContactRetry)
	prometheus.MustRegister(butlerContactRetryTime)
	prometheus.MustRegister(butlerContactSuccess)
	prometheus.MustRegister(butlerContactTime)
	prometheus.MustRegister(butlerKnownGoodCached)
	prometheus.MustRegister(butlerKnownGoodRestored)
	prometheus.MustRegister(butlerReloadCount)
	prometheus.MustRegister(butlerReloadSuccess)
	prometheus.MustRegister(butlerReloadTime)
	prometheus.MustRegister(butlerReloaderRetry)
	prometheus.MustRegister(butlerRemoteRepoSanity)
	prometheus.MustRegister(butlerRemoteRepoUp)
	prometheus.MustRegister(butlerRenderSuccess)
	prometheus.MustRegister(butlerRenderTime)
	prometheus.MustRegister(butlerRepoInSync)
	prometheus.MustRegister(butlerWriteTime)
	prometheus.MustRegister(butlerWriteSuccess)
}

func SetButlerReloadVal(res float64, label string) {
	if res == SUCCESS {
		butlerReloadCount.With(prometheus.Labels{"manager": label}).Inc()
		butlerReloadSuccess.With(prometheus.Labels{"manager": label}).Set(SUCCESS)
		butlerReloadTime.With(prometheus.Labels{"manager": label}).SetToCurrentTime()
	} else {
		butlerReloadCount.With(prometheus.Labels{"manager": label}).Inc()
		butlerReloadSuccess.With(prometheus.Labels{"manager": label}).Set(FAILURE)
	}
}

func DeleteButlerReloadVal(label string) {
	butlerReloadCount.Delete(prometheus.Labels{"manager": label})
	butlerReloadSuccess.Delete(prometheus.Labels{"manager": label})
	butlerReloadTime.Delete(prometheus.Labels{"manager": label})
	butlerReloaderRetry.Delete(prometheus.Labels{"manager": label})
}

func SetButlerRenderVal(res float64, repo string, file string) {
	if res == SUCCESS {
		butlerRenderSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(SUCCESS)
		butlerRenderTime.With(prometheus.Labels{"config_file": file, "repo": repo}).SetToCurrentTime()
	} else {
		butlerRenderSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(FAILURE)
	}
}

func SetButlerWriteVal(res float64, label string) {
	if res == SUCCESS {
		butlerWriteSuccess.With(prometheus.Labels{"config_file": label}).Set(SUCCESS)
		butlerWriteTime.With(prometheus.Labels{"config_file": label}).SetToCurrentTime()
	} else {
		butlerWriteSuccess.With(prometheus.Labels{"config_file": label}).Set(FAILURE)
	}
}

func SetButlerConfigVal(res float64, repo string, file string) {
	if res == SUCCESS {
		butlerConfigValid.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(SUCCESS)
	} else {
		butlerConfigValid.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(FAILURE)
	}
}

func SetButlerContactVal(res float64, repo string, file string) {
	if res == SUCCESS {
		butlerContactSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(SUCCESS)
		butlerContactTime.With(prometheus.Labels{"config_file": file, "repo": repo}).SetToCurrentTime()
	} else {
		butlerContactSuccess.With(prometheus.Labels{"config_file": file, "repo": repo}).Set(FAILURE)
	}
}

func SetButlerContactRetryVal(res float64, repo string, file string) {
	// If there are no legit labels, then we don't want to log anything
	// this is a bit hokey, but for some reason it's getting triggered
	if (repo == "") || (file == "") {
		return
	}

	butlerContactRetry.With(prometheus.Labels{"config_file": file, "repo": repo}).Inc()
	butlerContactRetryTime.With(prometheus.Labels{"config_file": file, "repo": repo}).SetToCurrentTime()
}

func SetButlerKnownGoodCachedVal(res float64, label string) {
	if res == SUCCESS {
		butlerKnownGoodCached.With(prometheus.Labels{"manager": label}).Set(SUCCESS)
	} else {
		butlerKnownGoodCached.With(prometheus.Labels{"manager": label}).Set(FAILURE)
	}
}

func SetButlerKnownGoodRestoredVal(res float64, label string) {
	if res == SUCCESS {
		butlerKnownGoodRestored.With(prometheus.Labels{"manager": label}).Set(SUCCESS)
	} else {
		butlerKnownGoodRestored.With(prometheus.Labels{"manager": label}).Set(FAILURE)
	}
}

func SetButlerReloaderRetry(res float64, manager string) {
	butlerReloaderRetry.With(prometheus.Labels{"manager": manager}).Inc()
}

func SetButlerRemoteRepoUp(res float64, manager string) {
	if res == SUCCESS {
		butlerRemoteRepoUp.With(prometheus.Labels{"manager": manager}).Set(SUCCESS)
	} else {
		butlerRemoteRepoUp.With(prometheus.Labels{"manager": manager}).Set(FAILURE)
	}
}

func SetButlerRemoteRepoSanity(res float64, manager string) {
	if res == SUCCESS {
		butlerRemoteRepoSanity.With(prometheus.Labels{"manager": manager}).Set(SUCCESS)
	} else {
		butlerRemoteRepoSanity.With(prometheus.Labels{"manager": manager}).Set(FAILURE)
	}
}

func SetButlerRepoInSync(res float64, manager string) {
	if res == SUCCESS {
		butlerRepoInSync.With(prometheus.Labels{"manager": manager}).Set(SUCCESS)
	} else {
		butlerRepoInSync.With(prometheus.Labels{"manager": manager}).Set(FAILURE)
	}
}

// GetStatsLabel returns the filename of the provided file in url format.
func GetStatsLabel(file string) string {
	fileSplit := strings.Split(file, "/")
	ret := fileSplit[len(fileSplit)-1]
	return ret
}
