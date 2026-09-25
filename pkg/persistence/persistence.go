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

package persistence

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/process-scheduler/pkg/configuration"
	"github.com/SENERGY-Platform/process-scheduler/pkg/model"
	"github.com/SENERGY-Platform/process-scheduler/pkg/scheduler"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

const TIMEOUT = 10 * time.Second

type Persistence struct {
	config configuration.Config
	client *mongo.Client
}

var (
	errEmptyDatabase   = errors.New("mongo database name must not be empty")
	errMissingPassword = errors.New("mongo password must not be empty when a mongo user is set")
)

func New(ctx context.Context, wg *sync.WaitGroup, config configuration.Config) (scheduler.Persistence, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	var parentCtx context.Context
	if ctx != nil {
		parentCtx = ctx
	} else {
		parentCtx = context.Background()
	}
	client, err := connect(parentCtx, clientOptions(config), config.MongoDatabase, TIMEOUT)
	if err != nil {
		return nil, err
	}
	result := &Persistence{config: config, client: client}

	if ctx != nil {
		if wg != nil {
			wg.Add(1)
		}
		go func() {
			<-ctx.Done()
			result.Disconnect()
			if wg != nil {
				wg.Done()
			}
		}()
	}
	return result, nil
}

// connect runs listCollections on database because Connect is lazy and ping needs no
// authentication; unreachable servers and wrong or missing credentials then fail at startup.
func connect(ctx context.Context, opts *options.ClientOptions, database string, timeout time.Duration) (*mongo.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client, err := mongo.Connect(connectCtx, opts)
	if err != nil {
		return nil, err
	}
	listOpts := options.ListCollections().SetNameOnly(true).SetAuthorizedCollections(true)
	if _, err = client.Database(database).ListCollectionNames(connectCtx, bson.D{}, listOpts); err != nil {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), timeout)
		defer disconnectCancel()
		_ = client.Disconnect(disconnectCtx)
		return nil, fmt.Errorf("mongo startup check failed: %w", err)
	}
	return client, nil
}

func validateConfig(conf configuration.Config) error {
	if conf.MongoDatabase == "" {
		return errEmptyDatabase
	}
	if conf.MongoUser != "" && conf.MongoPassword == "" {
		return errMissingPassword
	}
	return nil
}

// clientOptions applies the credentials after the URI so they replace any given in MONGO_URL.
func clientOptions(conf configuration.Config) *options.ClientOptions {
	opts := options.Client().ApplyURI(conf.MongoUrl)
	if conf.MongoUser != "" {
		opts.SetAuth(options.Credential{
			Username:   conf.MongoUser,
			Password:   conf.MongoPassword,
			AuthSource: conf.MongoAuthSource,
		})
	}
	return opts
}

func (this *Persistence) Disconnect() {
	ctx, _ := getTimeoutContext()
	this.client.Disconnect(ctx)
	return
}

func (this *Persistence) collection() *mongo.Collection {
	return this.client.Database(this.config.MongoDatabase).Collection(this.config.MongoCollection)
}

func (this *Persistence) GetAll() (result []model.ScheduleEntry, err error) {
	ctx, _ := getTimeoutContext()
	cursor, err := this.collection().Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.Background()) {
		entry := model.ScheduleEntry{}
		err = cursor.Decode(&entry)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	err = cursor.Err()
	return
}

func (this *Persistence) Set(entry model.ScheduleEntry) error {
	ctx, _ := getTimeoutContext()
	_, err := this.collection().ReplaceOne(ctx, bson.M{"user": entry.User, "id": entry.Id}, entry, options.Replace().SetUpsert(true))
	return err
}

func (this *Persistence) Get(id string, user string) (result model.ScheduleEntry, err error) {
	ctx, _ := getTimeoutContext()
	err = this.collection().FindOne(ctx, bson.M{"user": user, "id": id}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return result, model.ErrorNotFound
	}
	return result, err
}

func (this *Persistence) Remove(id string, user string) (err error) {
	ctx, _ := getTimeoutContext()
	_, err = this.collection().DeleteOne(ctx, bson.M{"user": user, "id": id})
	return
}

func (this *Persistence) List(user string, createdBy *string) (result []model.ScheduleEntry, err error) {
	filter := bson.M{"user": user}
	if createdBy != nil && *createdBy != "" {
		filter["created_by"] = *createdBy
	}
	ctx, _ := getTimeoutContext()
	cursor, err := this.collection().Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.Background()) {
		entry := model.ScheduleEntry{}
		err = cursor.Decode(&entry)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	err = cursor.Err()
	return
}

func getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TIMEOUT)
}
