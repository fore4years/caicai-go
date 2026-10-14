package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/objects"
)

// YunController 对应 Java yunCon（设备数据回调）。
type YunController struct{}

// Yun 设备数据回调（对应 Java yunCon.yun）。
// Java 中订阅消息发送逻辑已被注释，实际仅查询订单，这里还原其真实副作用。
func (y *YunController) Yun(c *gin.Context) {
	verify := c.Query("verify")

	var body struct {
		Type     string `json:"type"`
		DeviceID string `json:"deviceId"`
		Data     string `json:"data"`
	}
	_ = c.ShouldBindJSON(&body)

	if body.Data != "" {
		message := body.Data
		if decoded, err := base64.StdEncoding.DecodeString(body.Data); err == nil {
			message = string(decoded)
		}
		if message != "Heart" && strings.Contains(message, "zhiling") && strings.Contains(message, "close") {
			var o struct {
				Orderid string `json:"orderid"`
			}
			if err := json.Unmarshal([]byte(message), &o); err == nil && o.Orderid != "" {
				// faSongGuanSuoTiXing：查询订单（Java 中实际发送订阅消息的代码被注释）
				_, _ = objects.OrderTbl.WithContext(c.Request.Context()).Where(objects.OrderTbl.Orderid.Eq(o.Orderid)).First()
			}
		}
	}

	c.String(http.StatusOK, verify)
}
