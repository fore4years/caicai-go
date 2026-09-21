package main

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/middleware"
	"caicai-go/objects"
	"caicai-go/router"
	"caicai-go/service"
)

var serverPort = ":" + strconv.Itoa(int(conf.Cfg.App.Port))

func main() {
	// 初始化数据库连接
	if err := conf.Dbconnection(conf.Cfg.Mysql); err != nil {
		logger.Mylog.Fatal().Err(err).Msg("数据库初始化失败")
	}
	// 初始化 gorm gen 查询对象
	objects.SetDefault(conf.Db)

	// 初始化锁控制基础设施（MQTT 下发 + Redis 距离数据）
	service.InitLockControl(conf.Cfg.Mqtt, conf.Cfg.Redis)

	r := gin.New()
	// 全局日志 + 鉴权（对齐 Java WebAppConfigurer 的 PathInterceptor + LoginInterceptor）
	r.Use(gin.Recovery(), middleware.MiddleLog(logger.Mylog), middleware.Auth())

	router.RouterInit(r)

	r.Run(serverPort)
}
