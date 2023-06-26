// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/coordinator"
	"github.com/elastic/elastic-agent/internal/pkg/agent/application/info"
	"github.com/elastic/elastic-agent/pkg/component"
	"github.com/elastic/elastic-agent/pkg/component/runtime"
	"github.com/elastic/elastic-agent/pkg/control/v2/cproto"
)

func TestStateMapping(t *testing.T) {

	testcases := []struct {
		name         string
		agentState   cproto.State
		agentMessage string
		fleetState   cproto.State
		fleetMessage string
	}{
		{
			name:         "waiting first checkin response",
			agentState:   cproto.State_HEALTHY,
			agentMessage: "Healthy",
			fleetState:   cproto.State_STARTING,
			fleetMessage: "",
		},
		{
			name:         "last checkin successful",
			agentState:   cproto.State_HEALTHY,
			agentMessage: "Healthy",
			fleetState:   cproto.State_HEALTHY,
			fleetMessage: "Connected",
		},
		{
			name:         "last checkin failed",
			agentState:   cproto.State_HEALTHY,
			agentMessage: "Healthy",
			fleetState:   cproto.State_FAILED,
			fleetMessage: "<error value coming from fleet gateway>",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			inputState := &coordinator.State{
				State:        tc.agentState,
				Message:      tc.agentMessage,
				FleetState:   tc.fleetState,
				FleetMessage: tc.fleetMessage,
				LogLevel:     logp.ErrorLevel,
				Components: []runtime.ComponentComponentState{
					{
						Component: component.Component{
							ID: "some-component",
							InputSpec: &component.InputRuntimeSpec{
								InputType: "some-component-input-type",
							},
							Units: []component.Unit{
								{
									ID:   "some-input-unit",
									Type: client.UnitTypeInput,
								},
							},
						},
						State: runtime.ComponentState{
							State:   client.UnitStateHealthy,
							Message: "component healthy",
							VersionInfo: runtime.ComponentVersionInfo{
								Name:    "awesome-comp",
								Version: "0.0.1",
								Meta: map[string]string{
									"foo": "bar",
								},
							},
							Units: map[runtime.ComponentUnitKey]runtime.ComponentUnitState{
								{
									UnitType: client.UnitTypeInput,
									UnitID:   "some-input-unit",
								}: {
									State:   client.UnitStateHealthy,
									Message: "unit healthy",
									Payload: map[string]any{
										"foo": map[string]any{
											"bar": "baz"},
									},
								},
							},
						},
					},
				},
			}

			agentInfo := new(info.AgentInfo)

			stateResponse, err := stateToProto(inputState, agentInfo)
			require.NoError(t, err)

			assert.Equal(t, stateResponse.State, tc.agentState)
			assert.Equal(t, stateResponse.Message, tc.agentMessage)
			assert.Equal(t, stateResponse.FleetState, tc.fleetState)
			assert.Equal(t, stateResponse.FleetMessage, tc.fleetMessage)
			if assert.Len(t, stateResponse.Components, 1) {
				expectedCompState := &cproto.ComponentState{
					Id:      "some-component",
					State:   cproto.State_HEALTHY,
					Name:    "some-component-input-type",
					Message: "component healthy",
					Units: []*cproto.ComponentUnitState{
						{
							UnitId:   "some-input-unit",
							UnitType: cproto.UnitType_INPUT,
							State:    cproto.State_HEALTHY,
							Message:  "unit healthy",
							Payload:  "{\"foo\":{\"bar\":\"baz\"}}",
						},
					},
					VersionInfo: &cproto.ComponentVersionInfo{
						Name:    "awesome-comp",
						Version: "0.0.1",
						Meta:    map[string]string{"foo": "bar"},
					},
				}
				assert.Equal(t, expectedCompState, stateResponse.Components[0])
			}

		})
	}

}
