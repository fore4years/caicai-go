// Package service 业务基础设施。当前提供设备通信（MQTT）与锁控制。
// 硬件通信按 MQTT 调用资料_增强版.docx：下行发布到 device/{IMEI}/cmd，上行订阅 device/+/report|event|alarm|heartbeat。
package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/redis/go-redis/v9"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/protocol"
	"caicai-go/repository"
)

var (
	mqttClient     mqtt.Client
	rdb            *redis.Client
	orderDeviceMap map[string]string //用于设备反查orderid
)

// deviceTopicPrefix 设备主题前缀。
const deviceTopicPrefix = "device/"

// gatewayRegisteredKey 网关注册缓存（Redis SET，存放已注册网关的 imei/mac）。
const gatewayRegisteredKey = "gateway:registered"

// InitDevice 初始化设备通信依赖的 MQTT 与 Redis 客户端。
// 连接 broker，并订阅所有设备的上行主题（report/event/alarm/heartbeat）。
// MQTT 开启自动重连，并在每次（重）连接成功后重新订阅（避免重连后订阅丢失）。
func InitDevice(mqcfg conf.MqttConf, rcfg conf.RedisConf) {
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://" + mqcfg.Broker + ":" + strconv.Itoa(int(mqcfg.Port)))
	opts.SetClientID("caicai-go-device")
	opts.SetKeepAlive(60 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)
	if mqcfg.Username != "" {
		opts.SetUsername(mqcfg.Username)
	}
	if mqcfg.Password != "" {
		opts.SetPassword(mqcfg.Password)
	}

	// 连接成功（含重连成功）时重新订阅上行主题。
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Mylog.Info().Msg("MQTT 连接成功")
		for _, sub := range []string{"report", "event", "alarm"} {
			topic := deviceTopicPrefix + "+/" + sub
			if token := client.Subscribe(topic, byte(mqcfg.Qos), onDeviceMessage); token.Wait() && token.Error() != nil {
				logger.Mylog.Err(token.Error()).Msgf("订阅主题失败: %s", topic)
			} else {
				logger.Mylog.Info().Msgf("已订阅主题: %s", topic)
			}
		}
		// 心跳主题（JSON 载荷）：网关注册，首次心跳即注册
		heartbeatTopic := deviceTopicPrefix + "+/heartbeat"
		if token := client.Subscribe(heartbeatTopic, byte(mqcfg.Qos), onHeartbeat); token.Wait() && token.Error() != nil {
			logger.Mylog.Err(token.Error()).Msgf("订阅主题失败: %s", heartbeatTopic)
		} else {
			logger.Mylog.Info().Msgf("已订阅主题: %s", heartbeatTopic)
		}
		//
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.Mylog.Warn().Err(err).Msg("MQTT 连接断开，等待重连")
	})
	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		logger.Mylog.Info().Msg("MQTT 重连中...")
	})

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Mylog.Err(token.Error()).Msg("MQTT 连接失败")
	}

	// Redis（go-redis 按需自动重连，这里补启动探测）
	rdb = redis.NewClient(&redis.Options{
		Addr:     rcfg.Host + ":" + strconv.Itoa(int(rcfg.Port)),
		Password: rcfg.Password,
		DB:       int(rcfg.Dbnumber),
	})
	for i := range 3 {
		if err := rdb.Ping(context.Background()).Err(); err == nil {
			logger.Mylog.Info().Msg("Redis 连接成功")
			break
		} else if i == 2 {
			logger.Mylog.Warn().Err(err).Msg("Redis 连接失败")
		} else {
			time.Sleep(2 * time.Second)
		}
	}

	orderDeviceMap = make(map[string]string, 200)
}

// onDeviceMessage 处理 MQTT 上行设备数据（topic = device/{imei}/report|event|alarm）。
// MQTT 上报为 JSON（如 {"event":"relay_on","channel":1,"energy_kwh":...}）；非 JSON 时回退按二进制帧解析（兼容）。
func onDeviceMessage(client mqtt.Client, msg mqtt.Message) {
	payload := msg.Payload()
	var report mqttDeviceReport
	if err := json.Unmarshal(payload, &report); err == nil && report.Channel > 0 {
		handleMqttReport(extractDeviceFromTopic(msg.Topic()), report)
		return
	}
	processDeviceFrame(payload, msg.Topic())
}

// processDeviceFrame 解析上行设备帧（MQTT 或 TCP 均复用），按 cmd 缓存 data 到 Redis（供查询用）。
func processDeviceFrame(frame []byte, source string) {
	deviceID, ctl, cmd, data, err := protocol.Decode(frame)
	if err != nil {
		logger.Mylog.Warn().Msgf("设备帧解析失败, source: %s, err: %v", source, err)
		return
	}
	// 业务联动：按 cmd 缓存到对应用途的 Redis 键（功率/电流/电压/枪状态等）
	handleDeviceUplink(deviceID, ctl, cmd, data)

	if rdb != nil {
		key := "device:" + deviceID + ":" + strconv.Itoa(int(cmd))
		rdb.Set(context.Background(), key, data, 10*time.Minute)
	}
	logger.Mylog.Info().Msgf("收到设备数据, source: %s, device: %s, cmd: 0x%02X, data: %q", source, deviceID, cmd, data)
}

