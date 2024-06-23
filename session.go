package session

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/elliotchance/phpserialize"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"time"
)

type Options struct {
	RedisStore     *redis.Client
	RedisKeyPrefix string
	SessionName    string
	MaxAge         int
	Secure         bool
	HttpOnly       bool
	Expiration     time.Duration
}

var Session Options
var Values map[string]any
var SessionID string
var SessionName string
var StoreKey string

func (s *Options) Serialize(data map[string]any) (string, error) {
	value, err := phpserialize.Marshal(data, &phpserialize.MarshalOptions{OnlyStdClass: true})
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func (s *Options) UnSerialize(value []byte) (map[string]any, error) {
	data := make(map[any]any)
	err := phpserialize.Unmarshal(value, &data)
	if err == nil && len(data) > 0 {
		val := make(map[string]any)
		for k, v := range data {
			val[k.(string)] = ConvertMap(v)
		}
		return val, nil
	}
	return make(map[string]any), err
}

func ConvertMap(input any) any {
	switch d := input.(type) {
	case map[any]any:
		output := make(map[string]any)
		for k, v := range d {
			output[k.(string)] = ConvertMap(v)
		}
		return output
	case []any:
		return d[:]
	default:
		return input
	}
}

var setCookie func(value string, maxAge int)

func Init(options Options) gin.HandlerFunc {
	if options.RedisStore == nil {
		err := errors.New("redis store is nil")
		panic(err)
	}
	if options.RedisKeyPrefix == "" {
		options.RedisKeyPrefix = redisKeyPrefix
	}
	if options.MaxAge == 0 {
		options.MaxAge = maxAge
	}
	return func(Context *gin.Context) {
		setCookie = func(value string, maxAge int) {
			if maxAge == 0 {
				maxAge = options.MaxAge
			}
			Context.SetCookie(SessionName, value, maxAge, "/", Context.Request.Host, options.Secure, options.HttpOnly)
		}
		if options.SessionName == "" {
			options.SessionName = sessionName
		}
		SessionName = options.SessionName
		SessionID, _ = Context.Cookie(options.SessionName)
		Session = options
		Start()
		defer Save()
		Context.Next()
	}
}

func Start() {
	if SessionID == "" {
		SessionID = base64.URLEncoding.EncodeToString([]byte(uuid.NewString()))
		setCookie(SessionID, Session.MaxAge)
		StoreKey = Session.RedisKeyPrefix + SessionID
		Values = make(map[string]any)
	} else {
		StoreKey = Session.RedisKeyPrefix + SessionID
		data, err := Session.RedisStore.Get(StoreKey).Bytes()
		if err != nil && !errors.Is(err, redis.Nil) {
			panic(err)
		}
		if len(data) > 0 {
			Values, err = Session.UnSerialize(data)
			if err != nil {
				fmt.Println(err)
				panic(err)
			}
		}
		if Values == nil {
			Values = make(map[string]any)
		}
	}
}

func Set(key string, value any) {
	Values[key] = value
}

func Get(key string) any {
	v, ok := Values[key]
	if !ok {
		panic("key not found")
	}
	return v
}

func Exist(key string) bool {
	_, ok := Values[key]
	return ok
}

func Del(key string) {
	delete(Values, key)
}

func Save() {
	if Values != nil {
		data, err := Session.Serialize(Values)
		if err == nil && data != "" {
			err = Session.RedisStore.Set(StoreKey, data, Session.Expiration).Err()
		}
		if err != nil {

			panic(err)

		}
	}
}

func Destroy() {
	Values = nil
	Session.RedisStore.Del(StoreKey)
	setCookie("", -1)
	SessionName = sessionName
	SessionID = ""
}
