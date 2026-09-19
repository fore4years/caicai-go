package conf

type MqttConf struct {
	Broker string
	Port   int16
	Qos    int8
}

type RedisConf struct {
	Username string
	Port     int16
	host     string
	password string
	Dbnumber int8
}

type MysqlConf struct {
	Host     string
	Password string
	Username string
	Port     int16
}

type AppConf struct {
	Port int16
}
