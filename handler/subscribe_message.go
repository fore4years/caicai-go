package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/repository"
	"caicai-go/service"
)

// ============ SubscribeMessageController（/subscribeMessage） ============

type SubscribeMessageController struct{}

// formOrQuery 读取表单参数，缺失时回退到查询参数（对齐 Spring 未标注注解的 String 参数绑定）。
func formOrQuery(ctx *gin.Context, key string) string {
	if v := ctx.PostForm(key); v != "" {
		return v
	}
	return ctx.Query(key)
}

// SelectRid 查询 rid（对齐 Java SubscribeMessageServiceImpl.selectRid，返回微信接口原始报文）。
func (c *SubscribeMessageController) SelectRid(ctx *gin.Context) {
	rid := formOrQuery(ctx, "rid")
	if rid == "" {
		ctx.String(http.StatusOK, "")
		return
	}
	token := getUniAppToken()
	if token == "" {
		ctx.String(http.StatusOK, "")
		return
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.weixin.qq.com/cgi-bin/openapi/rid/get?access_token="+token,
		strings.NewReader(fmt.Sprintf(`{"rid":%q}`, rid)))
	if err != nil {
		ctx.String(http.StatusOK, "")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		ctx.String(http.StatusOK, "")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ctx.String(http.StatusOK, string(body))
}

// SendParkingMessage 发送车辆驶离通知（对齐 Java sendParkingMessage -> ParkingMessage）。
func (c *SubscribeMessageController) SendParkingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, parkingMessage(formOrQuery(ctx, "orderId")))
}

// SendRelocateCarMessage 发送资源归还提醒（对齐 Java sendRelocateCarMessage -> RelocateCarMessageV2）。
func (c *SubscribeMessageController) SendRelocateCarMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, relocateCarMessageV2(formOrQuery(ctx, "orderId")))
}

// SendParkingTicketMessage 发送停车限免券开通通知（对齐 Java sendParkingTicketMessage）。
func (c *SubscribeMessageController) SendParkingTicketMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, parkingTicketMessage(formOrQuery(ctx, "openid")))
}

// SendChargingMessage 发送充电开始提醒（对齐 Java sendChargingMessage）。
func (c *SubscribeMessageController) SendChargingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, sendChargingMessage(formOrQuery(ctx, "openid")))
}

// SendChargingSettleMessage 发送充电结算提醒（对齐 Java sendChargingSettleMessage）。
func (c *SubscribeMessageController) SendChargingSettleMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, sendChargingSettleMessage(formOrQuery(ctx, "orderId")))
}

// SendChargingExceptionMessage 发送充电异常通知（对齐 Java sendChargingExceptionMessage）。
func (c *SubscribeMessageController) SendChargingExceptionMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, sendChargingExceptionMessage(formOrQuery(ctx, "orderId")))
}

// SendOrderOverMessage 发送订单结束通知（对齐 Java sendOrderOverMessage）。
func (c *SubscribeMessageController) SendOrderOverMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, sendOrderOverMessage(formOrQuery(ctx, "orderId")))
}

