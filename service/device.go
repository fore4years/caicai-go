// Package service 业务基础设施。当前提供设备通信（MQTT）与锁控制。
// 硬件通信按 MQTT 调用资料_增强版.docx：下行发布到 device/{IMEI}/cmd，上行订阅 device/+/report|event|alarm|heartbeat。
package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
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

	heartTicker = make(map[string]*time.Timer) // 心跳时钟
	timerMutex  sync.Mutex
)

// deviceTopicPrefix 设备主题前缀。
const deviceTopicPrefix = "device/"

// gatewayRegisteredKey 网关注册缓存（Redis SET，存放已注册网关的 imei/mac）。
const gatewayRegisteredKey = "gateway:registered"

const (
	eventGun    = "gun_status"
	eventEnergy = "energy_status"
)

const (
	workNum     = 10   // mqtt处理的goroutine树
	queueLength = 1024 // mqtt接收队列changdu

	liveTime = 2 * time.Minute
)

type job struct {
	topic   string
	payload []byte
	qos     byte
	retain  bool
}

var topicMessageHandler = map[string]func(job){
	"report": onDeviceMessage,
	"event":  onEventMessage,
	"alarm":  onDeviceMessage,
	"heart":  onHeartbeat,
}

func onAnyMessage(j job) {
	suffix := j.topic
	if i := strings.LastIndex(suffix, "/"); i >= 0 {
		suffix = suffix[i+1:]
	}
	h, ok := topicMessageHandler[suffix]
	if !ok {
		logger.Mylog.Warn().Msgf("未注册的主题: %s", j.topic)
		return
	}
	h(j)
}

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

	jobs := make(chan job, queueLength)
	var wg sync.WaitGroup
	for i := range workNum {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range jobs {
				onAnyMessage(j)
			}
		}(i)
	}
	// 连接成功（含重连成功）时重新订阅上行主题。
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Mylog.Info().Msg("MQTT 连接成功")
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

	handler := func(client mqtt.Client, msg mqtt.Message) {
		j := job{
			topic:   msg.Topic(),
			payload: append([]byte(nil), msg.Payload()...),
			qos:     msg.Qos(),
			retain:  msg.Retained(),
		}

		select {
		case jobs <- j:
		default:
			logger.Mylog.Warn().Msgf("[MQTT] 工作队列已满, Topic: %s, Len: %d", j.topic, len(j.payload))
		}
	}

	filters := make(map[string]byte, len(topicMessageHandler))
	for suffix := range topicMessageHandler {
		filters[deviceTopicPrefix+"+/"+suffix] = byte(mqcfg.Qos)
	}

	// 一次往返批量订阅
	if token := mqttClient.SubscribeMultiple(filters, handler); token.Wait() && token.Error() != nil {
		logger.Mylog.Err(token.Error()).Msg("批量订阅失败，回退为逐个订阅")
		for topic, qos := range filters {
			if t := mqttClient.Subscribe(topic, qos, handler); t.Wait() && t.Error() != nil {
				logger.Mylog.Err(t.Error()).Msgf("订阅主题失败: %s", topic)
			} else {
				logger.Mylog.Info().Msgf("已订阅主题: %s", topic)
			}
		}
		return
	}
	for topic := range filters {
		logger.Mylog.Info().Msgf("已订阅主题: %s", topic)
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
// MQTT 上行均为 JSON，不做二进制帧解析（二进制帧仅 TCP 通道使用）。
func onDeviceMessage(j job) {
	payload := j.payload
	var report mqttDeviceReport
	if err := json.Unmarshal(payload, &report); err != nil {
		logger.Mylog.Warn().Msgf("MQTT 上报 JSON 解析失败, topic=%s, err=%v", j.topic, err)
		return
	}
	handleMqttReport(extractDeviceFromTopic(j.topic), report)
}

// processDeviceFrame 解析上行设备二进制帧（仅 TCP 通道使用），按 cmd 缓存 data 到 Redis（供查询用）。
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
		rdb.Set(context.Background(), key, data, 5*time.Second)
	}
	bcdData := bcdToInt(data)
	if cmd == protocol.OpGetPower && bcdData == 0 {
		return
	}
	logger.Mylog.Info().Msgf("收到设备数据, source: %s, device: %s, cmd: 0x%02X, dataHex: %X, bcd: %d", source, deviceID, cmd, data, bcdData)
}

