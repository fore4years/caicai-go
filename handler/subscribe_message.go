package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/logger"
	"caicai-go/repository"
)

// ============ SubscribeMessageController（/subscribeMessage） ============

type SubscribeMessageController struct{}

// SelectRid 查询 rid（微信接口，省略外呼，返回空串）。
func (c *SubscribeMessageController) SelectRid(ctx *gin.Context) {
	ctx.String(http.StatusOK, "")
}

func (c *SubscribeMessageController) SendParkingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendRelocateCarMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendParkingTicketMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingSettleMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingExceptionMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendOrderOverMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}

// SendMoveCarMessage 对齐 Java sendMoveCarMessage：更新 move_car_time。
func (c *SubscribeMessageController) SendMoveCarMessage(ctx *gin.Context) {
	var req struct {
		CarNumber string `json:"carNumber"`
		Phone     string `json:"phone"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	user, err := repository.UserByPhone(ctx.Request.Context(), req.Phone)
	if err != nil || user.Openid == "" {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if latest, err := repository.LatestNeighborOrderByOpenid(user.Openid); err == nil {
		repository.UpdateOrderMoveCarTime(latest.Orderid)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ 微信订阅消息发送（对齐 Java SubscribeMessageServiceImpl） ============

const (
	// 订阅消息模板 ID
	orderOverTemplateID      = "t0NC2308XRVwfJcVCW54rJqOpfhTQYYI1J7v-e_f9oI" // 订单结束通知
	ownerOrderOverTemplateID = "t0NC2308XRVwfJcVCW54rJqOpfhTQYYI1J7v-e_f9oI" // 桩主订单结束通知（同模板）
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
		"time8":             subscribeValue(time.Now().Format("2006年01月02日 15:04")),
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
		"time8":             subscribeValue(time.Now().Format("2006年01月02日 15:04")),
		"thing7":            subscribeValue("您的充电桩已完成充电，请即时查看"),
	}

	return sendWeChatSubscribeMessage(ownerOrderOverTemplateID, page, openid, data)
}
