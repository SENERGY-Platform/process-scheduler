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
	"context"
	"github.com/SENERGY-Platform/process-scheduler/pkg"
	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"github.com/SENERGY-Platform/process-scheduler/pkg/model"
	"github.com/SENERGY-Platform/process-scheduler/pkg/tests/services"
	"sync"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	t.Parallel()
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	wg1 := &sync.WaitGroup{}
	defer wg1.Wait()
	defer cancel1()
	apiPort, err := getFreePort()
	if err != nil {
		t.Error(err)
		return
	}
	_, ip, err := services.MongoContainer(ctx1, wg1)
	if err != nil {
		t.Error(err)
		return
	}
	config := &configuration.ConfigStruct{
		ApiPort:         apiPort,
		MongoUrl:        "mongodb://" + ip + ":27017",
		MongoTable:      "test",
		MongoCollection: "test",
	}
	var processApiRequests chan string
	config.ProcessEndpoint, processApiRequests = services.ProcessApiServer(ctx1, wg1)
	wg2, err := pkg.Start(ctx2, config)
	if err != nil {
		t.Error(err)
		return
	}

	id1 := ""
	t.Run("create schedule user1 deployment-1", createSchedule(config, "* * * * *", "deployment-1", "user1", &id1, nil, nil, nil))

	cancel2() //stop current connection
	wg2.Wait()

	wg1, err = pkg.Start(ctx1, config) //start new
	if err != nil {
		t.Error(err)
		return
	}

	second := time.Now().Second()
	if second > 50 {
		time.Sleep(15 * time.Second)
	}
	startMinute := time.Now().Minute()
	t.Run("list user1", listSchedules(config, "user1", []model.ScheduleEntry{{
		Id:                  id1,
		Cron:                "* * * * *",
		ProcessDeploymentId: "deployment-1",
	}}, nil))

	time.Sleep(time.Minute)
	t.Run("delete id2", deleteSchedule(config, "user1", id1))

	endMinute := time.Now().Minute()

	expectedCalls := endMinute - startMinute
	if len(processApiRequests) != expectedCalls {
		t.Error(len(processApiRequests), expectedCalls)
		return
	}

	request := <-processApiRequests

	if request != "/deployment/deployment-1/start user1" {
		t.Error(request)
		return
	}
}
