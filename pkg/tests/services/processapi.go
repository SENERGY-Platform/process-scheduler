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

package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
)

func ProcessApiServer(ctx context.Context, wg *sync.WaitGroup) (url string, requests chan string) {
	requests = make(chan string, 100)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.RawPath
		w.WriteHeader(http.StatusOK)
	}))
	url = ts.URL
	wg.Add(1)
	go func() {
		<-ctx.Done()
		ts.Close()
		close(requests)
		wg.Done()
	}()
	return
}
