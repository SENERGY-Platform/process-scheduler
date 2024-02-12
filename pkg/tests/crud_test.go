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
	"github.com/SENERGY-Platform/process-scheduler/pkg/processapi"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestCrud(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	wg, config, _, err := Start(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(config)
	defer wg.Wait()
	defer cancel()

	id1 := ""
	id2 := ""
	id3 := ""
	alias1 := "alias1"
	alias2 := "alias2"
	bTrue := true
	bFalse := false
	t.Run("create schedule user1 deployment-1", createSchedule(config, "* * * ? *", "deployment-1", "user1", &id1, &alias1, &bTrue, nil))
	t.Run("create schedule user1 deployment-2", createSchedule(config, "* * * ? *", "deployment-2", "user1", &id2, nil, nil, nil))
	t.Run("create schedule user2 deployment-3", createSchedule(config, "* * * ? *", "deployment-3", "user2", &id3, nil, nil, nil))

	t.Run("read schedule user1 deployment-1", readSchedule(config, "* * * ? *", "deployment-1", "user1", id1, &alias1, &bTrue, nil))
	t.Run("read schedule user1 deployment-2", readSchedule(config, "* * * ? *", "deployment-2", "user1", id2, nil, nil, nil))
	t.Run("read schedule user2 deployment-3", readSchedule(config, "* * * ? *", "deployment-3", "user2", id3, nil, nil, nil))

	t.Run("update schedule user1 deployment-1", updateSchedule(config, "* * * * ?", "deployment-1", "user1", id1, &alias2, &bFalse, nil))
	t.Run("update schedule user1 deployment-4", updateSchedule(config, "* * * ? *", "deployment-4", "user1", id2, nil, nil, nil))

	t.Run("read update schedule user1 deployment-1", readSchedule(config, "* * * * ?", "deployment-1", "user1", id1, &alias2, &bFalse, nil))
	t.Run("read update schedule user1 deployment-4", readSchedule(config, "* * * ? *", "deployment-4", "user1", id2, nil, nil, nil))

	t.Run("list user1", listSchedules(config, "user1", []model.ScheduleEntry{{
		Id:                  id1,
		Cron:                "* * * * ?",
		ProcessDeploymentId: "deployment-1",
		ProcessAlias:        &alias2,
		Disabled:            &bFalse,
	}, {
		Id:                  id2,
		Cron:                "* * * ? *",
		ProcessDeploymentId: "deployment-4",
	}}, nil))

	t.Run("list user2", listSchedules(config, "user2", []model.ScheduleEntry{{
		Id:                  id3,
		Cron:                "* * * ? *",
		ProcessDeploymentId: "deployment-3",
	}}, nil))

	t.Run("delete id2", deleteSchedule(config, "user1", id2))

	t.Run("list user1", listSchedules(config, "user1", []model.ScheduleEntry{{
		Id:                  id1,
		Cron:                "* * * * ?",
		ProcessDeploymentId: "deployment-1",
		ProcessAlias:        &alias2,
		Disabled:            &bFalse,
	}}, nil))

	t.Run("list user2", listSchedules(config, "user2", []model.ScheduleEntry{{
		Id:                  id3,
		Cron:                "* * * ? *",
		ProcessDeploymentId: "deployment-3",
	}}, nil))

	t1 := "t1"
	t2 := "t2"
	t.Run("create schedule user1 deployment-1 created_by=t1", createSchedule(config, "* * * ? *", "deployment-1", "user1", &id1, nil, nil, &t1))
	t.Run("create schedule user1 deployment-1 created_by=t2", createSchedule(config, "* * * ? *", "deployment-1", "user1", &id2, nil, nil, &t2))
	t.Run("list user1 t1", listSchedules(config, "user1", []model.ScheduleEntry{{
		Id:                  id1,
		Cron:                "* * * ? *",
		ProcessDeploymentId: "deployment-1",
		ProcessAlias:        nil,
		Disabled:            nil,
		CreatedBy:           &t1,
	}}, &t1))
	t.Run("list user1 t2", listSchedules(config, "user1", []model.ScheduleEntry{{
		Id:                  id2,
		Cron:                "* * * ? *",
		ProcessDeploymentId: "deployment-1",
		ProcessAlias:        nil,
		Disabled:            nil,
		CreatedBy:           &t2,
	}}, &t2))
	t.Run("delete id1", deleteSchedule(config, "user1", id1))
	t.Run("delete id2", deleteSchedule(config, "user1", id2))

}

func createSchedule(config configuration.Config, cron string, deploymentId string, userId string, entryId *string, alias *string, disabled *bool, createdBy *string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules"
		method := "POST"
		buf := bytes.NewBuffer([]byte{})
		err := json.NewEncoder(buf).Encode(model.ScheduleEntry{
			Cron:                cron,
			ProcessDeploymentId: deploymentId,
			ProcessAlias:        alias,
			Disabled:            disabled,
			CreatedBy:           createdBy,
		})
		if err != nil {
			t.Error(err)
			return
		}
		log.Println("HTTP-CALL=", method, endpoint+path)
		req, err := http.NewRequest(method, endpoint+path, buf)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = processapi.SetAuthToken(req, userId)
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

func updateSchedule(config configuration.Config, cron string, deploymentId string, userId string, entryId string, alias *string, disabled *bool, createdBy *string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules/" + url.PathEscape(entryId)
		method := "PUT"
		buf := bytes.NewBuffer([]byte{})
		err := json.NewEncoder(buf).Encode(model.ScheduleEntry{
			Cron:                cron,
			ProcessDeploymentId: deploymentId,
			ProcessAlias:        alias,
			Disabled:            disabled,
			CreatedBy:           createdBy,
		})
		if err != nil {
			t.Error(err)
			return
		}
		log.Println("HTTP-CALL=", method, endpoint+path)
		req, err := http.NewRequest(method, endpoint+path, buf)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = processapi.SetAuthToken(req, userId)
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

func readSchedule(config configuration.Config, cron string, deploymentId string, userId string, entryId string, alias *string, disabled *bool, createdBy *string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Skip("no read of single schedule implemented")
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules/" + url.PathEscape(entryId)
		method := "GET"
		log.Println("HTTP-CALL=", method, endpoint+path)
		req, err := http.NewRequest(method, endpoint+path, nil)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = processapi.SetAuthToken(req, userId)
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
		if result.ProcessAlias != nil || alias != nil {
			if result.ProcessAlias == nil || alias == nil {
				t.Error(result.ProcessAlias, alias)
				return
			}
			if *result.ProcessAlias != *alias {
				t.Error(*result.ProcessAlias, *alias)
				return
			}
		}
		if result.Disabled != nil || disabled != nil {
			if result.Disabled == nil || disabled == nil {
				t.Error(result.Disabled, disabled)
				return
			}
			if *result.Disabled != *disabled {
				t.Error(*result.Disabled, *disabled)
				return
			}
		}
		if result.CreatedBy != nil || createdBy != nil {
			if result.CreatedBy == nil || createdBy == nil {
				t.Error(result.CreatedBy, createdBy)
				return
			}
			if *result.CreatedBy != *createdBy {
				t.Error(*result.CreatedBy, *createdBy)
				return
			}
		}
	}
}

func deleteSchedule(config configuration.Config, userId string, entryId string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules/" + url.PathEscape(entryId)
		method := "DELETE"
		log.Println("HTTP-CALL=", method, endpoint+path)
		req, err := http.NewRequest(method, endpoint+path, nil)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = processapi.SetAuthToken(req, userId)
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

func listSchedules(config configuration.Config, userId string, expected []model.ScheduleEntry, createdBy *string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := "http://localhost:" + config.ApiPort
		path := "/schedules"
		if createdBy != nil {
			path += "?created_by=" + *createdBy
		}
		method := "GET"
		log.Println("HTTP-CALL=", method, endpoint+path)
		req, err := http.NewRequest(method, endpoint+path, nil)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		req.WithContext(ctx)
		err = processapi.SetAuthToken(req, userId)
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
