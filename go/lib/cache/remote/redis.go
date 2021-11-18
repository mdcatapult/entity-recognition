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
	pipe redis.Pipeliner
	cmds map[*pb.Snippet]*redis.StringCmd
}

type redisSetPipeline struct {
	pipe redis.Pipeliner
	cmds map[string]*redis.StatusCmd
}

func (r *redisClient) NewGetPipeline(size int) GetPipeline {
	return &redisGetPipeline{
		pipe: r.Pipeline(),
		cmds: make(map[*pb.Snippet]*redis.StringCmd, size),
	}
}

func (r *redisClient) NewSetPipeline(size int) SetPipeline {
	return &redisSetPipeline{
		pipe: r.Pipeline(),
		cmds: make(map[string]*redis.StatusCmd, size),
	}
}

func (r *redisClient) Ready() bool {
	return r.Ping().Err() == nil
}

func (r *redisSetPipeline) Set(key string, data []byte) {
	r.cmds[key] = r.pipe.Set(key, data, 0)
}

func (r *redisSetPipeline) ExecSet() error {
	_, err := r.pipe.Exec()
	return err
}

func (r *redisSetPipeline) Size() int {
	return len(r.cmds)
}

func (r *redisGetPipeline) Get(token *pb.Snippet) {
	r.cmds[token] = r.pipe.Get(token.GetNormalisedText())
}

func (r *redisGetPipeline) ExecGet(onResult func(*pb.Snippet, *cache.Lookup) error) error {

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

		var lookup cache.Lookup
		if err = json.Unmarshal(b, &lookup); err != nil {
			return err
		}

		if err = onResult(key, &lookup); err != nil {
			return err
		}
	}

	return nil
}

func (r *redisGetPipeline) Size() int {
	return len(r.cmds)
}
