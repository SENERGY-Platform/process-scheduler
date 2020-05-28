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
	"errors"
	"github.com/dgrijalva/jwt-go"
	"net/http"
)

func SetAuthToken(req *http.Request, user string) (err error) {
	if user == "" {
		return errors.New("missing user")
	}
	claims := &jwt.StandardClaims{
		ExpiresAt: 15000,
		Issuer:    "process-scheduler",
		Subject:   user,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte("ihopeyouareoneofourdevelopers"))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ss)
	return nil
}
