package conf

import (
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
	log.Printf("开始读取配置")
	viper.SetConfigName("conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("配置文件读取异常: %v", err)
	}

	Cfg = &Config{}
	if err := viper.Unmarshal(Cfg); err != nil {
		log.Fatalf("配置解析异常: %v", err)
	}
	log.Printf("配置加载完成: %+v", Cfg)

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("配置变更: %s，重新加载", e.Name)

		var newCfg Config
		if err := viper.Unmarshal(&newCfg); err != nil {
			log.Printf("配置重新加载失败: %v", err)
			return
		}
		Cfg = &newCfg

		log.Printf("配置更新完成: %+v", Cfg)
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
