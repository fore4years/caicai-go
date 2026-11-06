// Package service 延迟任务分发器（替代 RabbitMQ 的进程内实现）。
//
// Java 使用 RabbitMQ x-delayed-message 交换机实现「延迟投递 + 消费者异步处理」，
// 本文件用 Go 的 goroutine + time.AfterFunc + 带缓冲 channel 等价实现：
//   - SendMsg(routingKey, msg, delayMillis)：发布延迟消息（对齐 MessageServiceImpl.sendMsg）
//   - RegisterHandler(routingKey, handler)：注册消费者（对齐 @RabbitListener）
//   - Redis SET consumed_msg_ids 做消息幂等去重（对齐 MessageReceiver.isConsume）
//
// 注意：进程内延迟任务不跨进程、不持久化，进程重启后未投递的任务会丢失，
// 这是取消 RabbitMQ 后的已知取舍（见项目决策）。
package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"caicai-go/logger"
	"caicai-go/repository"
)

// 路由键常量（对齐 Java QueueConfig 中 bind 的 routing key）。
const (
	RoutingCloseLock            = "close_lock"
	RoutingCloseTwice           = "close_twice"
	RoutingCarLeave             = "car_leave"
	RoutingPowerException       = "power_exception"
	RoutingSaveTwiceMeter       = "save_twice_meter"
	RoutingCheckLockIsClose     = "check_lock_isClose"
	RoutingCloseOrder           = "close_order"
	RoutingNoteCloseOrder       = "note_close_order"
	RoutingChargingEndDetection = "charging_end_detection"
	RoutingStartCharge          = "start_charge"
	RoutingSendBluetoothMac     = "send_bluetooth_mac"
)

// delayedMsg 一条延迟消息（对齐 RabbitMQ 的 message + messageId）。
type delayedMsg struct {
	id         string
	routingKey string
	payload    string
}

var (
	handlersMu      sync.RWMutex
	routingHandlers = map[string]func(string){}

	msgQueue        chan delayedMsg
	dispatcherOnce  sync.Once
	dispatcherStop  chan struct{}
	dispatcherWg    sync.WaitGroup
	dispatcherWorkN = 5 // 对齐 Java simple.concurrency=5
)

// InitDispatcher 初始化延迟任务分发器（幂等）。启动 worker 池消费消息队列。
func InitDispatcher() {
	dispatcherOnce.Do(func() {
		msgQueue = make(chan delayedMsg, 1024)
		dispatcherStop = make(chan struct{})
		for i := 0; i < dispatcherWorkN; i++ {
			dispatcherWg.Add(1)
			go dispatcherWorker()
		}
		logger.Mylog.Info().Msg("延迟任务分发器已启动（替代 RabbitMQ）")
	})
}

// StopDispatcher 优雅关闭分发器：等待在途任务处理完成后退出。
func StopDispatcher() {
	select {
	case <-dispatcherStop: // 已关闭
		return
	default:
	}
	close(dispatcherStop)
	dispatcherWg.Wait()
}

// RegisterHandler 注册路由键对应的消费者（对齐 @RabbitListener(queues=...)）。
func RegisterHandler(routingKey string, h func(string)) {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	routingHandlers[routingKey] = h
}

// SendMsg 发布一条延迟消息（对齐 MessageServiceImpl.sendMsg）。
// delayMillis 为延迟毫秒；<=0 表示立即投递。
func SendMsg(routingKey, msg string, delayMillis int64) {
	InitDispatcher()
	id := uuid.NewString()
	logger.Mylog.Info().Msgf("分发延迟任务 routingKey=%s msg=%s delay=%dms", routingKey, msg, delayMillis)
	if delayMillis <= 0 {
		enqueue(delayedMsg{id: id, routingKey: routingKey, payload: msg})
		return
	}
	time.AfterFunc(time.Duration(delayMillis)*time.Millisecond, func() {
		enqueue(delayedMsg{id: id, routingKey: routingKey, payload: msg})
	})
}

func enqueue(m delayedMsg) {
	select {
	case <-dispatcherStop:
		return
	case msgQueue <- m:
	default:
		logger.Mylog.Warn().Msgf("[dispatcher] 队列已满，丢弃任务 routingKey=%s", m.routingKey)
	}
}

func dispatcherWorker() {
	defer dispatcherWg.Done()
	for {
		select {
		case <-dispatcherStop:
			return
		case m := <-msgQueue:
			dispatch(m)
		}
	}
}

func dispatch(m delayedMsg) {
	handlersMu.RLock()
	h := routingHandlers[m.routingKey]
	handlersMu.RUnlock()
	if h == nil {
		logger.Mylog.Warn().Msgf("[dispatcher] 未注册的 routing key: %s", m.routingKey)
		return
	}
	// 幂等去重（对齐 Java isConsume：Redis SET consumed_msg_ids，24h TTL）
	if isMsgConsumed(m.id) {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Mylog.Error().Msgf("[dispatcher] 任务 panic routingKey=%s err=%v", m.routingKey, r)
		}
	}()
	h(m.payload)
	markMsgConsumed(m.id)
}

