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
	"sync"
	"testing"

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
