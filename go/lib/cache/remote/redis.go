/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package remote

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
)

type RedisConfig struct {
	Host string
	Port int
}

func NewRedisClient(conf RedisConfig) Client {
	return &redisClient{
		Client: redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d", conf.Host, conf.Port)}),
	}
}

type redisClient struct {
	*redis.Client
}

type redisGetPipeline struct {
	pipeline redis.Pipeliner
	cmds     map[*pb.Snippet]*redis.StringCmd
}

type redisSetPipeline struct {
	pipeline redis.Pipeliner
	cmds     map[string]*redis.StatusCmd
}

func (r *redisClient) NewGetPipeline(size int) GetPipeline {
	return &redisGetPipeline{
		pipeline: r.Pipeline(),
		cmds:     make(map[*pb.Snippet]*redis.StringCmd, size),
	}
}

func (r *redisClient) NewSetPipeline(size int) SetPipeline {
	return &redisSetPipeline{
		pipeline: r.Pipeline(),
		cmds:     make(map[string]*redis.StatusCmd, size),
	}
}

func (r *redisClient) Ready() bool {
	return r.Ping().Err() == nil
}

// Set adds (key, data) to the pipeline. It won't go to redis until you call ExecSet.
func (r *redisSetPipeline) Set(key string, data []byte) {
	r.cmds[key] = r.pipeline.Set(key, data, 0)
}

// ExecSet empties the contents of the pipeline into redis.
func (r *redisSetPipeline) ExecSet() error {
	_, err := r.pipeline.Exec()
	return err
}

func (r *redisSetPipeline) Size() int {
	return len(r.cmds)
}

// Get queues a GET request from redis with token.GetNormalisedText() as the key. It won't be returned until
// you call ExecGet!
func (redisPipeline *redisGetPipeline) Get(token *pb.Snippet) {
	redisPipeline.cmds[token] = redisPipeline.pipeline.Get(token.GetNormalisedText())
}

// ExecGet retrieves values from redis based on the keys queued in the pipeline and executes
// the callback for each.
func (redisPipeline *redisGetPipeline) ExecGet(onResult func(*pb.Snippet, *cache.Lookup) error) error {

	_, err := redisPipeline.pipeline.Exec()
	if err != nil && err != redis.Nil {
		return err
	}

	for key, cmd := range redisPipeline.cmds {
		bytes, err := cmd.Bytes()
		if err == redis.Nil {
			if err = onResult(key, nil); err != nil {
				return err
			}
			continue
		} else if err != nil {
			return err
		}

		var lookup cache.Lookup
		if err = json.Unmarshal(bytes, &lookup); err != nil {
			return err
		}

		if err = onResult(key, &lookup); err != nil {
			return err
		}
	}

	return nil
}

func (redisPipeline *redisGetPipeline) Size() int {
	return len(redisPipeline.cmds)
}
