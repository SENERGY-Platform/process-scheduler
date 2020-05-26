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

package processapi

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ProcessApi struct {
	config configuration.Config
}

func New(config configuration.Config) (result *ProcessApi) {
	return &ProcessApi{config: config}
}

func (this ProcessApi) Execute(id string) {
	endpoint := this.config.ProcessEndpoint + "/deployment/" + url.PathEscape(id) + "/start"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Println("ERROR: decrypt new request", err)
		return
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("ERROR: decrypt request do", err)
		return
	}

	defer resp.Body.Close()
	temp, _ := ioutil.ReadAll(resp.Body) //ensure empty stream
	if resp.StatusCode != http.StatusOK {
		err = errors.New("unexpected response code from " + endpoint)
		log.Println("ERROR: ", err, resp.StatusCode, string(temp))
		return
	}
	return
}
