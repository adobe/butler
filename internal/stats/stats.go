/*
Copyright 2017 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package stats

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	//log "github.com/sirupsen/logrus"
)

// Prometheus metrics
var (
	ButlerConfigValid       *prometheus.GaugeVec
	ButlerContactSuccess    *prometheus.GaugeVec
	ButlerContactTime       *prometheus.GaugeVec
	ButlerContactRetry      *prometheus.GaugeVec
	ButlerContactRetryTime  *prometheus.GaugeVec
	ButlerKnownGoodCached   *prometheus.GaugeVec
	ButlerKnownGoodRestored *prometheus.GaugeVec
	ButlerReloadCount       *prometheus.GaugeVec
	ButlerReloadSuccess     *prometheus.GaugeVec
	ButlerReloadTime        *prometheus.GaugeVec
	ButlerReloaderRetry     *prometheus.GaugeVec
	ButlerRenderSuccess     *prometheus.GaugeVec
	ButlerRenderTime        *prometheus.GaugeVec
	ButlerWriteSuccess      *prometheus.GaugeVec
	ButlerWriteTime         *prometheus.GaugeVec
	ButlerRemoteRepoUp      *prometheus.GaugeVec
	ButlerRemoteRepoSanity  *prometheus.GaugeVec
	ButlerRepoInSync        *prometheus.GaugeVec
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
	if ButlerReloadCount == nil {
		ButlerReloadCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_localconfig_reload_count",
			Help: "butler reload counter",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerReloadCount)
	}

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
		ButlerReloadCount.With(prometheus.Labels{"manager": label}).Inc()
		ButlerReloadSuccess.With(prometheus.Labels{"manager": label}).Set(SUCCESS)
		ButlerReloadTime.With(prometheus.Labels{"manager": label}).SetToCurrentTime()
	} else {
		ButlerReloadCount.With(prometheus.Labels{"manager": label}).Inc()
		ButlerReloadSuccess.With(prometheus.Labels{"manager": label}).Set(FAILURE)
	}
}

func DeleteButlerReloadVal(label string) {
	statsMutex.Lock()
	if ButlerReloadCount != nil {
		ButlerReloadCount.Delete(prometheus.Labels{"manager": label})
	}
	if ButlerReloadSuccess != nil {
		ButlerReloadSuccess.Delete(prometheus.Labels{"manager": label})
	}
	if ButlerReloadTime != nil {
		ButlerReloadTime.Delete(prometheus.Labels{"manager": label})
	}
	if ButlerReloaderRetry != nil {
		ButlerReloaderRetry.Delete(prometheus.Labels{"manager": label})
	}
	statsMutex.Unlock()
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
			Help: "Did butler successfully contact the remote repository",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerContactSuccess)

	}

	if ButlerContactTime == nil {
		ButlerContactTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_contact_time",
			Help: "Time that butler successfully contacted the remote repository",
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

func SetButlerContactRetryVal(res float64, repo string, file string) {
	// If there are no legit labels, then we don't want to log anything
	// this is a bit hokey, but for some reason it's getting triggered
	if (repo == "") || (file == "") {
		return
	}

	// We don't want to have a race condition where two
	// we try to initialize the same stat at the same time
	// this will cause the prometheus client to panic
	statsMutex.Lock()
	if ButlerContactRetry == nil {
		ButlerContactRetry = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_contact_retry",
			Help: "Did butler retry contact the remote repository",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerContactRetry)

	}

	if ButlerContactRetryTime == nil {
		ButlerContactRetryTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_contact_retry_time",
			Help: "Time that butler retried contact the remote repository",
		}, []string{"config_file", "repo"})
		prometheus.MustRegister(ButlerContactRetryTime)
	}
	statsMutex.Unlock()

	ButlerContactRetry.With(prometheus.Labels{"config_file": file, "repo": repo}).Inc()
	ButlerContactRetryTime.With(prometheus.Labels{"config_file": file, "repo": repo}).SetToCurrentTime()
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

func SetButlerReloaderRetry(res float64, manager string) {
	statsMutex.Lock()
	if ButlerReloaderRetry == nil {
		ButlerReloaderRetry = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_manager_reload_retry",
			Help: "How many retries has butler attempted to reload manager",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerReloaderRetry)
	}
	statsMutex.Unlock()

	ButlerReloaderRetry.With(prometheus.Labels{"manager": manager}).Inc()
}

func SetButlerRemoteRepoUp(res float64, manager string) {
	statsMutex.Lock()
	if ButlerRemoteRepoUp == nil {
		ButlerRemoteRepoUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_up",
			Help: "Have all the files been downloaded by butler",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerRemoteRepoUp)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerRemoteRepoUp.With(prometheus.Labels{"manager": manager}).Set(SUCCESS)
	} else {
		ButlerRemoteRepoUp.With(prometheus.Labels{"manager": manager}).Set(FAILURE)
	}
}

func SetButlerRemoteRepoSanity(res float64, manager string) {
	statsMutex.Lock()
	if ButlerRemoteRepoSanity == nil {
		ButlerRemoteRepoSanity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_remoterepo_sanity",
			Help: "Did all butler managed files pass the sanity checking",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerRemoteRepoSanity)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerRemoteRepoSanity.With(prometheus.Labels{"manager": manager}).Set(SUCCESS)
	} else {
		ButlerRemoteRepoSanity.With(prometheus.Labels{"manager": manager}).Set(FAILURE)
	}
}

func SetButlerRepoInSync(res float64, manager string) {
	statsMutex.Lock()
	if ButlerRepoInSync == nil {
		ButlerRepoInSync = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "butler_local_remote_insync",
			Help: "Are the remote and local repo files the same",
		}, []string{"manager"})
		prometheus.MustRegister(ButlerRepoInSync)
	}
	statsMutex.Unlock()

	if res == SUCCESS {
		ButlerRepoInSync.With(prometheus.Labels{"manager": manager}).Set(SUCCESS)
	} else {
		ButlerRepoInSync.With(prometheus.Labels{"manager": manager}).Set(FAILURE)
	}
}

// GetStatsLabel returns the filename of the provided file in url format.
func GetStatsLabel(file string) string {
	fileSplit := strings.Split(file, "/")
	ret := fileSplit[len(fileSplit)-1]
	return ret
}
