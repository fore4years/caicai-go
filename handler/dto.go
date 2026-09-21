package handler

import (
	"time"
)

// LocalDateTime 对应 Java 的 LocalDateTime。
// Jackson 对 LocalDateTime 默认按 ISO 序列化（无时区），这里对齐为 2006-01-02T15:04:05。
type LocalDateTime time.Time

// MarshalJSON 将时间序列化为与 Java LocalDateTime 一致的 ISO 格式。
func (t LocalDateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format("2006-01-02T15:04:05") + `"`), nil
}

// timeToLocal 将 time.Time 转为可空时间；零值返回 nil（对齐 Java non_null 省略 null 字段）。
func timeToLocal(t time.Time) *LocalDateTime {
	if t.IsZero() {
		return nil
	}
	v := LocalDateTime(t)
	return &v
}

// timeToString 将 time.Time 转为字符串；零值返回空串。
// Java 中 fullTime/leaveTime 是 String（由 MyBatis 将 datetime 列转成字符串）。
func timeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// PlaceDTO 对应 Java domain.Place，JSON 字段名与 Jackson 序列化保持一致（驼峰）。
type PlaceDTO struct {
	Placeid         string `json:"placeid,omitempty"`
	Ownerid         string `json:"ownerid,omitempty"`
	Province        string `json:"province,omitempty"`
	City            string `json:"city,omitempty"`
	District        string `json:"district,omitempty"`
	Streer          string `json:"streer,omitempty"`
	RelatedBuilding string `json:"relatedBuilding,omitempty"`
	Longitude       string `json:"longitude,omitempty"`
	Latitude        string `json:"latitude,omitempty"`
	Rate            string `json:"rate,omitempty"`
	State           string `json:"state,omitempty"`
	OpenTime        string `json:"openTime,omitempty"`
	FixType         string `json:"fixType,omitempty"`
	Ischarge        int32  `json:"ischarge,omitempty"`
	Iscarlock       int32  `json:"iscarlock,omitempty"`
	ParkRate        string `json:"parkRate,omitempty"`
	ImageID         int32  `json:"imageId,omitempty"`
	Certificate     int32  `json:"certificate,omitempty"`
	ChargingPid     string `json:"chargingPid,omitempty"`
	LockPid         string `json:"lockPid,omitempty"`
	LedID           string `json:"ledId,omitempty"`
}

// PlaceDtoDTO 对应 Java dto.PlaceDto。注意 Java 的 PlaceDto 字段名本身是 park_rate（下划线），
// 与 Place.parkRate（驼峰）不同；saomaForCharge 中 BeanUtils.copyProperties 不会复制 parkRate，
// 因此 park_rate 恒为 null（这里保持零值，omitempty 省略）。
type PlaceDtoDTO struct {
	Placeid         string `json:"placeid,omitempty"`
	Ownerid         string `json:"ownerid,omitempty"`
	Province        string `json:"province,omitempty"`
	City            string `json:"city,omitempty"`
	District        string `json:"district,omitempty"`
	Streer          string `json:"streer,omitempty"`
	RelatedBuilding string `json:"relatedBuilding,omitempty"`
	Longitude       string `json:"longitude,omitempty"`
	Latitude        string `json:"latitude,omitempty"`
	Rate            string `json:"rate,omitempty"`
	State           string `json:"state,omitempty"`
	OpenTime        string `json:"openTime,omitempty"`
	FixType         string `json:"fixType,omitempty"`
	Ischarge        int32  `json:"ischarge,omitempty"`
	ParkRate        string `json:"park_rate,omitempty"`
	LockID          string `json:"lockId,omitempty"`
	ChargeID        string `json:"chargeId,omitempty"`
	Nid             string `json:"nid,omitempty"`
	Direction       string `json:"direction,omitempty"`
	Code            string `json:"code,omitempty"`
	LedID           string `json:"ledId,omitempty"`
}