// SendMoveCarMessage 发送挪车通知（对齐 Java sendMoveCarMessage）：更新 move_car_time 并发送订阅消息。
func (c *SubscribeMessageController) SendMoveCarMessage(ctx *gin.Context) {
	var req struct {
		CarNumber string `json:"carNumber"`
		Phone     string `json:"phone"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(sendMoveCarMessage(req.CarNumber, req.Phone)))
}

// ============ 微信订阅消息发送（对齐 Java SubscribeMessageServiceImpl） ============

// 订阅消息模板 ID（对齐 Java SubscribeMessageServiceImpl 的常量）。
const (
	relocateCarTemplateID           = "EFVYxG7-NHLfD4umggEPYPcSBb67pO96slftGyc64Zc" // 挪车/车辆驶离
	parkingTemplateID               = "mhDWeYhV8LiDBswSSspfTZjY5XJa8etzKjt1hlfta3U" // 车辆驶离
	parkingTicketTemplateID         = "Ue7zC1m-TyrMCLmPAjet233uMR_Oq1b_UWZFFne-rkA" // 停车限免券
	chargingTemplateID              = "1ubUtGQ5tGyeY2yJJKfBcmIAZkqnM3UtwnCRpdSwdaQ" // 充电开始
	chargingSettleTemplateID        = "YRqtOSjEvLkAvjo-TT93HArL-pbuOPU-Q3MKxvheoiA" // 充电结算
	powerExceptionTemplateID        = "CpG-NGtdhZTm0hMUre-BwH6EX4FRL-IU-H35tkwfenY" // 充电异常
	privateChargingShareTemplateID  = "WI9ydYRjMI4N3Xx22RxEMRyXWN7_59fsLpH0ypkk1lo" // 私桩共享申请
	orderOverTemplateID             = "t0NC2308XRVwfJcVCW54rJqOpfhTQYYI1J7v-e_f9oI" // 订单结束通知
	ownerOrderOverTemplateID        = "t0NC2308XRVwfJcVCW54rJqOpfhTQYYI1J7v-e_f9oI" // 桩主订单结束通知（同模板）
	neighborToPrivateUserTemplateID = "rUHl1Q8Qf7dctWfroxNI19berXdUC0m_fUXxJ2G1b2Y" // 邻居车位信息给桩主
	newRelocateCarTemplateID        = "EFVYxG7-NHLfD4umggEPYPxnS8riG5KSe_CVUeqXaSI" // 挪车（桩主发邻居）
	newRelocateCarV2TemplateID      = "OfU1V5GyoOja4HpSdlR3-5wix0KUBnryU__8sT1lHdQ" // 资源归还提醒
	moveCarTemplateID               = "EFVYxG7-NHLfD4umggEPYMimilxrQAWXDgS7oeStoBQ" // 挪车（车位无效）
	newParkingTemplateID            = "Dnp2kSbkPFCrNp2zY4kbVmhp5ykI3WFt_VK-H7rSsyc" // 车辆驶离（新版）
	returnGunTemplateID             = "t54uy7tq2uZvloNxY_i3f67PovedW4zltklUvFvRLP0" // 充电枪归位提醒
)

// subscribeSendURL 微信订阅消息发送接口（后面拼 access_token）。
const subscribeSendURL = "https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token="

// subscribeValue 订阅消息 data 关键词构造：每个关键词都是 {"value": "..."}。
func subscribeValue(v string) map[string]string {
	return map[string]string{"value": v}
}

// formatWeChatTime 格式化时间（对齐 Java 的 "yyyy年MM月dd日 HH:mm"）。
func formatWeChatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006年01月02日 15:04")
}

// nowWeChatTime 当前时间（对齐 Java SimpleDateFormat("yyyy年MM月dd日 HH:mm")）。
func nowWeChatTime() string {
	return time.Now().Format("2006年01月02日 15:04")
}

// nowWeChatTimeSeconds 当前时间（对齐 Java SimpleDateFormat("yyyy年MM月dd日 HH:mm:ss")）。
func nowWeChatTimeSeconds() string {
	return time.Now().Format("2006年01月02日 15:04:05")
}

// sysTimeString 当前时间（对齐 Java DateTimeUtil.getSysTime() = "yyyy-MM-dd HH:mm:ss"）。
func sysTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// marshalOrderJSON 序列化订单（用于小程序页面跳转参数 order=...，对齐 Java ObjectMapper.writeValueAsString(orderBean)）。
// 注意：Go 侧 time 字段序列化为 RFC3339，Java LocalDateTime 序列化格式不同，需与小程序端确认。
func marshalOrderJSON(order interface{}) string {
	b, err := json.Marshal(order)
	if err != nil {
		return ""
	}
	return string(b)
}

// spacesCodeByOrder 取订单对应车位的车位码（对齐 Java parkingSpacesMapper.selectById(order.getSpacesId()).getSpacesCode()）。
func spacesCodeByOrder(orderID string) string {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return ""
	}
	if s := repository.SpacesByID(order.SpacesID); s != nil {
		return s.SpacesCode
	}
	return ""
}

// sendWeChatSubscribeMessage 发送微信订阅消息（对齐 Java SubscribeMessageServiceImpl.sendMessage）。
// 返回 errcode == 0 是否成功；token 失效(40001)时刷新重试，最多重试 2 次。
func sendWeChatSubscribeMessage(templateID, page, openid string, data map[string]map[string]string) bool {
	payload := map[string]interface{}{
		"template_id":       templateID,
		"touser":            openid,
		"data":              data,
		"miniprogram_state": "formal",
		"lang":              "zh_CN",
	}
	if page != "" {
		payload["page"] = page
	}
	body, err := json.Marshal(payload)
	if err != nil {
		logger.Mylog.Err(err).Msg("[订阅消息] 序列化请求失败")
		return false
	}

	for attempt := range 3 {
		token := getUniAppToken()
		if token == "" {
			logger.Mylog.Warn().Msg("[订阅消息] 获取 access_token 失败")
			return false
		}

		resp, err := wxPostJSON(subscribeSendURL+token, string(body))
		if err != nil {
			logger.Mylog.Err(err).Msgf("[订阅消息] 发送失败, openid=%s", openid)
			return false
		}

		errcode, _ := resp["errcode"].(float64)
		if errcode == 0 {
			return true
		}
		if errcode == 40001 {
			logger.Mylog.Warn().Msgf("[订阅消息] token 失效(40001), 第%d次刷新重试", attempt+1)
			refreshUniAppToken()
			continue
		}
		if errcode == 43101 {
			logger.Mylog.Warn().Msgf("[订阅消息] 用户未订阅(43101), openid=%s", openid)
		} else {
			logger.Mylog.Warn().Msgf("[订阅消息] 发送返回错误: errcode=%v errmsg=%v", resp["errcode"], resp["errmsg"])
		}
		return false
	}
	return false
}

// parkingMessage 车辆驶离通知（对齐 Java ParkingMessage）。
func parkingMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		logger.Mylog.Warn().Msgf("[订阅消息] 未找到订单, orderId=%s", orderID)
		return false
	}
	page := "pages/charging/charging?order=" + marshalOrderJSON(order) +
		"&ScanQRconfirm=false&lockjpg=true&Startcharging=false&Endcharging=true"
	data := map[string]map[string]string{
		"character_string2": subscribeValue(spacesCodeByOrder(orderID)),
		"time3":             subscribeValue(sysTimeString()),
		"thing12":           subscribeValue("车辆已驶离,请及时结束使用及确认结账!"),
	}
	return sendWeChatSubscribeMessage(parkingTemplateID, page, order.Openid, data)
}

// newParkingMessage 车辆驶离通知（新版页面，对齐 Java newParkingMessage）。
func newParkingMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	page := "miniprogram/pages/overOrder/overOrder?order=" + marshalOrderJSON(order)
	data := map[string]map[string]string{
		"character_string2": subscribeValue(spacesCodeByOrder(orderID)),
		"time3":             subscribeValue(sysTimeString()),
		"thing12":           subscribeValue("车辆已驶离,请及时结束使用及确认结账!"),
	}
	return sendWeChatSubscribeMessage(newParkingTemplateID, page, order.Openid, data)
}

// relocateCarMessage 挪车通知（对齐 Java RelocateCarMessage）。
func relocateCarMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	data := map[string]map[string]string{
		"date3":             subscribeValue(nowWeChatTime()),
		"thing4":            subscribeValue("充电完成，请及时驶离或还枪，否则加收费用"),
		"character_string6": subscribeValue(spacesCodeByOrder(orderID)),
	}
	return sendWeChatSubscribeMessage(relocateCarTemplateID, "", order.Openid, data)
}

// relocateCarAt8clockMessage 挪车通知（8点前，对齐 Java RelocateCarAt8clockMessage）。
func relocateCarAt8clockMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	data := map[string]map[string]string{
		"date3":             subscribeValue(nowWeChatTime()),
		"thing4":            subscribeValue("充电完成,8点前驶离或还枪,否则加收费用"),
		"character_string6": subscribeValue(spacesCodeByOrder(orderID)),
	}
	return sendWeChatSubscribeMessage(relocateCarTemplateID, "", order.Openid, data)
}

// relocateCarMessageV2 资源归还提醒（对齐 Java RelocateCarMessageV2）。
func relocateCarMessageV2(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	data := map[string]map[string]string{
		"time1":             subscribeValue(nowWeChatTime()),
		"thing2":            subscribeValue("充电完成，请及时驶离或还枪，否则加收费用"),
		"character_string3": subscribeValue(spacesCodeByOrder(orderID)),
	}
	return sendWeChatSubscribeMessage(newRelocateCarV2TemplateID, "", order.Openid, data)
}

// relocateCarAt8clockMessageV2 资源归还提醒（8点前，对齐 Java RelocateCarAt8clockMessageV2）。
func relocateCarAt8clockMessageV2(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	data := map[string]map[string]string{
		"time1":             subscribeValue(nowWeChatTime()),
		"thing2":            subscribeValue("充电完成,8点前驶离或还枪,否则加收费用"),
		"character_string3": subscribeValue(spacesCodeByOrder(orderID)),
	}
	return sendWeChatSubscribeMessage(newRelocateCarV2TemplateID, "", order.Openid, data)
}

// carLeaveMessage 挪车通知（对齐 Java CarLeaveMessage）。
func carLeaveMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	data := map[string]map[string]string{
		"date3":             subscribeValue(nowWeChatTime()),
		"thing4":            subscribeValue("请及时驶离,否则将加收资源使用不当费用"),
		"character_string6": subscribeValue(spacesCodeByOrder(orderID)),
	}
	return sendWeChatSubscribeMessage(relocateCarTemplateID, "", order.Openid, data)
}

// sendCarLeaveMessage 余额不足驶离通知（对齐 Java sendCarLeaveMessage）。
func sendCarLeaveMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	data := map[string]map[string]string{
		"date3":             subscribeValue(nowWeChatTime()),
		"thing4":            subscribeValue("余额不足,充电已结束,请及时驶离"),
		"character_string6": subscribeValue(spacesCodeByOrder(orderID)),
	}
	return sendWeChatSubscribeMessage(relocateCarTemplateID, "", order.Openid, data)
}

// parkingTicketMessage 停车限免券开通通知（对齐 Java ParkingTicketMessage，openid 为酒店员工 openid）。
func parkingTicketMessage(openid string) bool {
	hotel := repository.HotelByStaffOpenid(openid)
	if hotel == nil {
		logger.Mylog.Warn().Msgf("[订阅消息] 未找到员工 %s 对应的酒店", openid)
		return false
	}
	data := map[string]map[string]string{
		"thing1":       subscribeValue(hotel.HotelName + "停车场"),
		"thing2":       subscribeValue("次数限免券"),
		"short_thing4": subscribeValue("999张"),
	}
	return sendWeChatSubscribeMessage(parkingTicketTemplateID, "", openid, data)
}

// sendChargingMessage 充电开始提醒（对齐 Java sendChargingMessage，openid 为用户 openid）。
func sendChargingMessage(openid string) bool {
	orders := repository.OrdersByOpenidAndState(openid, repository.OrderStateUse)
	if len(orders) == 0 {
		logger.Mylog.Warn().Msgf("[订阅消息] 未找到用户 %s 的使用中订单", openid)
		return false
	}
	order := orders[0]

	var gunID int32
	page := ""
	if order.ConsumptionType == repository.ConsumeTypeNeighbor {
		if place := repository.PrivatePlaceByID(order.SpacesID); place != nil {
			gunID = place.ChargingGunID
			page = "/neighborCharge/pages/neighborCharge/OpenCharging/OpenCharging?orderid=" + order.Orderid + "&ProductID="
		}
	} else if s := repository.SpacesByID(order.SpacesID); s != nil {
		gunID = s.ChargingGunID
	}

	gun := repository.ChargingGunByID(gunID)
	pid, gunNumber := "", ""
	if gun != nil {
		pid = gun.ProductID
		gunNumber = gun.ProductID + gun.Direction
	}
	if page != "" {
		page += pid
	}

	data := map[string]map[string]string{
		"thing8":             subscribeValue(order.ConsumptionLocation),
		"time9":              subscribeValue(formatWeChatTime(order.BeginTime)),
		"character_string12": subscribeValue(gunNumber),
		"character_string1":  subscribeValue(pid),
		"thing14":            subscribeValue("充电已开始,如果有疑问,请联系我们"),
	}
	return sendWeChatSubscribeMessage(chargingTemplateID, page, openid, data)
}

// sendChargingSettleMessage 充电结算通知（对齐 Java sendChargingSettleMessage）。
func sendChargingSettleMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	beginTime, overTime := "", ""
	if order.BeginTime != nil {
		beginTime = order.BeginTime.Format("15:04:05")
	}
	if order.OverTime != nil {
		overTime = order.OverTime.Format("15:04:05")
	}
	data := map[string]map[string]string{
		"character_string6": subscribeValue(order.Orderid),
		"thing1":            subscribeValue(order.ConsumptionLocation),
		"thing8":            subscribeValue(order.ChargingDegree.String() + "度," + order.BasicConsumption.String() + "元"),
		"time9":             subscribeValue(beginTime + " ~ " + overTime),
		"thing19":           subscribeValue("您的充电已完成,感谢您的支持"),
	}
	return sendWeChatSubscribeMessage(chargingSettleTemplateID, "", order.Openid, data)
}

// sendChargingExceptionMessage 充电异常通知（对齐 Java sendChargingExceptionMessage）。
func sendChargingExceptionMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	data := map[string]map[string]string{
		"character_string1": subscribeValue(order.Orderid),
		"character_string2": subscribeValue(order.Pid),
		"number3":           subscribeValue(order.Lockid),
		"thing4":            subscribeValue("充电功率超出上限"),
	}
	return sendWeChatSubscribeMessage(powerExceptionTemplateID, "", order.Openid, data)
}

// sendOrderOverMessage 邻享充电邻居/桩主订单结束通知（对齐 Java SubscribeMessageServiceImpl.sendOrderOverMessage）。
func sendOrderOverMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		logger.Mylog.Warn().Msgf("[订阅消息] 未找到订单, orderId=%s", orderID)
		return false
	}

	page := ""
	if order.ConsumptionType == repository.ConsumeTypeNeighbor {
		op := repository.OrderPrivateByOrderid(orderID)
		place := repository.PrivatePlaceByID(order.SpacesID)
		if place != nil {
			if gun := repository.ChargingGunByID(place.ChargingGunID); gun != nil {
				if op != nil && op.IsPrivateUser != 1 {
					page = "/neighborCharge/pages/neighborCharge/neighborSendMsgToOwner/neighborSendMsgToOwner?spacesCode=" + place.SpacesCode + "&orderid=" + orderID
				} else {
					page = "/neighborCharge/pages/neighborCharge/OpenCharging/OpenCharging?orderid=" + orderID + "&ProductID=" + gun.ProductID
				}
			}
		}
	}

	data := map[string]map[string]string{
		"character_string1": subscribeValue(orderID),
		"thing2":            subscribeValue(order.ConsumptionLocation),
		"time9":             subscribeValue(formatWeChatTime(order.BeginTime)),
		"time8":             subscribeValue(nowWeChatTime()),
		"thing7":            subscribeValue("充电完成请半小时内驶离，否则收超时占位费"),
	}

	return sendWeChatSubscribeMessage(orderOverTemplateID, page, order.Openid, data)
}

// sendOwnerOrderOverMessage 邻享充电桩主订单结束通知（对齐 Java SubscribeMessageServiceImpl.sendOwnerOrderOverMessage）。
func sendOwnerOrderOverMessage(orderID, openid string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		logger.Mylog.Warn().Msgf("[订阅消息] 未找到订单, orderId=%s", orderID)
		return false
	}

	page := ""
	if order.ConsumptionType == repository.ConsumeTypeNeighbor {
		if place := repository.PrivatePlaceByID(order.SpacesID); place != nil {
			if gun := repository.ChargingGunByID(place.ChargingGunID); gun != nil {
				page = "/neighborCharge/pages/neighborCharge/returnInformation/returnInformation?orderid=" + orderID + "&ProductID=" + gun.ProductID
			}
		}
	}

	data := map[string]map[string]string{
		"character_string1": subscribeValue(orderID),
		"thing2":            subscribeValue(order.ConsumptionLocation),
		"time9":             subscribeValue(formatWeChatTime(order.BeginTime)),
		"time8":             subscribeValue(nowWeChatTime()),
		"thing7":            subscribeValue("您的充电桩已完成充电，请即时查看"),
	}

	return sendWeChatSubscribeMessage(ownerOrderOverTemplateID, page, openid, data)
}

// sendMoveCarMessage 挪车通知（对齐 Java sendMoveCarMessage）：更新 move_car_time 并发送订阅消息。
func sendMoveCarMessage(carNumber, phone string) bool {
	user, err := repository.UserByPhone(context.Background(), phone)
	if err != nil || user.Openid == "" {
		logger.Mylog.Warn().Msgf("[订阅消息] 未找到手机号 %s 对应的用户或未绑定 openid", phone)
		return false
	}
	if latest, err := repository.LatestNeighborOrderByOpenid(user.Openid); err == nil {
		repository.UpdateOrderMoveCarTime(latest.Orderid)
	}

	data := map[string]map[string]string{
		"car_number1": subscribeValue(carNumber),
		"date3":       subscribeValue(nowWeChatTime()),
		"thing4":      subscribeValue("您提供的车位无效或已被其他车辆占用！"),
		"thing8":      subscribeValue("正在对您收取不当占位费，请及时离开充电位"),
	}
	return sendWeChatSubscribeMessage(moveCarTemplateID, "", user.Openid, data)
}

// returnGunMessage 充电枪归位提醒（对齐 Java ReturnGunMessage）。
func returnGunMessage(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order == nil {
		return false
	}
	s := repository.SpacesByID(order.SpacesID)
	if s == nil || s.ChargingGunID == 0 {
		return false
	}
	gun := repository.ChargingGunByID(s.ChargingGunID)
	if gun == nil {
		return false
	}

	remark := "充电完成，请及时还枪，否则将加收费用"
	if isAfterNextMidnight(time.Now()) {
		remark = "充电完成，于8点前还枪，否则将加收费用"
	}
	data := map[string]map[string]string{
		"character_string1": subscribeValue(gun.ProductID + gun.Direction),
		"time3":             subscribeValue(nowWeChatTime()),
		"thing5":            subscribeValue(remark),
	}
	return sendWeChatSubscribeMessage(returnGunTemplateID, "", order.Openid, data)
}

// setInfoToPrivateUserReq 对齐 Java setInfoToPrivateUserDto（通知桩主邻居车位信息）。
type setInfoToPrivateUserReq struct {
	CommunitySpacesNum string `json:"communitySpacesNum"`
	StartTime          string `json:"startTime"`
	EndTime            string `json:"endTime"`
	CarNumber          string `json:"carNumber"`
	Phone              string `json:"phone"`
	Openid             string `json:"openid"`
}

// sendPrivateShareMessage 私桩共享申请通知给桩主（对齐 Java sendPrivateShareMessage）。
func sendPrivateShareMessage(openid string, neighborID int32) bool {
	var rec model.NeighborShareUserTbl
	if err := repository.FindByID(&rec, neighborID).Error; err != nil {
		return false
	}
	var charging model.PrivateChargingTbl
	if err := repository.FindByID(&charging, rec.OwnerUserID).Error; err != nil {
		return false
	}
	place, _ := repository.PrivatePlaceBySpacesCode(charging.SpacesNum)
	placeName := ""
	if place != nil {
		placeName = place.Place
	}
	page := "neighborCharge/pages/neighborCharge/ShareWish/ShareWish?id=" + strconv.Itoa(int(neighborID))
	data := map[string]map[string]string{
		"thing4":            subscribeValue(placeName),
		"character_string5": subscribeValue(charging.SpacesNum),
		"phrase3":           subscribeValue("申请"),
		"time2":             subscribeValue(nowWeChatTimeSeconds()),
	}
	return sendWeChatSubscribeMessage(privateChargingShareTemplateID, page, openid, data)
}

// sendPrivateShareResultMessage 私桩共享申请结果通知给申请人（对齐 Java sendPrivateShareResultMessage）。
func sendPrivateShareResultMessage(user model.NeighborShareUserTbl) bool {
	var charging model.PrivateChargingTbl
	if err := repository.FindByID(&charging, user.OwnerUserID).Error; err != nil {
		return false
	}
	place, _ := repository.PrivatePlaceBySpacesCode(charging.SpacesNum)
	placeName := ""
	if place != nil {
		placeName = place.Place
	}
	phrase := "申请未通过"
	if user.Status == "1" {
		phrase = "申请通过"
	}
	data := map[string]map[string]string{
		"thing4":            subscribeValue(placeName),
		"character_string5": subscribeValue(charging.SpacesNum),
		"phrase3":           subscribeValue(phrase),
		"time2":             subscribeValue(nowWeChatTimeSeconds()),
	}
	return sendWeChatSubscribeMessage(privateChargingShareTemplateID, "neighborCharge/pages/neighborCharge/homePage/homePage", user.Openid, data)
}

// sendNeighborInfoToPrivateUser 邻居车位信息发送给桩主（对齐 Java sendNeighborInfoToPrivateUser）。
func sendNeighborInfoToPrivateUser(dto setInfoToPrivateUserReq) bool {
	data := map[string]map[string]string{
		"character_string29": subscribeValue(dto.CommunitySpacesNum),
		"thing1":             subscribeValue(dto.CarNumber),
		"phone_number30":     subscribeValue(dto.Phone),
		"time2":              subscribeValue(nowWeChatTimeSeconds()),
		"thing14":            subscribeValue("预计充电时段为:" + dto.StartTime + "~" + dto.EndTime),
	}
	page := "/neighborCharge/pages/neighborCharge/moveCarInformation/moveCarInformation?carNumber=" + dto.CarNumber + "&phone=" + dto.Phone
	return sendWeChatSubscribeMessage(neighborToPrivateUserTemplateID, page, dto.Openid, data)
}

// sendNewRelocateCarMessage 桩主给邻居发挪车通知（对齐 Java sendNewRelocateCarMessage）：更新挪车通知时间并发送。
func sendNewRelocateCarMessage(orderID, openid, isPrivateUser string) bool {
	remark := "半小时内未及时驶回将加收超时占位费"
	if isPrivateUser == "0" {
		remark = "感谢您提供的便利!"
	}
	data := map[string]map[string]string{
		"thing4": subscribeValue("我的车辆已驶离您的车位，请您亦及时驶回！"),
		"thing8": subscribeValue(remark),
		"time10": subscribeValue(nowWeChatTime()),
	}
	repository.UpdateOrderNeighborRelocateTime(orderID)
	return sendWeChatSubscribeMessage(newRelocateCarTemplateID, "", openid, data)
}

// init 把订阅消息发送函数注入 service（供延迟任务分发器/其它 service 层调用，打破 service→handler 循环依赖）。
func init() {
	service.SendChargingSettleMessageFn = sendChargingSettleMessage
	service.SendChargingExceptionMessageFn = sendChargingExceptionMessage
	service.SendChargingMessageFn = sendChargingMessage
	service.CarLeaveMessageFn = carLeaveMessage
	service.ParkingMessageFn = parkingMessage
	service.RelocateCarMessageV2Fn = relocateCarMessageV2
	service.RelocateCarAt8clockMessageV2Fn = relocateCarAt8clockMessageV2
}
