// Package service 业务基础设施。当前提供设备通信（MQTT）与锁控制。
// 硬件通信按 MQTT 调用资料_增强版.docx：下行发布到 device/{IMEI}/cmd，上行订阅 device/+/report|event|alarm|heartbeat。
package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/redis/go-redis/v9"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
	"caicai-go/protocol"
)

var (
	mqttClient mqtt.Client
	rdb        *redis.Client
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
}

// onDeviceMessage 处理上行设备数据：解析帧并按 cmd 缓存到 Redis（供查询用）。
func onDeviceMessage(client mqtt.Client, msg mqtt.Message) {
	deviceID, ctl, cmd, data, err := protocol.Decode(msg.Payload())
	if err != nil {
		logger.Mylog.Warn().Msgf("设备帧解析失败, topic: %s, err: %v", msg.Topic(), err)
		return
	}
	_ = ctl

	if rdb != nil {
		key := "device:" + deviceID + ":" + strconv.Itoa(int(cmd))
		rdb.Set(context.Background(), key, data, 10*time.Minute)
	}
	logger.Mylog.Info().Msgf("收到设备数据, topic: %s, device: %s, cmd: 0x%02X, data: %q", msg.Topic(), deviceID, cmd, data)
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
			return
		}
	}

	// 2. 未命中缓存，查库确认是否已存在（如服务重启后缓存丢失）
	var exist model.TabGateway
	if err := conf.Db.Where("mac = ?", req.Imei).First(&exist).Error; err == nil {
		if rdb != nil {
			rdb.SAdd(ctx, gatewayRegisteredKey, req.Imei)
		}
		return // 已注册，回填缓存后跳过
	}

	// 3. 未注册，入库 + 写缓存
	if err := conf.Db.Exec("INSERT INTO tab_gateway(mac, iccid) VALUES (?, ?)", req.Imei, req.Iccid).Error; err != nil {
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

// SendCommand 下发指令到指定设备（topic = device/{imei}/cmd）。
func SendCommand(imei string, cmd, ctl byte, data []byte) bool {
	topic := deviceTopicPrefix + imei + "/cmd"
	if !publish(topic, protocol.EncodeCmd(cmd, ctl, data)) {
		return false
	}
	logger.Mylog.Info().Msgf("已下发指令, topic: %s, cmd: 0x%02X", topic, cmd)
	return true
}

// SendHex 下发原始 hex 指令到指定设备（不经过 EncodeCmd，直接发原始字节）。
func SendHex(imei, hexStr string) bool {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		logger.Mylog.Err(err).Msgf("hex 解析失败: %s", hexStr)
		return false
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
	var lock model.LockTbl
	if err := conf.Db.Raw(
		"SELECT l.* FROM order_tbl o LEFT JOIN tab_parking_spaces ps ON o.spaces_id = ps.id LEFT JOIN lock_tbl l ON ps.lock_id = l.lockid WHERE o.orderid = ?",
		orderID,
	).Scan(&lock).Error; err != nil {
		return nil, err
	}
	if lock.Lockid == "" {
		return nil, nil
	}

	product, err := objects.TabProduct.WithContext(context.Background()).Where(objects.TabProduct.Pid.Eq(lock.ProductID)).First()
	if err != nil {
		return nil, err
	}
	gateway, err := objects.TabGateway.WithContext(context.Background()).Where(objects.TabGateway.ID.Eq(product.GatewayID)).First()
	if err != nil {
		return nil, err
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

	SendCommand(lg.deviceID(), protocol.OpClosePlaceLock, protocol.DirectionDefault, nil)
	logger.Mylog.Info().Msgf("已下发关锁指令, orderId: %s", orderID)
	return true
}

// StartLock 开启车位锁（对应 Java OrderServiceImpl.startLock）。
func StartLock(orderID string) bool {
	lg, err := getLockByOrder(orderID)
	if err != nil || lg == nil {
		return false
	}

	var count int64
	conf.Db.Model(&model.OrderTbl{}).
		Where("lockid = ? AND orderid <> ? AND state IN ('使用中','进行中')", lg.lock.Lockid, orderID).
		Count(&count)
	if count > 0 {
		return false
	}

	SendCommand(lg.deviceID(), protocol.OpOpenPlaceLock, protocol.DirectionDefault, nil)

	now := time.Now()
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", lg.lock.Placeid).Update("state", "使用中")
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
		"state":      "使用中",
		"begin_time": now,
	})
	logger.Mylog.Info().Msgf("已下发开锁指令, orderId: %s", orderID)
	return true
}
