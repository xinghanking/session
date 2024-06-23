# session
xinghanking/session包是一个实现与PHP程序共用redis存储session数据的包。
php的session除了文件存储这种方式，还可以使用redis存储。
对于php如何通过在php.ini中设置使用redis做session存储，在此略过。
本包暂时只支持php.ini的serialize_handler设置为php_serialize的情况。
配合gin的中间件使用，可实现与php程序共同session的功能

## 安装
```sh
go get github.com/xinghanking/session
```
## 前置
```sh
go get github.com/gin-gonic/gin
go get github.com/go-redis/redis
```

## 使用
```go
router := gin.Default()
Client = redis.NewClient(&redis.Options{
	Addr:     127.0.0.1:6379, //redis服务器地址
	Password: "", //redis密码
	DB:       0,
	//…………
})
router.Use(redis_session.Init(&redis_session.Options{
    RedisStore:     Client,              //必传
    SessionName：   "PHPSESSID",         //存储session_id的cookie名，默认值："PHPSESSID"
    RedisKeyPrefix："PHPREDIS_SESSION:", //session数据存储在redis中的KEY前缀名，默认值：" PHPREDIS_SESSION:"
    MaxAge:         86400,               //保存session_id值的cookie最大生存时间，单位：秒，默认值：86400
    Expiration:     0,                   //redis中保存session数据的时长，默认值：0
}))
router.GET("/", func(context *gin.Context){
    userName := redis_session.Get("USER_NAME")  //获取seession值
    redis_session.Set("IS_LOGIN", true)         //设置session
    redis_session.Del("PWD")                    //删除session
    redis_session.Destroy()                     //销毁当前session
})
```