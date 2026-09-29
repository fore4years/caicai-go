package handler

import (
	"crypto/des"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// desDecrypt DES/ECB/PKCS5Padding 解密（对齐 Java DesUtil.decrypt，默认密钥 lanlan12）。
func desDecrypt(encryptedBase64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", err
	}
	block, err := des.NewCipher([]byte("lanlan12"))
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw)%block.BlockSize() != 0 {
		return "", errors.New("DES密文长度非块对齐")
	}
	out := make([]byte, len(raw))
	for i := 0; i < len(raw); i += block.BlockSize() {
		block.Decrypt(out[i:i+block.BlockSize()], raw[i:i+block.BlockSize()])
	}
	// PKCS5 去填充
	pad := int(out[len(out)-1])
	if pad == 0 || pad > block.BlockSize() || pad > len(out) {
		return "", errors.New("DES填充无效")
	}
	return string(out[:len(out)-pad]), nil
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

	// 对齐 Java：先 DES 解密，再解析 JSON。
	decrypted, err := desDecrypt(req.Data)
	if err != nil {
		c.JSON(http.StatusOK, deviceDataResponse{Result: 1, Message: "处理失败: " + err.Error()})
		return
	}

	var inner map[string]interface{}
	if err := json.Unmarshal([]byte(decrypted), &inner); err != nil || len(inner) == 0 {
		// 数据为空时不落库（对齐 Java processDeviceData 的 early return）。
		c.JSON(http.StatusOK, deviceDataResponse{Result: 0})
		return
	}

	row := model.DeviceDataTbl{Imei: req.Imei, Ts: req.Ts, CreateTime: time.Now()}
	for k, v := range inner {
		if v == nil {
			continue
		}
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