// onHeartbeat 心跳处理（MQTT JSON 约定：首次心跳即网关注册）。
// 主题 device/{imei}/heartbeat，载荷 {"imei":"xx","iccid":"xx"}。
// imei == mac == 网关id。每次心跳都会刷新「离线检测」定时器，liveTime 内无心跳视为设备离线。
func onHeartbeat(j job) {
	var req struct {
		Imei  string `json:"imei"`
		Iccid string `json:"iccid"`
	}
	if err := json.Unmarshal(j.payload, &req); err != nil {
		logger.Mylog.Warn().Msgf("网关注册 JSON 解析失败, topic: %s, err: %v", j.topic, err)
		return
	}
	if req.Imei == "" {
		logger.Mylog.Warn().Msgf("网关注册缺少 imei, topic: %s", j.topic)
		return
	}

	// 每次心跳刷新离线检测定时器（无论是否已注册，都要重置，否则定时器会误判已在线设备为离线）
	refreshHeartbeatTimer(req.Imei)

	ctx := context.Background()
	// 1. 先查 Redis 缓存，已注册则直接跳过，避免每次心跳查库
	if rdb != nil {
		if ok, err := rdb.SIsMember(ctx, gatewayRegisteredKey, req.Imei).Result(); err == nil && ok {
			logger.Mylog.Info().Msgf("网关IMEI %s 心跳: %s", req.Imei, string(j.payload))
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

// refreshHeartbeatTimer 刷新设备离线检测定时器：每次心跳重置，超时后触发 handleDeviceOffline。
func refreshHeartbeatTimer(imei string) {
	timerMutex.Lock()
	defer timerMutex.Unlock()
	if t, ok := heartTicker[imei]; ok {
		t.Stop()
	}
	timer := time.AfterFunc(liveTime, func() {
		handleDeviceOffline(imei)

		timerMutex.Lock()
		delete(heartTicker, imei)
		timerMutex.Unlock()
	})
	heartTicker[imei] = timer
}

// handleDeviceOffline 设备心跳超时（离线）后的清理：关闭该设备上进行中的订单，并释放相关资源状态。
// 扫描的业务状态：
//   - 二轮车充电订单：结算（离线不关闸，结算事务内释放插座），并额外兜底释放插座
//   - 驿享充电 / 邻享订单：按最后一次上报电量结算扣款（不含超时占位费）
//   - 踩踩停车订单：置为「已取消」，释放关联车位
func handleDeviceOffline(imei string) {
	logger.Mylog.Warn().Msgf("设备离线(心跳超时), 清理进行中订单: imei=%s", imei)
	seen := map[string]bool{}

	// 1. 二轮车充电订单：按产品 pid 结算（离线不关闸）
	if product := repository.ProductByDeviceID(imei); product != nil {
		for _, o := range repository.ActiveTwiceOrdersByPid(product.Pid) {
			if seen[o.Orderid] {
				continue
			}
			seen[o.Orderid] = true
			logger.Mylog.Info().Msgf("设备离线, 结算二轮车订单: %s", o.Orderid)
			if SettleTwiceOrderFn != nil {
				SettleTwiceOrderFn(o.Orderid, false)
			}
			// 兜底释放插座（结算失败也能保证插座恢复空闲）
			if rec := repository.ReceptacleByOrderid(o.Orderid); rec != nil {
				repository.FreeReceptacle(rec.ID)
			}
		}
	}

	// 2. 驿享充电 / 邻享订单：按最后一次上报电量结算扣款；踩踩停车订单：取消并释放车位
	candidates := []*model.OrderTbl{
		repository.FindUseOrderByDevice(imei),
		repository.FindPrivateUseOrderByDevice(imei),
		repository.FindPrivateReturnGunOrderByDevice(imei),
	}
	for _, o := range candidates {
		if o == nil || seen[o.Orderid] {
			continue
		}
		seen[o.Orderid] = true
		switch o.ConsumptionType {
		case repository.ConsumeTypeYiXiang, repository.ConsumeTypeNeighbor:
			logger.Mylog.Info().Msgf("设备离线, 结算充电订单: %s (type=%s, state=%s)", o.Orderid, o.ConsumptionType, o.State)
			if OfflineSettleOrderFn != nil {
				OfflineSettleOrderFn(o.Orderid)
			}
		default:
			logger.Mylog.Info().Msgf("设备离线, 取消订单: %s (type=%s, state=%s)", o.Orderid, o.ConsumptionType, o.State)
			repository.OfflineCancelOrder(o.Orderid)
		}
	}
}

// onEventMessage 处理设备上报事件
func onEventMessage(j job) {
	var body struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(j.payload, &body); err != nil {
		logger.Mylog.Err(err).Msgf("[MQTT] event事件解析异常: %v", j.payload)
	}

	switch body.Type {
	case eventGun:
		var isDraw struct {
			IsDraw    int8 `json:"isDraw"`
			Direction int8 `json:"direction"`
		}
		if len(body.Data) == 0 {
			logger.Mylog.Warn().Msgf("[MQTT] event %s 缺少 data: %s", body.Type, j.payload)
			return
		}
		if err := json.Unmarshal(body.Data, &isDraw); err != nil {
			logger.Mylog.Err(err).Msgf("[MQTT] gun 事件 data 解析异常: %s", body.Data)
			return
		}
		imei := strings.ToUpper(strings.Split(j.topic, "/")[1])
		// 订单结束: 检测是否还枪 ->还枪: 订单结束      ->未还: 超时占位费
		if isDraw.IsDraw != 0 {
			logger.Mylog.Info().Msgf("充电桩: %s ", imei)
			return
		}

		// 枪已归还,结算订单：优先待还枪（私桩/邻享），其次使用中（四轮车/私桩）
		order := repository.FindPrivateReturnGunOrderByDevice(imei)
		if order == nil {
			order = repository.FindUseOrderByDevice(imei)
		}
		if order == nil {
			order = repository.FindPrivateUseOrderByDevice(imei)
		}
		if order == nil {
			logger.Mylog.Error().Msgf("设备 %s 通道 %d 未查询到订单", imei, isDraw.Direction)
			return
		}
		logger.Mylog.Info().Msgf("[MQTT Event]找到订单号: %s ", order.Orderid)
		if FinishGunReturnOrderFn != nil {
			FinishGunReturnOrderFn(order.Orderid, 0)
		}

	case eventEnergy:
		var isFull struct {
			Ctl  string `json:"ctl"`
			Type string `json:"type"`
			Data struct {
				IsFull bool `json:"isFull"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body.Data, &isFull); err != nil {
			logger.Mylog.Warn().Msgf("[MQTT] energy 事件解析异常: %s", body.Data)
			return
		}
		// isFull=1 表示充电已满/枪已离车；未充满直接忽略
		if !isFull.Data.IsFull {
			return
		}

		imei := strings.ToUpper(strings.Split(j.topic, "/")[1])
		// 找订单（同 eventGun 还枪结算的查找顺序：待还枪 > 四轮车使用中 > 私桩使用中）
		order := repository.FindPrivateReturnGunOrderByDevice(imei)
		if order == nil {
			order = repository.FindUseOrderByDevice(imei)
		}
		if order == nil {
			order = repository.FindPrivateUseOrderByDevice(imei)
		}
		if order == nil {
			logger.Mylog.Warn().Msgf("[MQTT] energy 事件未找到设备对应订单, imei: %s, ctl: %s", imei, isFull.Ctl)
			return
		}

		// 清除缓存：记录充满时间（供后续还枪结算计算超时占位费）+ 状态置「充电完成」（对齐 TCP OpPowerFull）
		if order.FullTime == nil || order.FullTime.IsZero() {
			repository.UpdateOrderFullTime(order.Orderid)
		}
		if order.ConsumptionType == repository.ConsumeTypeNeighbor {
			SetRedisState("PrivateOrderState:"+order.Orderid, "充电完成")
		} else {
			SetRedisState("orderState:"+order.Orderid, "充电完成")
		}
		// 还枪检测与超时占位费计费由 gun_status 事件（eventGun → FinishGunReturnOrderFn）处理：
		// 还枪则结算（按 full_time 计超时占位费），未还则等待还枪事件。
	default:
		logger.Mylog.Warn().Msgf("[MQTT] event事件 %s 未匹配到", body.Type)
	}
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

// SendCommand 下发指令到指定设备（方向=默认）。
// 路由规则：设备已建立 TCP 连接（老设备）→ 走二进制帧；否则 → 走 MQTT JSON。
// data 为单字节数据（如插座号/占空比）；无数据传 -1。
func SendCommand(imei string, cmd int, data int) bool {
	return sendCommand(imei, cmd, protocol.DirectionDefault, intDataToBytes(data))
}

// SendCommandBytes 下发带多字节数据的指令（如时段/网关地址等）。路由规则同 SendCommand。
func SendCommandBytes(imei string, cmd int, data []byte) bool {
	return sendCommand(imei, cmd, protocol.DirectionDefault, data)
}

// SendCommandDir 下发带方向的指令（direction 为协议方向字节：DirectionLeft/Right/Default）。
// 用于开枪锁/拉闸/合闸等需要区分左右枪的指令。
func SendCommandDir(imei string, cmd int, direction byte, data int) bool {
	return sendCommand(imei, cmd, direction, intDataToBytes(data))
}

// sendCommand 核心下发：优先 TCP 二进制帧，否则 MQTT JSON。
func sendCommand(imei string, cmd int, direction byte, data []byte) bool {
	// 老设备：走 TCP 二进制协议（连接即设备身份，帧不带 nid 前缀，4G 帧）
	if _, ok := tcpConns.Load(imei); ok {
		frame := protocol.Encode(imei, byte(cmd), direction, data)
		if SendTCP(imei, frame) {
			logger.Mylog.Info().Msgf("[TCP]已通过 TCP 下发指令, device: %s, cmd: 0x%02X, direction: 0x%02X", imei, cmd, direction)
			return true
		}
	}

	// 新设备：MQTT JSON（方向在 data 字段以 -1/-2/0 编码，ctl 恒为 0）
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
	logger.Mylog.Info().Msgf("[MQTT]已下发指令, topic: %s, payload: %s", topic, payload)
	return true
}

// intDataToBytes 将单字节数据（int）转为字节切片；负数表示无数据。
func intDataToBytes(data int) []byte {
	if data < -2 {
		return []byte{}
	}
	return []byte{byte(int8(data))}
}

// dataInt 将字节数据转回 int（JSON 载荷用）；空数据返回 -1，多字节取首字节。
func dataInt(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	return int(int8(data[0]))
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
	logger.Mylog.Warn().Msgf("imei %s 掉线,无法发送 %q", imei, hexStr)
	return false
}

// SetRedisValue 设置 Redis 键值，5 秒过期（用于 TCP/MQTT 上报数据：功率/电流/电压/距离/电表等临时读数）。
func SetRedisValue(key, value string) {
	if rdb == nil {
		return
	}
	if err := rdb.Set(context.Background(), key, value, 70*time.Second).Err(); err != nil {
		logger.Mylog.Error().Err(err).Msgf("[Redis] SET 失败 key=%s value=%s", key, value)
		return
	}
	logger.Mylog.Debug().Msgf("[Redis] SET 成功 key=%s value=%s", key, value)
}

// SetRedisState 设置 Redis 键值，不过期（用于订单状态标记 orderState/PrivateOrderState）。
func SetRedisState(key, value string) {
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