// consumedMsgIDsKey Redis 中保存已消费消息 ID 的集合键（对齐 Java CONSUMED_MSG_IDS_KEY）。
const consumedMsgIDsKey = "consumed_msg_ids"

func isMsgConsumed(msgID string) bool {
	if rdb == nil {
		return false
	}
	ok, err := rdb.SIsMember(context.Background(), consumedMsgIDsKey, msgID).Result()
	return err == nil && ok
}

func markMsgConsumed(msgID string) {
	if rdb == nil {
		return
	}
	ctx := context.Background()
	rdb.SAdd(ctx, consumedMsgIDsKey, msgID)
	rdb.Expire(ctx, consumedMsgIDsKey, 24*time.Hour)
}

// ============================================================================
// 消费者实现（对齐 Java MessageReceiver 各 @RabbitListener 方法）
// 微信订阅消息推送通过 handler 注入的 SendXxxFn 函数指针触发（见 handler/subscribe_message.go 的 init）。
// ============================================================================

// registerMQHandlers 注册全部延迟任务消费者。
func registerMQHandlers() {
	RegisterHandler(RoutingCloseLock, closeLockHandler)
	RegisterHandler(RoutingCloseTwice, closeTwiceHandler)
	RegisterHandler(RoutingCarLeave, carLeaveHandler)
	RegisterHandler(RoutingPowerException, powerExceptionHandler)
	RegisterHandler(RoutingSaveTwiceMeter, saveTwiceMeterHandler)
	RegisterHandler(RoutingCheckLockIsClose, checkLockIsCloseHandler)
	RegisterHandler(RoutingCloseOrder, closeOrderHandler)
	RegisterHandler(RoutingNoteCloseOrder, noteCloseOrderHandler)
	RegisterHandler(RoutingChargingEndDetection, chargingEndDetectionHandler)
	RegisterHandler(RoutingStartCharge, startChargeHandler)
	RegisterHandler(RoutingSendBluetoothMac, sendBluetoothMacHandler)
}

// closeLockHandler 关闭车位锁（对齐 MessageReceiver.receive）。
func closeLockHandler(orderID string) {
	lock := repository.LockByOrder(orderID)
	if lock == nil {
		logger.Mylog.Warn().Msgf("未找到对应的车位锁信息, 订单id: %s", orderID)
		return
	}
	logger.Mylog.Info().Msgf("执行关闭车位锁任务, 车位锁信息: %+v", lock)
	if !ClosePlaceLock(lock.DeviceID()) {
		logger.Mylog.Error().Msgf("关闭车位锁失败, 订单id: %s", orderID)
	}
}

// closeTwiceHandler 结束二轮车充电订单（对齐 MessageReceiver.TwiceReceive）。
func closeTwiceHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		logger.Mylog.Info().Msgf("查无此订单: %s", orderID)
		return
	}
	if order.State == repository.OrderStateFinished || order.State == repository.OrderStateRevoked {
		logger.Mylog.Info().Msgf("订单id:%s, 此订单已结束", orderID)
		return
	}

	// 异步执行关闸操作（不阻塞主流程）
	go CloseGaugeByOrder(orderID)

	// 执行订单关闭和支付（不含关闸）
	if SettleTwiceOrderFn != nil {
		if !SettleTwiceOrderFn(orderID, false) {
			logger.Mylog.Error().Msgf("关闭二轮车订单失败, 订单id: %s", orderID)
		}
	}
	// 发送充电结算消息（对齐 Java sendChargingSettleMessage）
	if SendChargingSettleMessageFn != nil {
		SendChargingSettleMessageFn(orderID)
	}
}

// carLeaveHandler 检测车辆是否驶离（对齐 MessageReceiver.carLeaveReceive）。
func carLeaveHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		logger.Mylog.Info().Msgf("未查询到订单信息, 订单id: %s", orderID)
		return
	}
	if order.State == repository.OrderStateFinished || order.State == repository.OrderStateRevoked {
		return
	}
	if order.LeaveTime != nil {
		logger.Mylog.Info().Msgf("订单id:%s, 已驶离车位", orderID)
		return
	}
	// 未驶离, 发送挪车通知（对齐 Java CarLeaveMessage）
	if CarLeaveMessageFn != nil {
		CarLeaveMessageFn(orderID)
	}
	// 30 分钟后再次检测
	SendMsg(RoutingCarLeave, orderID, 1000*60*30)
}

// powerExceptionHandler 功率异常（对齐 MessageReceiver.PowerExceptionReceive）。
func powerExceptionHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		logger.Mylog.Info().Msgf("查无此订单: %s", orderID)
		return
	}
	if order.State == repository.OrderStateFinished {
		return
	}
	logger.Mylog.Info().Msgf("执行关闸任务, 结算订单, 订单id: %s", orderID)
	if SettleTwiceOrderFn != nil {
		SettleTwiceOrderFn(orderID, true)
	}
	// 发送充电异常通知 + 结算消息（对齐 Java sendChargingExceptionMessage + sendChargingSettleMessage）
	if SendChargingExceptionMessageFn != nil {
		SendChargingExceptionMessageFn(orderID)
	}
	if SendChargingSettleMessageFn != nil {
		SendChargingSettleMessageFn(orderID)
	}
}

