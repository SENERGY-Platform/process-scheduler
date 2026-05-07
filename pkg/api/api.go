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
	"context"
	"errors"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/SENERGY-Platform/process-scheduler/pkg/api/util"
	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"github.com/SENERGY-Platform/process-scheduler/pkg/scheduler"
	"github.com/SENERGY-Platform/service-commons/pkg/accesslog"
	"github.com/julienschmidt/httprouter"
)

var endpoints = []func(router *httprouter.Router, config configuration.Config, jwt util.Jwt, control *scheduler.Scheduler){}

// starts http server; if wg is not nil it will be set as done when the server is stopped
func Start(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, ctrl *scheduler.Scheduler, jwt util.Jwt) (err error) {
	config.GetLogger().Info("start api on " + config.ApiPort)
	router := Router(config, ctrl, jwt)
	server := &http.Server{Addr: ":" + config.ApiPort, Handler: router, WriteTimeout: 10 * time.Second, ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	wg.Add(1)
	go func() {
		config.GetLogger().Info("Listening on " + server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			config.GetLogger().Error("FATAL: api server error", "error", err)
			log.Fatal(err)
		}
	}()
	go func() {
		<-ctx.Done()
		config.GetLogger().Info("api shutdown", "result", server.Shutdown(context.Background()))
		wg.Done()
	}()
	return nil
}

func Router(config configuration.Config, ctrl *scheduler.Scheduler, jwt util.Jwt) http.Handler {
	router := httprouter.New()
	for _, e := range endpoints {
		config.GetLogger().Info("add endpoints", "endpoint", runtime.FuncForPC(reflect.ValueOf(e).Pointer()).Name())
		e(router, config, jwt, ctrl)
	}
	config.GetLogger().Info("add logging and cors")
	corsHandler := util.NewCors(router)
	return accesslog.New(corsHandler)
}
