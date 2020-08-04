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

import "github.com/SENERGY-Platform/process-scheduler/pkg/model"

type ProcessApi interface {
	Execute(entry model.ScheduleEntry)
}

type Persistence interface {
	GetAll() ([]model.ScheduleEntry, error)
	Set(entry model.ScheduleEntry) error
	Get(id string, userId string) (model.ScheduleEntry, error)
	Remove(id string, user string) error
	List(user string, createdBy *string) ([]model.ScheduleEntry, error)
}
