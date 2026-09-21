// Package service 业务基础设施。当前提供锁控制（开锁/关锁），
// 下发通道为 MQTT（对齐 Java 中 DirectiveService -> Session -> Channel 的职责）。
package service

import (
	"context"
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

// InitLockControl 初始化锁控制依赖的 MQTT 与 Redis 客户端。
// 需在 main 中、数据库初始化之后调用。
func InitLockControl(mqcfg conf.MqttConf, rcfg conf.RedisConf) {
	// MQTT
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://" + mqcfg.Broker + ":" + strconv.Itoa(int(mqcfg.Port)))
	opts.SetClientID("caicai-go-lock")
	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Mylog.Err(token.Error()).Msg("MQTT 连接失败")
	} else {
		logger.Mylog.Info().Msg("MQTT 连接成功")
	}

	// Redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     rcfg.Host + ":" + strconv.Itoa(int(rcfg.Port)),
		Password: rcfg.Password,
		DB:       int(rcfg.Dbnumber),
	})
}

// lockGateway 锁及其关联的产品/网关信息。
type lockGateway struct {
	lock    model.LockTbl
	product model.TabProduct
	gateway model.TabGateway
}

// nid 返回下发指令用的设备标识：4G 用 IMEI（8 字节），LoRa 用 NID（4 字节）。
func (lg *lockGateway) nid() string {
	if len(lg.gateway.Mac) > 12 {
		return lg.product.Imei
	}
	return lg.product.Nid
}

// topic 返回 MQTT 下发主题，格式 /caicai/{SCC|SCL}/{Imei}。
// 型号前缀取 product.pid 前三位（SCC/SCL），4G 用 IMEI，LoRa 回退 NID。
func (lg *lockGateway) topic() string {
	prefix := "SCC"
	if len(lg.product.Pid) >= 3 {
		prefix = lg.product.Pid[:3]
	}
	id := lg.product.Imei
	if id == "" {
		id = lg.product.Nid
	}
	return "/caicai/" + prefix + "/" + id
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

	// 单探头只看车位锁距离；双探头任一检测到遮挡则不能关锁
	hasObstacle := (distance < 1600 && distance != 0) ||
		(chargingPileDistance != nil && *chargingPileDistance < 3000 && *chargingPileDistance != 0)
	if hasObstacle {
		logger.Mylog.Warn().Msgf("检测到遮挡物, 无法关锁, orderId: %s", orderID)
		return false
	}

	// 下发关锁指令
	payload := protocol.Encode(lg.nid(), protocol.OpClosePlaceLock, protocol.DirectionDefault, nil)
	mqttClient.Publish(lg.topic(), byte(conf.Cfg.Mqtt.Qos), false, payload)
	logger.Mylog.Info().Msgf("已下发关锁指令, topic: %s, nid: %s", lg.topic(), lg.nid())
	return true
}

// StartLock 开启车位锁（对应 Java OrderServiceImpl.startLock）。
// 注意：Java 通过 futureSendMessage 同步等待设备应答，MQTT 下暂简化为异步下发。
func StartLock(orderID string) bool {
	lg, err := getLockByOrder(orderID)
	if err != nil || lg == nil {
		return false
	}

	// 占用检查：该锁是否已有其他进行中的订单
	var count int64
	conf.Db.Model(&model.OrderTbl{}).
		Where("lockid = ? AND orderid <> ? AND state IN ('使用中','进行中')", lg.lock.Lockid, orderID).
		Count(&count)
	if count > 0 {
		return false
	}

	// 下发开锁指令
	payload := protocol.Encode(lg.nid(), protocol.OpOpenPlaceLock, protocol.DirectionDefault, nil)
	mqttClient.Publish(lg.topic(), byte(conf.Cfg.Mqtt.Qos), false, payload)

	// 更新车位与订单状态
	now := time.Now()
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", lg.lock.Placeid).Update("state", "使用中")
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
		"state":      "使用中",
		"begin_time": now,
	})
	logger.Mylog.Info().Msgf("已下发开锁指令, orderId: %s", orderID)
	return true
}
