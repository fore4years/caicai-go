package service

// 订阅消息发送函数指针（由 handler 层在 init 中注入，供延迟任务分发器/消费者/设备上行触发，避免 service→handler 循环依赖）。
var (
	SendChargingSettleMessageFn    func(orderID string) bool
	SendChargingExceptionMessageFn func(orderID string) bool
	SendChargingMessageFn          func(openid string) bool
	CarLeaveMessageFn              func(orderID string) bool
	ParkingMessageFn               func(orderID string) bool
	RelocateCarMessageV2Fn         func(orderID string) bool
	RelocateCarAt8clockMessageV2Fn func(orderID string) bool
)