// onHeartbeat 心跳处理（MQTT JSON 约定：首次心跳即网关注册）。
// 主题 device/{imei}/heartbeat，载荷 {"imei":"xx","iccid":"xx"}。
// tab_gateway 已存在该 imei（作为 mac）则跳过；心跳一分钟一次，仅首次注册。
func onHeartbeat(client mqtt.Client, msg mqtt.Message) {
	var req struct {
		Imei  string `json:"imei"`
		Iccid string `json:"iccid"`
	}
	// imei == mac == 网关id
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		logger.Mylog.Warn().Msgf("网关注册 JSON 解析失败, topic: %s, err: %v", msg.Topic(), err)
		return
	}
	if req.Imei == "" {
		logger.Mylog.Warn().Msgf("网关注册缺少 imei, topic: %s", msg.Topic())
		return
	}

	ctx := context.Background()
	// 1. 先查 Redis 缓存，已注册则直接跳过，避免每次心跳查库
	if rdb != nil {
		if ok, err := rdb.SIsMember(ctx, gatewayRegisteredKey, req.Imei).Result(); err == nil && ok {
			logger.Mylog.Info().Msgf("网关IMEI %s 心跳: %s", req.Imei, string(msg.Payload()))
			return
		}
	}

	// 2. 未命中缓存，查库确认是否已存在（如服务重启后缓存丢失）
	if repository.GatewayByMac(req.Imei) != nil {
		if rdb != nil {
			rdb.SAdd(ctx, gatewayRegisteredKey, req.Imei)
		}
		return // 已注册，回填缓存后跳过
	}

	// 3. 未注册，入库 + 写缓存
	if err := repository.InsertGateway(req.Imei, req.Iccid); err != nil {
		logger.Mylog.Err(err).Msgf("网关注册入库失败: imei=%s, iccid=%s", req.Imei, req.Iccid)
		return
	}
	if rdb != nil {
		rdb.SAdd(ctx, gatewayRegisteredKey, req.Imei)
	}
	logger.Mylog.Info().Msgf("网关注册成功: imei=%s, iccid=%s", req.Imei, req.Iccid)
}

// publish 发布原始 payload 到指定 topic。
func publish(topic string, payload []byte) bool {
	if mqttClient == nil || !mqttClient.IsConnected() {
		logger.Mylog.Warn().Msg("MQTT 未连接，无法下发指令")
		return false
	}
	if token := mqttClient.Publish(topic, byte(conf.Cfg.Mqtt.Qos), false, payload); token.Wait() && token.Error() != nil {
		logger.Mylog.Err(token.Error()).Msgf("下发失败, topic: %s", topic)
		return false
	}
	return true
}

// SendCommand 下发指令到指定设备。
// 路由规则：设备已建立 TCP 连接（老设备）→ 走二进制帧；否则 → 走 MQTT JSON。
// data 为单字节数据（如插座号/占空比）；无数据传 -1。
func SendCommand(imei string, cmd int, data int) bool {
	return sendCommand(imei, cmd, intDataToBytes(data))
}

// SendCommandBytes 下发带多字节数据的指令（如时段/网关地址等）。路由规则同 SendCommand。
func SendCommandBytes(imei string, cmd int, data []byte) bool {
	return sendCommand(imei, cmd, data)
}

// sendCommand 核心下发：优先 TCP 二进制帧，否则 MQTT JSON。
func sendCommand(imei string, cmd int, data []byte) bool {
	// 老设备：走 TCP 二进制协议（连接即设备身份，帧不带 nid 前缀，4G 帧）
	if _, ok := tcpConns.Load(imei); ok {
		frame := protocol.Encode(imei, byte(cmd), protocol.DirectionDefault, data)
		if SendTCP(imei, frame) {
			logger.Mylog.Info().Msgf("已通过 TCP 下发指令, device: %s, cmd: 0x%02X", imei, cmd)
			return true
		}
	}

	// 新设备：MQTT JSON
	body := CmdBody{
		Ctl:  0,
		Cmd:  cmd,
		Data: dataInt(data),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		logger.Mylog.Err(err).Msg("指令 JSON 序列化失败")
		return false
	}
	topic := deviceTopicPrefix + imei + "/cmd"
	if !publish(topic, payload) {
		return false
	}
	logger.Mylog.Info().Msgf("已下发指令, topic: %s, payload: %s", topic, payload)
	return true
}

// intDataToBytes 将单字节数据（int）转为字节切片；负数表示无数据。
func intDataToBytes(data int) []byte {
	if data < 0 {
		return []byte{}
	}
	return []byte{byte(data)}
}

// dataInt 将字节数据转回 int（JSON 载荷用）；空数据返回 -1，多字节取首字节。
func dataInt(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	return int(data[0])
}

