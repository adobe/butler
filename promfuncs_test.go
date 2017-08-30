package main

import  (
	"time"

	. "gopkg.in/check.v1"
	//log "github.com/sirupsen/logrus"
	"github.com/prometheus/client_model/go"
)

type ButlerPromTestSuite struct {
}

var _ = Suite(&ButlerPromTestSuite{})

func (s *ButlerPromTestSuite) SetUpSuite(c *C) {
}

func (s *ButlerPromTestSuite) TestSetButlerReloadVal(c *C) {
	metric_success := io_prometheus_client.Metric{}
	metric_success_ts := io_prometheus_client.Metric{}
	metric_failure := io_prometheus_client.Metric{}
	metric_failure_ts := io_prometheus_client.Metric{}

	// Initial values should be nil for the variables
	c.Assert(ButlerReloadSuccess, IsNil)
	c.Assert(ButlerReloadTime, IsNil)

	// Set it to FAILURE
	SetButlerReloadVal(FAILURE)

	// Now they should NOT be nil
	c.Assert(ButlerReloadSuccess, NotNil)
	c.Assert(ButlerReloadTime, NotNil)

	// There should also be some values for the metric Desc()
	c.Assert(ButlerReloadSuccess.Desc().String(), Equals, "Desc{fqName: \"butler_localconfig_reload_success\", help: \"Did butler successfully reload prometheus\", constLabels: {}, variableLabels: []}")
	c.Assert(ButlerReloadTime.Desc().String(), Equals, "Desc{fqName: \"butler_localconfig_reload_time\", help: \"Time that butler successfully reload prometheus\", constLabels: {}, variableLabels: []}")

	// Let's get the metric values for FAILURE
	ButlerReloadSuccess.Write(&metric_failure)
	ButlerReloadTime.Write(&metric_failure_ts)
	c.Assert(*metric_failure.Gauge.Value, Equals, FAILURE)
	c.Assert(*metric_failure_ts.Gauge.Value, Equals, 0.0)

	// Get timestamp for right now to compare with timestamp of SUCCESSS
	ts_now := time.Now()

	// Set it to SUCCESS
	SetButlerReloadVal(SUCCESS)

	// Let's get the metric values for SUCCESS
	ButlerReloadSuccess.Write(&metric_success)
	ButlerReloadTime.Write(&metric_success_ts)

	c.Assert(*metric_success.Gauge.Value, Equals, SUCCESS)

	// Convert the flat64 to a unix timestamp
	ts_metric := time.Unix(int64(*metric_success_ts.Gauge.Value), 0)

	// The timestamps should be the same since (within a second)
	c.Assert(ts_metric.Truncate(time.Second), Equals, ts_now.Truncate(time.Second))
}
