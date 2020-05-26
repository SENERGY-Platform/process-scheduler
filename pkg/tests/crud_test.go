/*
 * Copyright 2020 InfAI (CC SES)
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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"github.com/SENERGY-Platform/process-scheduler/pkg/model"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestCrud(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg, config, _, err := Start(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	defer wg.Wait()
	defer cancel()

	id1 := ""
	id2 := ""
	id3 := ""
	t.Run("create schedule user1 deployment-1", createSchedule(config, "* * * * * ? *", "deployment-1", "user1", &id1))
	t.Run("create schedule user1 deployment-2", createSchedule(config, "* * * * * ? *", "deployment-2", "user1", &id2))
	t.Run("create schedule user2 deployment-3", createSchedule(config, "* * * * * ? *", "deployment-3", "user2", &id3))

	t.Run("read schedule user1 deployment-1", readSchedule(config, "* * * * * ? *", "deployment-1", "user1", id1))
	t.Run("read schedule user1 deployment-2", readSchedule(config, "* * * * * ? *", "deployment-2", "user1", id2))
	t.Run("read schedule user2 deployment-3", readSchedule(config, "* * * * * ? *", "deployment-3", "user2", id3))

	t.Run("update schedule user1 deployment-1", updateSchedule(config, "* * * * * * ?", "deployment-1", "user1", id1))
	t.Run("update schedule user1 deployment-4", updateSchedule(config, "* * * * * ? *", "deployment-4", "user1", id2))

	t.Run("read update schedule user1 deployment-1", readSchedule(config, "* * * * * * ?", "deployment-1", "user1", id1))
	t.Run("read update schedule user1 deployment-4", readSchedule(config, "* * * * * ? *", "deployment-4", "user1", id2))

	t.Run("list user1", listSchedules(config, "user1", []model.ScheduleEntry{{
		Id:                  id1,
		Cron:                "* * * * * * ?",
		ProcessDeploymentId: "deployment-1",
	}, {
		Id:                  id2,
		Cron:                "* * * * * ? *",
		ProcessDeploymentId: "deployment-4",
	}}))

	t.Run("list user2", listSchedules(config, "user2", []model.ScheduleEntry{{
		Id:                  id3,
		Cron:                "* * * * * ? *",
		ProcessDeploymentId: "deployment-3",
	}}))

	t.Run("delete id2", deleteSchedule(config, "user1", id2))

	t.Run("list user1", listSchedules(config, "user1", []model.ScheduleEntry{{
		Id:                  id1,
		Cron:                "* * * * * * ?",
		ProcessDeploymentId: "deployment-1",
	}}))

	t.Run("list user2", listSchedules(config, "user2", []model.ScheduleEntry{{
		Id:                  id3,
		Cron:                "* * * * * ? *",
		ProcessDeploymentId: "deployment-3",
	}}))
}

func createSchedule(config configuration.Config, cron string, deploymentId string, userId string, entryId *string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules"
		method := "POST"
		buf := bytes.NewBuffer([]byte{})
		err := json.NewEncoder(buf).Encode(model.ScheduleEntry{
			Cron:                cron,
			ProcessDeploymentId: deploymentId,
		})
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest(method, endpoint+path, buf)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = SetMockAuthToken(req, userId)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err = errors.New(resp.Status + ": " + buf.String())
			t.Error(err)
			return
		}
		result := model.ScheduleEntry{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Error(err)
			return
		}
		if entryId != nil {
			*entryId = result.Id
		}
	}
}

func updateSchedule(config configuration.Config, cron string, deploymentId string, userId string, entryId string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules/" + url.PathEscape(entryId)
		method := "PUT"
		buf := bytes.NewBuffer([]byte{})
		err := json.NewEncoder(buf).Encode(model.ScheduleEntry{
			Cron:                cron,
			ProcessDeploymentId: deploymentId,
		})
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest(method, endpoint+path, buf)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = SetMockAuthToken(req, userId)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err = errors.New(resp.Status + ": " + buf.String())
			t.Error(err)
			return
		}
	}
}

func readSchedule(config configuration.Config, cron string, deploymentId string, userId string, entryId string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules/" + url.PathEscape(entryId)
		method := "GET"
		req, err := http.NewRequest(method, endpoint+path, nil)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = SetMockAuthToken(req, userId)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err = errors.New(resp.Status + ": " + buf.String())
			t.Error(err)
			return
		}
		result := model.ScheduleEntry{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Error(err)
			return
		}
		if result.ProcessDeploymentId != deploymentId {
			t.Error(result.ProcessDeploymentId, deploymentId)
			return
		}
		if result.Cron != cron {
			t.Error(result.Cron, cron)
			return
		}
	}
}

func deleteSchedule(config configuration.Config, userId string, entryId string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules/" + url.PathEscape(entryId)
		method := "DELETE"
		req, err := http.NewRequest(method, endpoint+path, nil)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = SetMockAuthToken(req, userId)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err = errors.New(resp.Status + ": " + buf.String())
			t.Error(err)
			return
		}
	}
}

func listSchedules(config configuration.Config, userId string, expected []model.ScheduleEntry) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules"
		method := "GET"
		req, err := http.NewRequest(method, endpoint+path, nil)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = SetMockAuthToken(req, userId)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err = errors.New(resp.Status + ": " + buf.String())
			t.Error(err)
			return
		}
		result := []model.ScheduleEntry{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Error(err)
			return
		}
		sort.Slice(expected, func(i, j int) bool {
			return expected[i].Id < expected[j].Id
		})
		sort.Slice(result, func(i, j int) bool {
			return result[i].Id < result[j].Id
		})
		if !reflect.DeepEqual(result, expected) {
			t.Error(result, expected)
			return
		}
	}
}
