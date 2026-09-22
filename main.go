package main

import (
	"fmt"
	"io"
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

	// 初始化设备通信基础设施（MQTT 下发 + 订阅上行主题 + Redis）
	service.InitDevice(conf.Cfg.Mqtt, conf.Cfg.Redis)

	gin.DefaultWriter = io.Discard
	r := gin.New()
	// 全局日志 + 鉴权（对齐 Java WebAppConfigurer 的 PathInterceptor + LoginInterceptor）
	r.Use(gin.Recovery(), middleware.MiddleLog(logger.Mylog), middleware.Auth())

	router.RouterInit(r)

	logger.Mylog.Info().Msg(fmt.Sprintf("应用启动成功, 监听端口: %d", conf.Cfg.App.Port))
	r.Run(serverPort)
}
