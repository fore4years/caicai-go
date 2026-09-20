package conf

import (
	"caicai-go/logger"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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

func MqttConnection(mqcfg MqttConf) {
	option := mqtt.NewClientOptions().AddBroker(mqcfg.Broker)
	client := mqtt.NewClient(option)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logger.Mylog.Fatal().Err(token.Error()).Msg("MQTT连接异常")
	}

	topic := "xxxtopicxxx"
	message := "xxxxxx"

	token := client.Publish(topic, byte(mqcfg.Qos), false, message)
	token.Wait()
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

	Db = db
	return nil
}
