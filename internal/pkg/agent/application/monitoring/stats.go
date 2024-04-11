// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

const formValueKey = "failon"

type LivenessFailConfig struct {
	Degraded bool `yaml:"degraded" config:"degraded"`
	Failed   bool `yaml:"failed" config:"failed"`
}

// process the form values we get via HTTP
func handleFormValues(req *http.Request) (LivenessFailConfig, error) {
	err := req.ParseForm()
	if err != nil {
		return LivenessFailConfig{}, fmt.Errorf("Error parsing form: %w", err)
	}

	defaultUserCfg := LivenessFailConfig{Degraded: false, Failed: true}

	for formKey, _ := range req.Form {
		if formKey != formValueKey {
			return defaultUserCfg, fmt.Errorf("got invalid HTTP form key: '%s'", formKey)
		}
	}

	userConfig := req.Form.Get(formValueKey)
	switch userConfig {
	case "failed":
		return defaultUserCfg, nil
	case "degraded":
		return LivenessFailConfig{Failed: true, Degraded: true}, nil
	case "":
		return defaultUserCfg, nil
	default:
		return defaultUserCfg, fmt.Errorf("got unexpected value for `%s` attribute: %s", formValueKey, userConfig)
	}
}

func statsHandler(ns *monitoring.Namespace) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := monitoring.CollectStructSnapshot(
			ns.GetRegistry(),
			monitoring.Full,
			false,
		)

		bytes, err := json.Marshal(data)
		var content string
		if err != nil {
			content = fmt.Sprintf("Not valid json: %v", err)
		} else {
			content = string(bytes)
		}
		fmt.Fprint(w, content)

		return nil
	}
}
