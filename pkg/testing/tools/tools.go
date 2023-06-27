// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/kibana"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/go-elasticsearch/v8"

	"github.com/stretchr/testify/require"
)

// WaitForAgentStatus returns a niladic function that returns true if the agent
// has reached expectedStatus; false otherwise. The returned function is intended
// for use with assert.Eventually or require.Eventually.
func WaitForAgentStatus(t *testing.T, client *kibana.Client, expectedStatus string) func() bool {
	return func() bool {
		currentStatus, err := GetAgentStatus(client)
		if err != nil {
			t.Errorf("unable to determine agent status: %s", err.Error())
			return false
		}

		if currentStatus == expectedStatus {
			return true
		}

		t.Logf("Agent status: %s", currentStatus)
		return false
	}
}

// WaitForPolicyRevision returns a niladic function that returns true if the
// given agent's policy revision has reached the given policy revision; false
// otherwise. The returned function is intended
// for use with assert.Eventually or require.Eventually.
func WaitForPolicyRevision(t *testing.T, client *kibana.Client, agentID string, expectedPolicyRevision int) func() bool {
	return func() bool {
		getAgentReq := kibana.GetAgentRequest{ID: agentID}
		updatedPolicyAgent, err := client.GetAgent(getAgentReq)
		require.NoError(t, err)

		return updatedPolicyAgent.PolicyRevision == expectedPolicyRevision
	}
}

// InstallAgentWithPolicy creates the given policy, enrolls the given agent
// fixture in Fleet using the default Fleet Server, waits for the agent to be
// online, and returns the created policy.
func InstallAgentWithPolicy(t *testing.T, agentFixture *atesting.Fixture, kibClient *kibana.Client, esClient *elasticsearch.Client, createPolicyReq kibana.AgentPolicy) (*kibana.PolicyResponse, error) {
	policy, err := kibClient.CreatePolicy(createPolicyReq)
	if err != nil {
		return nil, fmt.Errorf("unable to create policy: %w", err)
	}

	// Create enrollment API key
	createEnrollmentApiKeyReq := kibana.CreateEnrollmentAPIKeyRequest{
		PolicyID: policy.ID,
	}
	enrollmentToken, err := kibClient.CreateEnrollmentAPIKey(createEnrollmentApiKeyReq)
	if err != nil {
		return nil, fmt.Errorf("unable to create enrollment API key: %w", err)
	}

	// Get default Fleet Server URL
	fleetServerURL, err := GetDefaultFleetServerURL(kibClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get default Fleet Server URL: %w", err)
	}

	// Enroll agent
	output, err := InstallAgent(fleetServerURL, enrollmentToken.APIKey, agentFixture)
	if err != nil {
		t.Log(string(output))
		return nil, fmt.Errorf("unable to enroll Elastic Agent: %w", err)
	}
	t.Logf(">>> Ran Enroll. Output: %s", output)

	// Wait for Agent to be healthy
	timeout, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	ticker := time.Tick(time.Second * 10)
	for {
		select {
		case <-timeout.Done():
			t.Logf(">>> Elastic agent failed")
			logs, err := GetDocumentsInIndex(esClient, "*agent*")
			if err != nil {
				return nil, fmt.Errorf("error fetching debug documents while agent install failed: Elastic Agent status is not online")
			}
			t.Logf("Got agent logs:")
			for _, log := range logs.Hits.Hits {
				t.Logf("%#v", log.Source)
			}
			return nil, fmt.Errorf("Elastic Agent status is not online")
		case <-ticker:
			cb := WaitForAgentStatus(t, kibClient, "online")
			if cb() {
				return policy, nil
			}
		}
	}
	// require.Eventually(
	// 	t,
	// 	WaitForAgentStatus(t, kibClient, "online"),
	// 	2*time.Minute,
	// 	10*time.Second,
	// 	"Elastic Agent status is not online",
	// )

	return policy, nil
}
