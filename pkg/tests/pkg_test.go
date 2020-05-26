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
	"github.com/SENERGY-Platform/process-scheduler/pkg/tests/services"
	"github.com/dgrijalva/jwt-go"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
)

func Start(ctx context.Context) (wg *sync.WaitGroup, config configuration.Config, processApiRequests chan string, err error) {
	wg = &sync.WaitGroup{}
	apiPort, err := getFreePort()
	if err != nil {
		return wg, nil, nil, err
	}
	_, ip, err := services.MongoContainer(ctx, wg)
	if err != nil {
		return wg, nil, nil, err
	}
	config = &configuration.ConfigStruct{
		ApiPort:         apiPort,
		MongoUrl:        "mongodb://" + ip + ":27017",
		MongoTable:      "test",
		MongoCollection: "test",
	}
	config.ProcessEndpoint, processApiRequests = services.ProcessApiServer(ctx, wg)
	wg2, err := pkg.Start(ctx, config)
	if err != nil {
		return wg, config, processApiRequests, err
	}
	wg.Add(1)
	go func() {
		wg2.Wait()
		wg.Done()
	}()
	return
}

func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer listener.Close()
	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port), nil
}

func SetMockAuthToken(req *http.Request, user string) (err error) {
	claims := &jwt.StandardClaims{
		ExpiresAt: 15000,
		Issuer:    "test",
		Subject:   user,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte("foobar"))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ss)
	return nil
}

func TestStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wg, _, _, err := Start(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	defer wg.Wait()
	defer cancel()
}
