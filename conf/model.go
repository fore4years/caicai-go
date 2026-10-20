package conf

type MqttConf struct {
	Broker   string
	Port     int16
	Qos      int8
	Username string
	Password string
}

type RedisConf struct {
	Host     string
	Username string
	Password string
	Port     int16
	Dbnumber int8
}

type MysqlConf struct {
	Host     string
	Password string
	Username string
	Port     int16
	Dbname   string
}

type AppConf struct {
	Port int16
}

type TcpConf struct {
	Port int16
}

type WxpayConf struct {
	APIv3Key       string `mapstructure:"api-v3-key"`
	Appid          string `mapstructure:"appid"`
	MchID          string `mapstructure:"mch-id"`
	MchSerialNo    string `mapstructure:"mch-serial-no"`
	ServiceID      string `mapstructure:"service-id"`
	ServiceName    string `mapstructure:"service-name"`
	NotifyDomain   string `mapstructure:"notify-domain"`
	PrivateKeyPath string `mapstructure:"private-key-path"`
}