// SendHex 下发原始 hex 指令到指定设备（不经过 EncodeCmd，直接发原始字节）。
func SendHex(imei, hexStr string) bool {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		logger.Mylog.Err(err).Msgf("hex 解析失败: %s", hexStr)
		return false
	}
	// 老设备走 TCP：直接发原始字节
	if _, ok := tcpConns.Load(imei); ok {
		return SendTCP(imei, data)
	}
	return publish(deviceTopicPrefix+imei+"/cmd", data)
}

// SetRedisValue 设置 Redis 键值（对齐 Java DirectiveController.setRedisValue 测试接口）。
func SetRedisValue(key, value string) {
	if rdb == nil {
		return
	}
	rdb.Set(context.Background(), key, value, 0)
}

// GetRedisValue 读取 Redis 字符串键值；未命中或客户端未初始化时返回空串。
func GetRedisValue(key string) string {
	if rdb == nil {
		return ""
	}
	v, err := rdb.Get(context.Background(), key).Result()
	if err != nil {
		return ""
	}
	return v
}

// DelRedisValue 删除 Redis 键（对齐 Java redisTemplate.delete）。
func DelRedisValue(key string) {
	if rdb == nil {
		return
	}
	rdb.Del(context.Background(), key)
}

var orderDeviceMu sync.RWMutex

// SetMap 设置device -> orderid
// key的格式: imei::channel
func SetMap(key, value string) {
	orderDeviceMu.Lock()
	defer orderDeviceMu.Unlock()
	orderDeviceMap[key] = value
}

func GetMap(key string) (string, bool) {
	orderDeviceMu.RLock()
	defer orderDeviceMu.RUnlock()
	v, ok := orderDeviceMap[key]
	return v, ok
}

func DeleteMap(key string) {
	orderDeviceMu.Lock()
	defer orderDeviceMu.Unlock()
	delete(orderDeviceMap, key)
}

// lockGateway 锁及其关联的产品/网关信息。
type lockGateway struct {
	lock    model.LockTbl
	product model.TabProduct
	gateway model.TabGateway
}

// deviceID 返回主题用设备标识：优先 4G IMEI，否则 LoRa NID。
func (lg *lockGateway) deviceID() string {
	if lg.product.Imei != "" {
		return lg.product.Imei
	}
	return lg.product.Nid
}

// getLockByOrder 根据订单查询锁及其产品/网关信息（对应 Java LockService.getByOrder + product + gateway）。
func getLockByOrder(orderID string) (*lockGateway, error) {
	lock, product, gateway, err := repository.LockProductGatewayByOrder(orderID)
	if err != nil {
		return nil, err
	}
	if lock.Lockid == "" {
		return nil, nil
	}
	return &lockGateway{lock: lock, product: *product, gateway: *gateway}, nil
}

// StopLock 关闭车位锁（对应 Java OrderServiceImpl.stopLock）。
// 读取 Redis 中的距离数据，判断上方无遮挡后通过 MQTT 下发关锁指令。
func StopLock(orderID string) bool {
	lg, err := getLockByOrder(orderID)
	if err != nil || lg == nil {
		return true
	}

	ctx := context.Background()
	distanceStr, err := rdb.Get(ctx, "placeLockDistance:orderId:"+orderID).Result()
	if err != nil {
		logger.Mylog.Warn().Msgf("未获取到车位锁距离数据, orderId: %s", orderID)
		return false
	}
	chargingPile, _ := rdb.Get(ctx, "chargingPileDetection:orderId:"+orderID).Result()

	distance, err := strconv.Atoi(distanceStr)
	if err != nil {
		logger.Mylog.Err(err).Msgf("解析距离数据失败, distanceStr: %s, orderId: %s", distanceStr, orderID)
		return false
	}

	var chargingPileDistance *int
	if chargingPile != "" {
		if v, e := strconv.Atoi(chargingPile); e == nil {
			chargingPileDistance = &v
		}
	}

	hasObstacle := (distance < 1600 && distance != 0) ||
		(chargingPileDistance != nil && *chargingPileDistance < 3000 && *chargingPileDistance != 0)
	if hasObstacle {
		logger.Mylog.Warn().Msgf("检测到遮挡物, 无法关锁, orderId: %s", orderID)
		return false
	}

	//SendCommand(lg.deviceID(), protocol.OpClosePlaceLock, protocol.DirectionDefault, nil)
	logger.Mylog.Info().Msgf("已下发关锁指令, orderId: %s", orderID)
	return true
}

// StartLock 开启车位锁（对应 Java OrderServiceImpl.startLock）。
func StartLock(orderID string) bool {
	lg, err := getLockByOrder(orderID)
	if err != nil || lg == nil {
		return false
	}

	if repository.CountActiveOrdersByLock(lg.lock.Lockid, orderID) > 0 {
		return false
	}

	//SendCommand(lg.deviceID(), protocol.OpOpenPlaceLock, protocol.DirectionDefault, nil)

	repository.UpdatePlaceState(lg.lock.Placeid, "使用中")
	repository.StartOrder(orderID, time.Now())
	logger.Mylog.Info().Msgf("已下发开锁指令, orderId: %s", orderID)
	return true
}
