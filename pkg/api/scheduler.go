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

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/process-scheduler/pkg/api/util"
	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"github.com/SENERGY-Platform/process-scheduler/pkg/model"
	"github.com/SENERGY-Platform/process-scheduler/pkg/scheduler"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"runtime/debug"
)

func init() {
	endpoints = append(endpoints, SchedulerEndpoints)
}

func SchedulerEndpoints(router *httprouter.Router, config configuration.Config, jwt util.Jwt, ctrl *scheduler.Scheduler) {

	router.POST("/schedules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		user, err := jwt.ParseRequest(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		entry := model.ScheduleEntry{}
		err = json.NewDecoder(request.Body).Decode(&entry)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}
		err = entry.Validate()
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}
		result, err, code := ctrl.Add(entry, user)
		if err != nil {
			http.Error(writer, err.Error(), code)
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
		}
	})

	router.PUT("/schedules/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		user, err := jwt.ParseRequest(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		entry := model.ScheduleEntry{}
		err = json.NewDecoder(request.Body).Decode(&entry)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}
		err = entry.ValidateAndEnsureId(id)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}
		result, err, code := ctrl.Update(entry, user)
		if err != nil {
			http.Error(writer, err.Error(), code)
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
		}
	})

	router.GET("/schedules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		user, err := jwt.ParseRequest(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		result, err, code := ctrl.List(user)
		if err != nil {
			http.Error(writer, err.Error(), code)
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
		}
	})

	router.DELETE("/schedules/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		user, err := jwt.ParseRequest(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		err, code := ctrl.Delete(id, user)
		if err != nil {
			http.Error(writer, err.Error(), code)
		}
		writer.WriteHeader(http.StatusOK)
	})
}
