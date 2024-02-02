// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// //go:build integration

package integration

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/sajari/regression"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"

	"github.com/elastic/elastic-agent-libs/kibana"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/estools"
)

type ExtendedRunner struct {
	suite.Suite
	info         *define.Info
	agentFixture *atesting.Fixture

	ESHost string
}

type ComponentMetrics struct {
	Memory        MemoryMetrics  `mapstructure:"memstats"`
	Handles       HandlesMetrics `mapstructure:"handles"`
	UnixTimestamp int64
}

// TestComponent is used as a key in our map of component metrics
type TestComponent struct {
	Binary   string `mapstructure:"binary"`
	Dataset  string `mapstructure:"dataset"`
	ID       string `mapstructure:"id"`
	CompType string `mapstructure:"type"`
}

type MemoryMetrics struct {
	GcNext      uint64 `mapstructure:"gc_next"`
	MemoryAlloc uint64 `mapstructure:"memory_alloc"`
	MemorySys   uint64 `mapstructure:"memory_sys"`
	MemoryTotal uint64 `mapstructure:"memory_total"`
	RSS         uint64 `mapstructure:"rss"`
}

type HandlesMetrics struct {
	Open  int           `mapstructure:"open"`
	Limit HandlesLimits `mapstructure:"limit"`
}

type HandlesLimits struct {
	Hard uint   `mapstructure:"hard"`
	Soft uint64 `mapstructure:"soft"`
}

// MetricsSystem is used for windows handles metrics
type MetricsSystem struct {
	Handles HandlesMetrics `mapstructure:"handles"`
}

func TestAgentLong(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Group: "fleet",
		Stack: &define.Stack{},
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
		OS: []define.OS{
			{Type: define.Linux},
			{Type: define.Windows},
		},
	})

	if os.Getenv("TEST_EXTENDED") == "" {
		t.Skipf("not running extended test unless TEST_EXTENDED is set")
	}

	suite.Run(t, &ExtendedRunner{info: info})
}

func (runner *ExtendedRunner) SetupSuite() {
	// create ~40 1MB files that will be picked up by the `/var/log/httpd/error_log*` pattern
	cmd := exec.Command("go", "install", "-v", "github.com/mingrammer/flog@latest")
	out, err := cmd.CombinedOutput()
	require.NoError(runner.T(), err, "got out: %s", string(out))

	cmd = exec.Command("flog", "-t", "log", "-f", "apache_error", "-o", "/var/log/httpd/error_log", "-b", "50485760", "-p", "1048576")
	out, err = cmd.CombinedOutput()
	require.NoError(runner.T(), err, "got out: %s", string(out))
	runner.T().Logf("printed: %#v", string(out))

	policyUUID := uuid.New().String()
	unpr := false
	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
		Unprivileged:   &unpr,
	}

	fixture, err := define.NewFixture(runner.T(), define.Version())
	require.NoError(runner.T(), err)
	runner.agentFixture = fixture

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	basePolicy := kibana.AgentPolicy{
		Name:        "test-policy-" + policyUUID,
		Namespace:   "default",
		Description: "Test policy " + policyUUID,
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
	}

	policyResp, err := tools.InstallAgentWithPolicy(ctx, runner.T(), installOpts, runner.agentFixture, runner.info.KibanaClient, basePolicy)
	require.NoError(runner.T(), err)
	// install system package

	systemPackage := kibana.PackagePolicyRequest{}

	jsonRaw, err := os.ReadFile("agent_long_test_base_system_integ.json")
	require.NoError(runner.T(), err)

	err = json.Unmarshal(jsonRaw, &systemPackage)
	require.NoError(runner.T(), err)

	systemPackage.ID = policyUUID
	systemPackage.PolicyID = policyResp.ID
	systemPackage.Namespace = "default"
	systemPackage.Name = fmt.Sprintf("system-long-test-%s", policyUUID)
	systemPackage.Vars = map[string]interface{}{}

	runner.T().Logf("Installing fleet package....")
	_, err = runner.info.KibanaClient.InstallFleetPackage(ctx, systemPackage)
	require.NoError(runner.T(), err, "error creating fleet package")

	// install apache

	policyUUIDApache := uuid.New().String()
	apachePackage := kibana.PackagePolicyRequest{}

	jsonRaw, err = os.ReadFile("agent_long_test_apache_integ.json")
	require.NoError(runner.T(), err)

	err = json.Unmarshal(jsonRaw, &apachePackage)
	require.NoError(runner.T(), err)

	apachePackage.ID = policyUUIDApache
	apachePackage.PolicyID = policyResp.ID
	apachePackage.Namespace = "default"
	apachePackage.Name = fmt.Sprintf("system-long-test-%s", policyUUIDApache)
	apachePackage.Vars = map[string]interface{}{}

	runner.T().Logf("Installing fleet package....")
	_, err = runner.info.KibanaClient.InstallFleetPackage(ctx, apachePackage)
	require.NoError(runner.T(), err, "error creating fleet package")

}

