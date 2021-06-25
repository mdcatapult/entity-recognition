package db

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type RedisConfig struct {
	Host string
	Port int
}

func NewRedisClient(conf RedisConfig) Client {
	return redisClient{
		Client: redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d", conf.Host, conf.Port)}),
	}
}

type redisClient struct {
	*redis.Client
}

type redisPipeline struct {
	pipe redis.Pipeliner
	cmds map[*pb.Snippet]*redis.StringCmd
}

func (r redisClient) NewPipeline(size int) Pipeline {
	return redisPipeline{
		pipe: r.Pipeline(),
		cmds: make(map[*pb.Snippet]*redis.StringCmd, size),
	}
}

func (r redisPipeline) Set(key string, data []byte) {
	r.pipe.Set(key, data, 0)
}

func (r redisPipeline) ExecSet() error {
	_, err := r.pipe.Exec()
	return err
}

func (r redisPipeline) Get(token *pb.Snippet) {
	r.cmds[token] = r.pipe.Get(string(token.GetData()))
}

func (r redisPipeline) ExecGet(onResult func(*pb.Snippet, *Lookup) error) error {

	_, err := r.pipe.Exec()
	if err != nil && err != redis.Nil {
		return err
	}

	for key, cmd := range r.cmds {
		b, err := cmd.Bytes()
		if err == redis.Nil {
			if err = onResult(key, nil); err != nil {
				return err
			}
			continue
		} else if err != nil {
			return err
		}

		var lookup Lookup
		if err = json.Unmarshal(b, &lookup); err != nil {
			return err
		}

		if err = onResult(key, &lookup); err != nil {
			return err
		}
	}

	return nil
}

func (r redisPipeline) Size() int {
	return len(r.cmds)
}