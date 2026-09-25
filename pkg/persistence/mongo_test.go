/*
 * Copyright 2026 InfAI (CC SES)
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

package persistence

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const replicaSetURL = "mongodb://mongo-0.mongo:27017,mongo-1.mongo:27017/?replicaSet=rs0&readPreference=primary"

func TestClientOptions_AuthWhenUserGiven(t *testing.T) {
	opts := clientOptions(&configuration.ConfigStruct{
		MongoUrl:        replicaSetURL,
		MongoUser:       "process-scheduler",
		MongoPassword:   "s3cr3t",
		MongoAuthSource: "admin",
		MongoDatabase:   "process_scheduler",
	})
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	want := &options.Credential{Username: "process-scheduler", Password: "s3cr3t", AuthSource: "admin"}
	if !reflect.DeepEqual(opts.Auth, want) {
		t.Errorf("auth = %+v, want %+v", opts.Auth, want)
	}
}

func TestClientOptions_NoAuthWhenUserEmpty(t *testing.T) {
	// A password without a user must not switch auth on.
	opts := clientOptions(&configuration.ConfigStruct{
		MongoUrl:        "mongodb://localhost:27017",
		MongoPassword:   "s3cr3t",
		MongoAuthSource: "admin",
		MongoDatabase:   "process_scheduler",
	})
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	if opts.Auth != nil {
		t.Errorf("auth = %+v, want nil", opts.Auth)
	}
}

func TestClientOptions_ConfiguredCredentialsReplaceURICredentials(t *testing.T) {
	opts := clientOptions(&configuration.ConfigStruct{
		MongoUrl:        "mongodb://old:oldpw@localhost:27017/?authSource=other&authMechanism=SCRAM-SHA-1",
		MongoUser:       "process-scheduler",
		MongoPassword:   "newpw",
		MongoAuthSource: "admin",
		MongoDatabase:   "process_scheduler",
	})
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	want := &options.Credential{Username: "process-scheduler", Password: "newpw", AuthSource: "admin"}
	if !reflect.DeepEqual(opts.Auth, want) {
		t.Errorf("auth = %+v, want %+v", opts.Auth, want)
	}
}

func TestClientOptions_URIPassedUnchanged(t *testing.T) {
	opts := clientOptions(&configuration.ConfigStruct{MongoUrl: replicaSetURL, MongoDatabase: "process_scheduler"})
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := opts.GetURI(); got != replicaSetURL {
		t.Errorf("uri = %q, want %q", got, replicaSetURL)
	}
	if want := []string{"mongo-0.mongo:27017", "mongo-1.mongo:27017"}; !reflect.DeepEqual(opts.Hosts, want) {
		t.Errorf("hosts = %v, want %v", opts.Hosts, want)
	}
	if opts.ReplicaSet == nil || *opts.ReplicaSet != "rs0" {
		t.Errorf("replica set = %v, want rs0", opts.ReplicaSet)
	}
}

func TestClientOptions_NoSchemeAdded(t *testing.T) {
	opts := clientOptions(&configuration.ConfigStruct{MongoUrl: "localhost:27017", MongoDatabase: "process_scheduler"})
	if err := opts.Validate(); err == nil {
		t.Fatal("expected an error for a url without scheme")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     configuration.ConfigStruct
		wantErr error
	}{
		{"no auth", configuration.ConfigStruct{MongoDatabase: "process_scheduler"}, nil},
		{"user and password", configuration.ConfigStruct{MongoDatabase: "process_scheduler", MongoUser: "u", MongoPassword: "p"}, nil},
		{"password without user", configuration.ConfigStruct{MongoDatabase: "process_scheduler", MongoPassword: "p"}, nil},
		{"user without password", configuration.ConfigStruct{MongoDatabase: "process_scheduler", MongoUser: "u"}, errMissingPassword},
		{"empty database", configuration.ConfigStruct{}, errEmptyDatabase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateConfig(&tt.cfg); !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCollectionUsesConfiguredDatabase(t *testing.T) {
	// mongo.Connect does not contact the server, so no running instance is needed.
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	db := &Persistence{config: &configuration.ConfigStruct{MongoDatabase: "custom_db", MongoCollection: "schedules"}, client: client}
	coll := db.collection()
	if coll.Database().Name() != "custom_db" || coll.Name() != "schedules" {
		t.Errorf("collection = %s.%s, want custom_db.schedules", coll.Database().Name(), coll.Name())
	}
}

// unreachableURL points at a port that was just free, so only the startup check can fail.
func unreachableURL(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err = l.Close(); err != nil {
		t.Fatal(err)
	}
	return "mongodb://" + addr + "/?directConnection=true"
}

// The startup check would fail as well, so these check for the specific validation error.
func TestNew_RejectsBeforeConnecting(t *testing.T) {
	cases := map[string]configuration.ConfigStruct{
		"empty database":        {MongoUser: "process-scheduler", MongoPassword: "s3cr3t"},
		"user without password": {MongoUser: "process-scheduler", MongoDatabase: "process_scheduler"},
	}
	want := map[string]error{"empty database": errEmptyDatabase, "user without password": errMissingPassword}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			cfg.MongoUrl = unreachableURL(t)
			wg := &sync.WaitGroup{}
			db, err := New(context.Background(), wg, &cfg)
			if !errors.Is(err, want[name]) {
				t.Fatalf("err = %v, want %v", err, want[name])
			}
			if db != nil {
				t.Error("expected no db on failure")
			}
			if strings.Contains(err.Error(), "s3cr3t") {
				t.Errorf("error leaks the password: %v", err)
			}
		})
	}
}
