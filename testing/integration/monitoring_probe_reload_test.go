//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent/pkg/control/v2/cproto"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MonitoringRunner struct {
	suite.Suite
	info         *define.Info
	agentFixture *atesting.Fixture

	ESHost string

	healthCheckTime        time.Duration
	healthCheckRefreshTime time.Duration

	policyID   string
	policyName string
}

func TestMonitoringLivenessReloadable(t *testing.T) {
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

	suite.Run(t, &MonitoringRunner{info: info, healthCheckTime: time.Minute * 5, healthCheckRefreshTime: time.Second * 5})
}

func (runner *MonitoringRunner) SetupSuite() {
	fixture, err := define.NewFixture(runner.T(), define.Version())
	require.NoError(runner.T(), err)
	runner.agentFixture = fixture

	policyUUID := uuid.New().String()
	basePolicy := kibana.AgentPolicy{
		Name:        "test-policy-" + policyUUID,
		Namespace:   "default",
		Description: "Test policy " + policyUUID,
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
	}

	installOpts := atesting.InstallOpts{
		NonInteractive: true,
		Force:          true,
		Privileged:     true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	policyResp, err := tools.InstallAgentWithPolicy(ctx, runner.T(), installOpts, runner.agentFixture, runner.info.KibanaClient, basePolicy)
	require.NoError(runner.T(), err)

	runner.policyID = policyResp.ID
	runner.policyName = basePolicy.Name

	_, err = tools.InstallPackageFromDefaultFile(ctx, runner.info.KibanaClient, "system", "1.53.1", "system_integration_setup.json", uuid.New().String(), policyResp.ID)
	require.NoError(runner.T(), err)
}

func (runner *MonitoringRunner) TestBeatsMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	runner.AllComponentsHealthy(ctx)

	client := http.Client{Timeout: time.Second}
	endpoint := "http://localhost:6792/liveness"
	// first stage: ensure the default behavior, http monitoring is off
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	require.NoError(runner.T(), err)

	_, err = client.Do(req)
	require.Error(runner.T(), err)

	// set fleet override
	override := map[string]interface{}{
		"name":      runner.policyName,
		"namespace": "default",
		"overrides": map[string]interface{}{
			"agent": map[string]interface{}{
				"monitoring": map[string]interface{}{
					"http": map[string]interface{}{
						"enabled": true,
						"host":    "localhost",
						"port":    6792,
					},
				},
			},
		},
	}

	// set fleet override
	raw, err := json.Marshal(override)
	require.NoError(runner.T(), err)
	reader := bytes.NewBuffer(raw)
	overrideEndpoint := fmt.Sprintf("/api/fleet/agent_policies/%s", runner.policyID)
	statusCode, overrideResp, err := runner.info.KibanaClient.Request("PUT", overrideEndpoint, nil, nil, reader)
	require.NoError(runner.T(), err)
	require.Equal(runner.T(), http.StatusOK, statusCode, "non-200 status code; got response: %s", string(overrideResp))

	runner.AllComponentsHealthy(ctx)

	// check to make sure that we now have a liveness probe response
	req, err = http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	require.NoError(runner.T(), err)

	livenessResp, err := client.Do(req)
	require.NoError(runner.T(), err)
	defer livenessResp.Body.Close()
	require.Equal(runner.T(), http.StatusOK, livenessResp.StatusCode)

	statusStr, err := io.ReadAll(livenessResp.Body)
	require.NoError(runner.T(), err)

	processData := map[string]interface{}{}
	err = json.Unmarshal(statusStr, &processData)
	require.NoError(runner.T(), err)

	// check for list of processes
	respList := processData["processes"].([]interface{})
	require.NotZero(runner.T(), respList)
}

// AllComponentsHealthy ensures all the beats and agent are healthy and working before we continue
func (runner *MonitoringRunner) AllComponentsHealthy(ctx context.Context) {
	compDebugName := ""
	require.Eventually(runner.T(), func() bool {
		allHealthy := true
		status, err := runner.agentFixture.ExecStatus(ctx)

		require.NoError(runner.T(), err)
		for _, comp := range status.Components {
			runner.T().Logf("component state: %s", comp.Message)
			if comp.State != int(cproto.State_HEALTHY) {
				compDebugName = comp.Name
				allHealthy = false
			}
		}
		return allHealthy
	}, runner.healthCheckTime, runner.healthCheckRefreshTime, "install never became healthy: components did not return a healthy state: %s", compDebugName)
}
