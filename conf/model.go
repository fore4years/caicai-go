package conf

type MqttConf struct {
	Broker string
	Port   int16
	Qos    int8
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