// OrderDTO 对应 Java domain.OrderBean，JSON 字段名对齐 Jackson 序列化（驼峰）。
// 时间字段（LocalDateTime）使用可空指针，nil 时省略，对齐 Java 的 non_null 配置。
type OrderDTO struct {
	Orderid              string         `json:"orderid,omitempty"`
	Openid               string         `json:"openid,omitempty"`
	Lockid               string         `json:"lockid,omitempty"`
	ReserveTime          *LocalDateTime `json:"reserveTime,omitempty"`
	BeginTime            *LocalDateTime `json:"beginTime,omitempty"`
	OverTime             *LocalDateTime `json:"overTime,omitempty"`
	State                string         `json:"state,omitempty"`
	TotalPrice           float64        `json:"totalPrice,omitempty"`
	CloseTime            *LocalDateTime `json:"closeTime,omitempty"`
	PlateNum             string         `json:"plateNum,omitempty"`
	SpacesID             int32          `json:"spacesId,omitempty"`
	BookingStartTime     *LocalDateTime `json:"bookingStartTime,omitempty"`
	BookingEndTime       *LocalDateTime `json:"bookingEndTime,omitempty"`
	ConsumptionLocation  string         `json:"consumptionLocation,omitempty"`
	ConsumptionType      string         `json:"consumptionType,omitempty"`
	ChargingRates        float64        `json:"chargingRates,omitempty"`
	ChargerPower         int32          `json:"chargerPower,omitempty"`
	StartMode            string         `json:"startMode,omitempty"`
	StopMode             string         `json:"stopMode,omitempty"`
	ChargingTime         string         `json:"chargingTime,omitempty"`
	SettlementTime       *LocalDateTime `json:"settlementTime,omitempty"`
	BasicConsumption     float64        `json:"basicConsumption,omitempty"`
	GiftAmount           string         `json:"giftAmount,omitempty"`
	TwiceChargeTime      string         `json:"twiceChargeTime,omitempty"`
	Pid                  string         `json:"pid,omitempty"`
	ChargingDegree       float64        `json:"chargingDegree,omitempty"`
	FullTime             string         `json:"fullTime,omitempty"`
	LeaveTime            string         `json:"leaveTime,omitempty"`
	ChargeGunPull        int32          `json:"chargeGunPull,omitempty"`
	GunDisconnectTime    *LocalDateTime `json:"gunDisconnectTime,omitempty"`
	DecideAmount         int32          `json:"decideAmount,omitempty"`
	NeighborRelocateTime *LocalDateTime `json:"neighborRelocateTime,omitempty"`
	MoveCarTime          *LocalDateTime `json:"moveCarTime,omitempty"`
	ImageID              string         `json:"imageId,omitempty"`
	WechatTransactionID  string         `json:"wechatTransactionId,omitempty"`
}

// ChargeOrderResultDTO 对应 Java dto.ChargeOrderResultDto。注意 plate_num 是下划线字段名。
type ChargeOrderResultDTO struct {
	Orderid   string `json:"orderid,omitempty"`
	Openid    string `json:"openid,omitempty"`
	Lockid    string `json:"lockid,omitempty"`
	State     string `json:"state,omitempty"`
	PlateNum  string `json:"plate_num,omitempty"`
	BeginTime string `json:"beginTime,omitempty"`
}

// UsermessageDTO 对应 Java domain.usermessage。
type UsermessageDTO struct {
	ID      int32  `json:"id,omitempty"`
	Openid  string `json:"openid,omitempty"`
	Message string `json:"message,omitempty"`
	MsgTime string `json:"msgTime,omitempty"`
	Status  string `json:"status,omitempty"`
}

// Result 对应 Java model.sys.Result 统一封装。
type Result struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Msg     string      `json:"msg,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ResultSuccess 构造成功响应（对齐 Java Result.success(data)）。
func ResultSuccess(data interface{}) Result {
	return Result{Success: true, Code: 200, Data: data}
}

// UserDTO 对应 Java model.UserBean，JSON 字段名对齐 Jackson 序列化（驼峰）。
type UserDTO struct {
	Openid       string  `json:"openid,omitempty"`
	Omid         string  `json:"omid,omitempty"`
	YiparlOpenid string  `json:"yiparlOpenid,omitempty"`
	NickName     string  `json:"nickName,omitempty"`
	Province     string  `json:"province,omitempty"`
	City         string  `json:"city,omitempty"`
	Phone        string  `json:"phone,omitempty"`
	Integral     string  `json:"integral,omitempty"`
	FreeTime     string  `json:"freeTime,omitempty"`
	PlateNum     string  `json:"plateNum,omitempty"`
	IDNumber     string  `json:"idNumber,omitempty"`
	Name         string  `json:"name,omitempty"`
	Avatar       []byte  `json:"avatar,omitempty"`
	IDCardEmblem int32   `json:"idCardEmblem,omitempty"`
	IDCardAvatar int32   `json:"idCardAvatar,omitempty"`
	Balans       float64 `json:"balans,omitempty"`
	FreezeBalans float64 `json:"freezeBalans,omitempty"`
	IDEntity     string  `json:"idEntity,omitempty"`
	OmEnable     string  `json:"omEnable,omitempty"`
	IsSteer      string  `json:"isSteer,omitempty"`
	IsProcedure  string  `json:"isProcedure,omitempty"`
	IsLogin      string  `json:"isLogin,omitempty"`
}
