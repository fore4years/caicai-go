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
	Host         string
	Password     string
	Username     string
	Port         int16
	Dbname       string
	MaxOpenConns int `mapstructure:"max-open-conns"` // 连接池最大连接数（0 = 使用默认值）
	MaxIdleConns int `mapstructure:"max-idle-conns"` // 连接池最大空闲连接数（0 = 使用默认值）
}

type AppConf struct {
	Port    int16
	Env     string // 运行环境标识（prod/local），仅用于日志与后续按环境切换
	Migrate bool   // 是否在启动时执行 AutoMigrate（生产环境默认 false）
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
