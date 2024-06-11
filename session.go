package session

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/elliotchance/phpserialize"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

type Options struct {
	RedisStore     *redis.Client
	RedisKeyPrefix string
	SessionName    string
	MaxAge         int
	Secure         bool
	HttpOnly       bool
	Expiration     int64
}

var Session *Options
var Values map[string]any
var SessionID string
var SessionName string
var StoreKey string

func (s *Options) Serialize(data map[string]any) (string, error) {
	value, err := phpserialize.Marshal(data, nil)
	if err != nil {
		return "", err
	}
	return string(value), nil
}
func (s *Options) UnSerialize(value string) (map[string]any, error) {
	var data map[string]any
	err := phpserialize.Unmarshal([]byte(value), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func Init(options *Options) gin.HandlerFunc {
	Session = options
	if options.RedisStore == nil {
		err := errors.New("redis store is nil")
		panic(err)
	}
	return func(Context *gin.Context) {
		if Session.SessionName == "" {
			Session.SessionName = "PHPSESSID"
		}
		SessionName = Session.SessionName
		SessionID, _ = Context.Cookie(Session.SessionName)
		if SessionID == "" {
			SessionID = base64.URLEncoding.EncodeToString([]byte(uuid.NewString()))
			Values = make(map[string]any)
			Context.SetCookie(SessionName, SessionID, 864000, "/", Context.Request.Host, false, false)
		}
		if Session.RedisKeyPrefix == "" {
			Session.RedisKeyPrefix = "PHPREDIS_SESSION:"
		}
		StoreKey = Session.RedisKeyPrefix + SessionID
		data, err := Session.RedisStore.Get(StoreKey).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				panic(err)
			}
			Values = make(map[string]any)
		}
		if data != "" {
			Values, err = Session.UnSerialize(data)
			if err != nil {
				fmt.Println(err)
			}
		}
		defer Save()
		Context.Next()
	}
}

func Set(key string, value any) {
	Values[key] = value
}
func Get(key string) any {
	return Values[key]
}
func Del(key string) {
	delete(Values, key)
}
func Save() {
	if Values != nil && len(Values) > 0 {
		data, err := Session.Serialize(Values)
		if err != nil {
			fmt.Println(err)
		} else {
			Session.RedisStore.Set(StoreKey, data, 0)
		}
	}
}
func Destroy() {
	Values = nil
	Session.RedisStore.Del(StoreKey)
}
