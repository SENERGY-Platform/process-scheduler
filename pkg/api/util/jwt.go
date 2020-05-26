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

package util

import (
	"errors"
	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strings"
)

type Jwt interface {
	ParseRequest(request *http.Request) (user string, roles []string, groups []string, err error)
}

type JwtImpl struct {
	config configuration.Config
}

func NewJwt(config configuration.Config) Jwt {
	return JwtImpl{config: config}
}

const PEM_BEGIN = "-----BEGIN PUBLIC KEY-----"
const PEM_END = "-----END PUBLIC KEY-----"

func (this JwtImpl) Parse(token string) (user string, roles []string, groups []string, err error) {
	claims := jwt.MapClaims{}
	parser := jwt.Parser{}
	_, _, err = parser.ParseUnverified(token, &claims)
	if err != nil {
		return user, roles, groups, err
	}
	user, ok := claims["sub"].(string)
	if !ok {
		return user, roles, groups, errors.New("missing jwt sub")
	}
	realmAccess, ok := claims["realm_access"].(map[string]interface{})
	if !ok {
		return user, roles, groups, errors.New("missing jwt realm_access")
	}
	realmRoles, ok := realmAccess["roles"].([]interface{})
	if !ok {
		return user, roles, groups, errors.New("missing jwt realm_access.roles")
	}
	for _, role := range realmRoles {
		roleName, ok := role.(string)
		if !ok {
			return user, roles, groups, errors.New("jwt realm_access.roles enty is not string")
		}
		roles = append(roles, roleName)
	}
	userGroups, ok := claims["groups"].([]interface{})
	if !ok {
		userGroups = []interface{}{}
	}
	for _, group := range userGroups {
		groupName, ok := group.(string)
		if !ok {
			return user, roles, groups, errors.New("jwt realm_access.groups entry is not string")
		}
		groups = append(groups, groupName)
	}
	return
}

func (this JwtImpl) ParseRequest(request *http.Request) (user string, roles []string, groups []string, err error) {
	auth := request.Header.Get("Authorization")
	if auth == "" {
		err = errors.New("missing Authorization header")
	}
	authParts := strings.Split(auth, " ")
	if len(authParts) != 2 {
		return user, roles, groups, errors.New("expect auth string format like '<type> <token>'")
	}
	return this.Parse(strings.Join(authParts[1:], " "))
}
