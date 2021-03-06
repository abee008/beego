package session

import (
	"github.com/garyburd/redigo/redis"
	"strconv"
	"strings"
)

var redispder = &RedisProvider{}

var MAX_POOL_SIZE = 100

var redisPool chan redis.Conn

type RedisSessionStore struct {
	c   redis.Conn
	sid string
}

func (rs *RedisSessionStore) Set(key, value interface{}) error {
	//_, err := rs.c.Do("HSET", rs.sid, key, value)
	_, err := rs.c.Do("HSET", rs.sid, key, value)
	return err
}

func (rs *RedisSessionStore) Get(key interface{}) interface{} {
	reply, err := rs.c.Do("HGET", rs.sid, key)
	if err != nil {
		return nil
	}
	return reply
}

func (rs *RedisSessionStore) Delete(key interface{}) error {
	//_, err := rs.c.Do("HDEL", rs.sid, key)
	_, err := rs.c.Do("HDEL", rs.sid, key)
	return err
}

func (rs *RedisSessionStore) Flush() error {
	_, err := rs.c.Do("DEL", rs.sid)
	return err
}

func (rs *RedisSessionStore) SessionID() string {
	return rs.sid
}

func (rs *RedisSessionStore) SessionRelease() {
	rs.c.Close()
}

type RedisProvider struct {
	maxlifetime int64
	savePath    string
	poolsize    int
	password    string
	poollist    *redis.Pool
}

//savepath like redisserveraddr,poolsize,password
//127.0.0.1:6379,100,astaxie
func (rp *RedisProvider) SessionInit(maxlifetime int64, savePath string) error {
	rp.maxlifetime = maxlifetime
	configs := strings.Split(savePath, ",")
	if len(configs) > 0 {
		rp.savePath = configs[0]
	}
	if len(configs) > 1 {
		poolsize, err := strconv.Atoi(configs[1])
		if err != nil || poolsize <= 0 {
			rp.poolsize = MAX_POOL_SIZE
		} else {
			rp.poolsize = poolsize
		}
	} else {
		rp.poolsize = MAX_POOL_SIZE
	}
	if len(configs) > 2 {
		rp.password = configs[2]
	}
	rp.poollist = redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", rp.savePath)
		if err != nil {
			return nil, err
		}
		if rp.password != "" {
			if _, err := c.Do("AUTH", rp.password); err != nil {
				c.Close()
				return nil, err
			}
		}
		return c, err
	}, rp.poolsize)
	return nil
}

func (rp *RedisProvider) SessionRead(sid string) (SessionStore, error) {
	c := rp.poollist.Get()
	//if str, err := redis.String(c.Do("GET", sid)); err != nil || str == "" {
	if str, err := redis.String(c.Do("HGET", sid, sid)); err != nil || str == "" {
		//c.Do("SET", sid, sid, rp.maxlifetime)
		c.Do("HSET", sid, sid, rp.maxlifetime)
	}
	c.Do("EXPIRE", sid, rp.maxlifetime)
	rs := &RedisSessionStore{c: c, sid: sid}
	return rs, nil
}

func (rp *RedisProvider) SessionRegenerate(oldsid, sid string) (SessionStore, error) {
	c := rp.poollist.Get()
	if str, err := redis.String(c.Do("HGET", oldsid, oldsid)); err != nil || str == "" {
		c.Do("HSET", oldsid, oldsid, rp.maxlifetime)
	}
	c.Do("RENAME", oldsid, sid)
	c.Do("EXPIRE", sid, rp.maxlifetime)
	rs := &RedisSessionStore{c: c, sid: sid}
	return rs, nil
}

func (rp *RedisProvider) SessionDestroy(sid string) error {
	c := rp.poollist.Get()
	c.Do("DEL", sid)
	return nil
}

func (rp *RedisProvider) SessionGC() {
	return
}

func init() {
	Register("redis", redispder)
}
