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

package scheduler

import (
	"context"
	"github.com/SENERGY-Platform/process-scheduler/pkg/model"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"net/http"
	"sync"
)

type Scheduler struct {
	persistence Persistence
	processes   ProcessApi
	cron        *cron.Cron
	jobById     map[string]cron.EntryID
}

func New(persistence Persistence, processes ProcessApi) *Scheduler {
	return &Scheduler{
		persistence: persistence,
		processes:   processes,
		jobById:     map[string]cron.EntryID{},
	}
}

func (this *Scheduler) Start(ctx context.Context, wg *sync.WaitGroup) error {
	this.cron = cron.New()
	entries, err := this.persistence.GetAll()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		err = this.addCron(entry)
		if err != nil {
			return err
		}
	}
	this.cron.Start()

	if ctx != nil {
		if wg != nil {
			wg.Add(1)
		}
		go func() {
			<-ctx.Done()
			this.Stop()
			if wg != nil {
				wg.Done()
			}
		}()
	}
	return nil
}

func (this *Scheduler) Stop() {
	if this.cron != nil {
		this.cron.Stop()
	}
}

func (this *Scheduler) Add(entry model.ScheduleEntry, user string) (result model.ScheduleEntry, err error, code int) {
	entry.Id = uuid.New().String()
	entry.User = user
	err = this.addCron(entry)
	if err != nil {
		return entry, err, http.StatusBadRequest
	}
	err = this.persistence.Set(entry)
	if err != nil {
		this.removeCron(entry.Id)
		return entry, err, http.StatusInternalServerError
	}
	return entry, nil, http.StatusOK
}

func (this *Scheduler) Update(entry model.ScheduleEntry, user string) (result model.ScheduleEntry, err error, code int) {
	entry.User = user
	old, err := this.persistence.Get(entry.Id, user)
	if err != nil {
		return result, err, getErrCode(err)
	}
	this.removeCron(entry.Id)
	err = this.addCron(entry)
	if err != nil {
		this.addCron(old) //try to recover
		return entry, err, http.StatusBadRequest
	}
	err = this.persistence.Set(entry)
	if err != nil {
		this.removeCron(entry.Id)
		this.addCron(old) //try to recover
		return entry, err, http.StatusInternalServerError
	}
	return entry, nil, http.StatusOK
}

func (this *Scheduler) Delete(id string, user string) (err error, code int) {
	err = this.persistence.Remove(id, user)
	if err != nil {
		return err, getErrCode(err)
	}
	this.removeCron(id)
	return nil, http.StatusOK
}

func (this *Scheduler) List(user string) (result []model.ScheduleEntry, err error, code int) {
	result, err = this.persistence.List(user)
	return result, err, getErrCode(err)
}

func (this *Scheduler) addCron(entry model.ScheduleEntry) error {
	if entry.Disabled != nil && *entry.Disabled == true {
		return nil
	}
	id, err := this.cron.AddFunc(entry.Cron, func() {
		this.runJob(entry)
	})
	if err != nil {
		return err
	}
	this.jobById[entry.Id] = id
	return nil
}

func (this *Scheduler) removeCron(externalId string) {
	id, ok := this.jobById[externalId]
	if !ok {
		return
	}
	this.cron.Remove(id)
	return
}

func (this *Scheduler) runJob(entry model.ScheduleEntry) {
	this.processes.Execute(entry)
}

func getErrCode(err error) int {
	if err == model.ErrorNotFound {
		return http.StatusNotFound
	}
	if err == model.ErrorAccessDenied {
		return http.StatusNotFound
	}
	if err != nil {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}