func (runner *ExtendedRunner) TestHandleLeak() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	testRuntime := os.Getenv("LONG_TEST_RUNTIME")
	if testRuntime == "" {
		testRuntime = "6m"
	}

	testDuration, err := time.ParseDuration(testRuntime)
	require.NoError(runner.T(), err)

	timer := time.NewTimer(testDuration)

	// time to perform a health check
	ticker := time.Tick(time.Minute)

	done := false
	for {
		if done {
			break
		}
		select {
		case <-timer.C:
			done = true
		case <-ticker:
			err := runner.agentFixture.IsHealthy(ctx)
			require.NoError(runner.T(), err)
		}
	}

	status, err := runner.agentFixture.ExecStatus(ctx)
	require.NoError(runner.T(), err)
	runner.T().Logf("Looking for logs matching agent ID %s", status.Info.ID)

	docs, err := estools.FindMatchingLogLinesForAgent(runner.info.ESClient, status.Info.ID, "last 30s")
	require.NoError(runner.T(), err)

	// iterate over hits, compile metrics based on the component
	componentCollection := map[TestComponent][]ComponentMetrics{}
	for _, doc := range docs.Hits.Hits {

		componentData := doc.Source["component"].(map[string]interface{})
		toComponent := TestComponent{}
		err := mapstructure.Decode(componentData, &toComponent)
		require.NoError(runner.T(), err)

		monitoringData := doc.Source["monitoring"].(map[string]interface{})["metrics"].(map[string]interface{})["beat"].(map[string]interface{})
		metrics := ComponentMetrics{}
		err = mapstructure.Decode(monitoringData, &metrics)
		require.NoError(runner.T(), err)

		// for whatever reason, the windows metrics are reported in a different location
		systemDataSource := doc.Source["monitoring"].(map[string]interface{})["metrics"].(map[string]interface{})["system"]
		systemData := MetricsSystem{}
		err = mapstructure.Decode(systemDataSource, &systemData)
		require.NoError(runner.T(), err)
		if runtime.GOOS == "windows" {
			metrics.Handles.Open = systemData.Handles.Open
		}

		timestamp, err := time.Parse(time.RFC3339Nano, doc.Source["@timestamp"].(string))
		require.NoError(runner.T(), err)

		metrics.UnixTimestamp = timestamp.UnixMicro()

		if foundComp, ok := componentCollection[toComponent]; ok {
			updated := append(foundComp, metrics)
			componentCollection[toComponent] = updated
		} else {
			componentCollection[toComponent] = []ComponentMetrics{metrics}
		}

	}

	handleLimit := 500
	// after we get all the metrics, sort by memory/handles to see if we've passed a threshold
	for comp, metrics := range componentCollection {

		runner.T().Logf("===============================")

		reg := new(regression.Regression)
		reg.SetObserved(fmt.Sprintf("%s handle usage", comp.Dataset))
		reg.SetVar(0, "open handles")
		points := regression.DataPoints{}
		// we're using Sys from `runtime.ReadMemStats` for this. From the `runtime` godoc:
		//
		//Sys is the sum of the XSys fields below.
		//Sys measures the virtual address space reserved by the Go runtime for the heap, stacks, and other internal data structures.
		//It's likely that not all of the virtual address space is backed by physical memory at any given moment, though in general it all was at some point.

		// At some point in the future, this is where we'll fail the test if our metrics go over a given threshold.
		slices.SortFunc(metrics, func(a, b ComponentMetrics) int { return cmp.Compare(a.Memory.MemorySys, b.Memory.MemorySys) })
		highestMem := metrics[len(metrics)-1].Memory.MemorySys
		runner.T().Logf("Top memory usage for %s: %d bytes", comp.Dataset, highestMem)

		highestHandleOpen := metrics[len(metrics)-1].Handles.Open
		slices.SortFunc(metrics, func(a, b ComponentMetrics) int { return cmp.Compare(a.Handles.Open, b.Handles.Open) })

		runner.T().Logf("Top count of open handles for %s: %d", comp.Dataset, highestHandleOpen)
		require.LessOrEqual(runner.T(), highestHandleOpen, handleLimit, "handle count is higher than %d after %s (count: %d), check for resource leaks", handleLimit, testDuration, highestHandleOpen)

		// the limit seems to only exist on linux
		if softLimit := metrics[len(metrics)-1].Handles.Limit.Soft; softLimit > 0 {
			limitPct := (float64(highestHandleOpen) / float64(softLimit)) * 100
			runner.T().Logf("Percent of handle soft limit: %f", limitPct)
		}

		for _, metric := range metrics {
			points = append(points, regression.DataPoint(float64(metric.Handles.Open), []float64{float64(metric.UnixTimestamp)}))
		}

		reg.Train(points...)
		err = reg.Run()
		require.NoError(runner.T(), err)

		runner.T().Logf("formula: %v", reg.Formula)
		runner.T().Logf("regression: %s", reg)
		runner.T().Logf("Coeff: %#v", reg.GetCoeffs())

		runner.T().Logf("===============================")
	}

}
