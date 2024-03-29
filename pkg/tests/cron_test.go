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
	"testing"
	"time"
)

func TestCron(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	wg, config, processRequests, err := Start(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(config)
	defer wg.Wait()
	defer cancel()

	id1 := ""
	t.Run("create schedule user1 deployment-1", createSchedule(config, "* * * * *", "deployment-1", "user1", &id1, nil, nil, nil))
	time.Sleep(61 * time.Second)
	t.Run("delete id2", deleteSchedule(config, "user1", id1))

	if len(processRequests) != 1 {
		t.Error(len(processRequests))
		return
	}

	request := <-processRequests

	if request != "/deployment/deployment-1/start user1" {
		t.Error(request)
		return
	}

	time.Sleep(61 * time.Second)

	if len(processRequests) != 0 {
		t.Error(len(processRequests))
		return
	}
}

func TestCronWithSeconds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg, config, processRequests, err := Start(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(config)
	defer wg.Wait()
	defer cancel()

	second := time.Now().Second()
	if second < 5 {
		time.Sleep(5 * time.Second)
	}
	if second > 50 {
		time.Sleep(15 * time.Second)
	}

	startSecond := time.Now().Second()

	id1 := ""
	t.Run("create second schedule", createSchedule(config, "* * * * * *", "deployment-1", "user1", &id1, nil, nil, nil))

	time.Sleep(1 * time.Second)
	t.Run("delete id1", deleteSchedule(config, "user1", id1))

	endSecond := time.Now().Second()

	expectedCalls := endSecond - startSecond

	if len(processRequests) != expectedCalls {
		t.Error(len(processRequests), expectedCalls)
		return
	}

	request := <-processRequests

	if request != "/deployment/deployment-1/start user1" {
		t.Error(request)
		return
	}

	time.Sleep(2 * time.Second)

	if len(processRequests) != 0 {
		t.Error(len(processRequests))
		return
	}
}

func TestCronWithSomeSeconds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg, config, processRequests, err := Start(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(config)
	defer wg.Wait()
	defer cancel()

	second := time.Now().Second()
	if second < 5 {
		time.Sleep(5 * time.Second)
	}
	if second > 50 {
		time.Sleep(15 * time.Second)
	}

	startSecond := time.Now().Second()

	id1 := ""
	t.Run("create second schedule", createSchedule(config, "*/2 * * * * *", "deployment-1", "user1", &id1, nil, nil, nil))
	time.Sleep(2 * time.Second)
	t.Run("delete id1", deleteSchedule(config, "user1", id1))

	endSecond := time.Now().Second()

	expectedCalls := (endSecond - startSecond) / 2

	if len(processRequests) != expectedCalls {
		t.Error(len(processRequests), expectedCalls)
		return
	}

	request := <-processRequests

	if request != "/deployment/deployment-1/start user1" {
		t.Error(request)
		return
	}

	time.Sleep(3 * time.Second)

	if len(processRequests) != 0 {
		t.Error(len(processRequests))
		return
	}
}
