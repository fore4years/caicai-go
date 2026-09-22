package conf

import (
	"caicai-go/logger"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config struct {
	Mqtt  MqttConf  `mapstructure:"mqtt"`
	Redis RedisConf `mapstructure:"redis"`
	Mysql MysqlConf `mapstructure:"mysql"`
	App   AppConf   `mapstructure:"app"`
}

var (
	Cfg *Config
	Db  *gorm.DB
)

func init() {
	logger.Mylog.Info().Msg("配置初始化...")
	viper.SetConfigName("conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		// log.Fatalf("配置文件读取异常: %v", err)
		logger.Mylog.Fatal().Err(err).Msg("配置文件读取异常")
	}

	Cfg = &Config{}
	if err := viper.Unmarshal(Cfg); err != nil {
		logger.Mylog.Fatal().Err(err).Msg("配置反序列化异常")
	}
	logger.Mylog.Info().Msg("配置加载成功")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		// log.Printf("配置变更: %s，重新加载", e.Name)
		logger.Mylog.Info().Msg("开始更新配置")

		var newCfg Config
		if err := viper.Unmarshal(&newCfg); err != nil {
			logger.Mylog.Fatal().Err(err).Msg("配置文件更新异常")
			return
		}
		Cfg = &newCfg

		// log.Printf("配置更新完成: %+v", Cfg)
		logger.Mylog.Info().Msg("配置更新成功")
	})

}

func Dbconnection(dbconf MysqlConf) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbconf.Username, dbconf.Password, dbconf.Host, dbconf.Port, dbconf.Dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("数据库连接异常: %w", err)
	}
	option, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取连接池失败: %w", err)
	}
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量。
	option.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	option.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了可以重新使用连接的最大时间。
	option.SetConnMaxLifetime(time.Hour)
	// SetConnMaxIdleTime 设置空闲连接的最大空闲时间。
	option.SetConnMaxIdleTime(30 * time.Minute)

	// 启动时 Ping 验证连接，重试 3 次（间隔 2s）。
	var pingErr error
	for i := range 3 {
		if pingErr = option.Ping(); pingErr == nil {
			break
		}
		logger.Mylog.Warn().Err(pingErr).Msgf("MySQL Ping 失败，重试 %d/3", i+1)
		time.Sleep(2 * time.Second)
	}
	if pingErr != nil {
		return fmt.Errorf("数据库连接异常: %w", pingErr)
	}

	Db = db

	// 后台健康检查：每 30s Ping 一次，记录断连/恢复（database/sql 会自动重建连接）。
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		healthy := true
		for range ticker.C {
			if err := option.Ping(); err != nil {
				if healthy {
					logger.Mylog.Warn().Err(err).Msg("MySQL 连接断开")
					healthy = false
				}
			} else if !healthy {
				logger.Mylog.Info().Msg("MySQL 连接恢复")
				healthy = true
			}
		}
	}()

	return nil
}
