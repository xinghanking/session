package redis_session

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

var Session *Options
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

func Init(options *Options) gin.HandlerFunc {
	if options.RedisStore == nil {
		err := errors.New("redis store is nil")
		panic(err)
	}
	return func(Context *gin.Context) {
		if options.SessionName == "" {
			options.SessionName = "PHPSESSID"
		}
		SessionName = options.SessionName
		SessionID, _ = Context.Cookie(options.SessionName)
		if SessionID == "" {
			SessionID = base64.URLEncoding.EncodeToString([]byte(uuid.NewString()))
			Context.SetCookie(SessionName, SessionID, 864000, "/", Context.Request.Host, false, false)
		}
		if options.RedisKeyPrefix == "" {
			options.RedisKeyPrefix = "PHPREDIS_SESSION:"
		}
		StoreKey = options.RedisKeyPrefix + SessionID
		data, err := options.RedisStore.Get(StoreKey).Bytes()
		if err != nil && !errors.Is(err, redis.Nil) {
			panic(err)
		}
		Values, err = options.UnSerialize(data)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		if Values == nil || len(Values) == 0 {
			Values = make(map[string]any)
		}
		Session = options
		defer Save()
		Context.Next()
	}
}

func Set(key string, value any) {
	Values[key] = value
}
func Get(key string) any {
	v, ok := Values[key]
	if !ok {
		return nil
	}
	return v
}
func Del(key string) {
	delete(Values, key)
}
func Save() {
	if Values != nil && len(Values) > 0 {
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
}
