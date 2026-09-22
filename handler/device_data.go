package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// DeviceDataController 对齐 Java controller.DeviceDataController（设备数据接收，HTTP 兼容入口）。
type DeviceDataController struct{}

type deviceDataResponse struct {
	Result  int    `json:"result"`
	Message string `json:"message,omitempty"`
}

// Receive POST /api/device/data，body {act, imei, ts, data}。
func (d *DeviceDataController) Receive(c *gin.Context) {
	var req struct {
		Act  string `json:"act"`
		Imei string `json:"imei"`
		Ts   int64  `json:"ts"`
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, deviceDataResponse{Result: 1, Message: "参数解析失败"})
		return
	}
	if req.Act != "udata" {
		c.JSON(http.StatusOK, deviceDataResponse{Result: 1, Message: "不支持的act类型"})
		return
	}

	// 解析 data（对齐 Java：DES 解密后的 JSON）。此处直接按 JSON 解析，解密逻辑待补充。
	row := model.DeviceDataTbl{Imei: req.Imei, Ts: req.Ts}
	var inner map[string]interface{}
	if err := json.Unmarshal([]byte(req.Data), &inner); err == nil {
		for k, v := range inner {
			field := k
			if idx := strings.Index(k, ":"); idx >= 0 {
				field = k[idx+1:]
			}
			val := parseFloat(v)
			switch field {
			case "ept":
				row.Ept = val
			case "ua":
				row.Ua = val
			case "ub":
				row.Ub = val
			case "uc":
				row.Uc = val
			case "ia":
				row.Ia = val
			case "ib":
				row.Ib = val
			case "ic":
				row.Ic = val
			case "pt":
				row.Pt = val
			case "pft":
				row.Pft = val
			case "qt":
				row.Qt = val
			case "st":
				row.St = val
			}
		}
	}

	if err := conf.Db.Create(&row).Error; err != nil {
		c.JSON(http.StatusOK, deviceDataResponse{Result: 1, Message: "处理失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, deviceDataResponse{Result: 0})
}

func parseFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}