// saveTwiceMeterHandler 存储二轮车订单电量信息（对齐 MessageReceiver.SaveTwiceMeterReceive）。
func saveTwiceMeterHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order.State == repository.OrderStateFinished {
		return
	}
	meterValue := repository.ReOrderPowerValue(orderID)
	SetRedisValue(orderID, meterValue.String())
}

// checkLockIsCloseHandler 检测是否结束订单（对齐 MessageReceiver.CheckLockIsCloseReceive）。
func checkLockIsCloseHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order.State == repository.OrderStateFinished {
		return
	}
	if order.State == repository.OrderStateUse {
		// 发送车辆驶离通知（对齐 Java ParkingMessage）
		if ParkingMessageFn != nil {
			ParkingMessageFn(orderID)
		}
		// 5 分钟后再次检测
		SendMsg(RoutingCheckLockIsClose, orderID, 1000*60*5)
	}
}

// closeOrderHandler 驿享充电订单(时间模式) 拉闸断电（对齐 MessageReceiver.closeOrderReceive）。
func closeOrderHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order.State == repository.OrderStateFinished ||
		order.State == repository.OrderStateRevoked || order.State == repository.OrderStateWaitPay {
		return
	}
	if order.State != repository.OrderStateUse {
		return
	}

	gun := repository.ChargingGunByOrder(orderID)
	if gun == nil {
		logger.Mylog.Info().Msgf("根据此订单 %s 未查询到充电枪信息", orderID)
		return
	}
	logger.Mylog.Info().Msgf("执行订单拉闸（断电）任务, 订单id: %s", orderID)
	CloseEle(gun.DeviceID(), gun.Gun.Direction)

	// 记录订单中充满电的时间（为了方便计算超时停车费），已有充满时间则不重复设置
	isNewFullTimeSet := false
	if order.FullTime == nil {
		now := time.Now()
		repository.UpdateOrderFields(orderID, map[string]interface{}{"full_time": now})
		isNewFullTimeSet = true
	}

	if isNewFullTimeSet {
		// 发送资源归还提醒（对齐 Java RelocateCarAt8clockMessageV2 / RelocateCarMessageV2）
		if time.Now().Hour() < 8 {
			if RelocateCarAt8clockMessageV2Fn != nil {
				RelocateCarAt8clockMessageV2Fn(orderID)
			}
		} else {
			if RelocateCarMessageV2Fn != nil {
				RelocateCarMessageV2Fn(orderID)
			}
		}
	}
}

// noteCloseOrderHandler 记录关闭订单时间并延迟重投 close_order（对齐 MessageReceiver.noteCloseOrderReceive）。
func noteCloseOrderHandler(msg string) {
	// msg = [202507221530047970316, 2]
	list := strings.Split(strings.Trim(msg, "[]"), ",")
	if len(list) < 2 {
		logger.Mylog.Warn().Msgf("note_close_order 消息格式异常: %s", msg)
		return
	}
	orderID := strings.TrimSpace(list[0])
	hours, err := strconv.Atoi(strings.TrimSpace(list[1]))
	if err != nil || hours <= 0 {
		logger.Mylog.Warn().Msgf("note_close_order 解析延迟小时数失败: %s", msg)
		return
	}
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order.State == repository.OrderStateFinished ||
		order.State == repository.OrderStateRevoked || order.State == repository.OrderStateWaitPay {
		return
	}
	SendMsg(RoutingCloseOrder, orderID, int64(1000*60*60*hours))
}

// chargingEndDetectionHandler 异步执行结束订单时的断电检测（对齐 MessageReceiver.chargingEndDetectionReceive）。
func chargingEndDetectionHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		return
	}
	var gun = repository.ChargingGunByOrder(orderID)
	if order.ConsumptionType == repository.ConsumeTypeNeighbor {
		gun = repository.ChargingGunByPrivateOrder(orderID)
	}
	if gun == nil {
		return
	}
	ChargingEndDetection(gun.DeviceID())
}

// startChargeHandler 异步开启驿享充电订单并发送关闭蓝牙检测（对齐 MessageReceiver.startChargeReceive）。
func startChargeHandler(orderID string) {
	if StartOrderFn != nil {
		StartOrderFn(orderID)
	}
	gun := repository.ChargingGunByOrder(orderID)
	if gun == nil {
		return
	}
	BluetoothMacCancel(gun.DeviceID())
}

// sendBluetoothMacHandler 异步发送蓝牙设备 mac 地址到硬件（对齐 MessageReceiver.sendBluetoothMacReceive）。
func sendBluetoothMacHandler(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		return
	}
	binding, err := repository.UserBluetoothBindingsByOpenid(order.Openid)
	if err != nil || binding == nil {
		logger.Mylog.Info().Msgf("用户:%s, 未绑定蓝牙设备", order.Openid)
		return
	}
	gun := repository.ChargingGunByOrder(orderID)
	if gun == nil {
		return
	}
	SendBluetoothMac(gun.DeviceID(), binding.BluetoothMac)
}

func init() {
	registerMQHandlers()
}
