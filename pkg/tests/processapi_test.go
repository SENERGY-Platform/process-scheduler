/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"github.com/SENERGY-Platform/process-scheduler/pkg/model"
	"github.com/SENERGY-Platform/process-scheduler/pkg/processapi"
	"github.com/SENERGY-Platform/process-scheduler/pkg/tests/services"
)

func TestProcessApiStart(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		wantSkipped bool
	}{
		{"skips the run when the deployment is switched off (409)", http.StatusConflict, true},
		{"does not skip the run on success (200)", http.StatusOK, false},
		{"does not treat a server error (500) as skipped", http.StatusInternalServerError, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			wg := &sync.WaitGroup{}
			defer wg.Wait()
			defer cancel()

			config := &configuration.ConfigStruct{}
			var requests chan string
			config.ProcessEndpoint, requests = services.ProcessApiServerWithStatus(ctx, wg, c.status)

			skipped := processapi.New(config).Start(model.ScheduleEntry{ProcessDeploymentId: "deployment-1", User: "user1"})
			if skipped != c.wantSkipped {
				t.Errorf("skipped = %v, want %v", skipped, c.wantSkipped)
			}
			if len(requests) != 1 || <-requests != "/deployment/deployment-1/start user1" {
				t.Error("expected exactly one start request for deployment-1 as user1")
			}
		})
	}
}

func TestProcessApiStartGivesUpOnAHangingEngine(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)

	config := &configuration.ConfigStruct{ProcessEndpoint: server.URL}
	done := make(chan bool)
	go func() {
		done <- processapi.New(config).Start(model.ScheduleEntry{ProcessDeploymentId: "deployment-1", User: "user1"})
	}()
	select {
	case skipped := <-done:
		if skipped {
			t.Error("a timed out start must not count as skipped")
		}
	case <-time.After(10 * time.Second):
		t.Error("Start did not return within 10s although its request times out after 5s")
	}
}
