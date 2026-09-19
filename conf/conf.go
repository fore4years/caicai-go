package conf

import (
	"caicai-go/logger"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Mqtt  MqttConf  `mapstructure:"mqtt"`
	Redis RedisConf `mapstructure:"redis"`
	Mysql MysqlConf `mapstructure:"mysql"`
	App   AppConf   `mapstructure:"app"`
}

var (
	Cfg *Config
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
		log.Printf("mqtt链接异常: %v", token.Error())
	}

	topic := "xxxtopicxxx"
	message := "xxxxxx"

	token := client.Publish(topic, byte(mqcfg.Qos), false, message)
	token.Wait()
}
