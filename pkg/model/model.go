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

package model

import (
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
)

type ScheduleEntry struct {
	Id                  string `json:"id" bson:"id"`
	Cron                string `json:"cron" bson:"cron"`
	ProcessDeploymentId string `json:"process_deployment_id" bson:"process_deployment_id"`
}

var ErrorMissingCronExpr = errors.New("missing cron expression")
var ErrorMissingProcessDeploymentId = errors.New("missing process_deployment_id")
var ErrorIdMissmatch = errors.New("path id does not match body id")
var ErrorNotFound = errors.New("not found")
var ErrorAccessDenied = errors.New("access denied")

func (this *ScheduleEntry) Validate() error {
	if this.Cron == "" {
		return ErrorMissingCronExpr
	}
	if this.ProcessDeploymentId == "" {
		return ErrorMissingProcessDeploymentId
	}

	_, err := cron.ParseStandard(this.Cron)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}

func (this *ScheduleEntry) ValidateAndEnsureId(pathId string) error {
	if err := this.Validate(); err != nil {
		return err
	}
	if this.Id == "" {
		this.Id = pathId
	}
	if this.Id != pathId {
		return ErrorIdMissmatch
	}
	return nil
}
