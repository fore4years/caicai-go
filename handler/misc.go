package handler

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
)

// ServiceNumberController 对应 Java ServiceNumberCon（微信服务器验证）。
type ServiceNumberController struct{}

const wxToken = "basdlfdjfadgfucizbcx"

// CheckSignature 微信服务器验证（对应 Java ServiceNumberCon.checkSignature）。
func (s *ServiceNumberController) CheckSignature(c *gin.Context) {
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	strs := []string{wxToken, timestamp, nonce}
	sort.Strings(strs)
	h := sha1.Sum([]byte(strs[0] + strs[1] + strs[2]))
	if fmt.Sprintf("%x", h) == signature {
		c.String(http.StatusOK, echostr)
		return
	}
	c.String(http.StatusOK, "")
}

// WxFujinController 对应 Java wxFujinCon。
type WxFujinController struct{}

// GetLockid 根据车位 id 获取车位锁 id（对应 Java wxFujinSer.getLockid）。
func (w *WxFujinController) GetLockid(c *gin.Context) {
	placeid := c.Query("placeid")
	lock, err := objects.LockTbl.WithContext(c.Request.Context()).Where(objects.LockTbl.Placeid.Eq(placeid)).First()
	if err != nil {
		c.String(http.StatusOK, "")
		return
	}
	c.String(http.StatusOK, lock.Lockid)
}

// MessageController 对应 Java MessageCon。
type MessageController struct{}

// GetMsgStatus 获取未读消息数（对应 Java MessageSer.getMsgStatus）。
func (m *MessageController) GetMsgStatus(c *gin.Context) {
	openid := c.Query("openid")
	var count int64
	if err := conf.Db.Raw("SELECT count(*) FROM usermessage_tbl WHERE openid = ? AND status = '未读'", openid).Scan(&count).Error; err != nil {
		logger.Mylog.Err(err).Caller().Send()
	}
	c.JSON(http.StatusOK, count)
}

// UpdateMsgStatus 将消息标记为已读（对应 Java MessageSer.updateMsgStatus）。
func (m *MessageController) UpdateMsgStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	info, err := objects.UsermessageTbl.WithContext(c.Request.Context()).Where(objects.UsermessageTbl.ID.Eq(int32(id))).Update(objects.UsermessageTbl.Status, "已读")
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, 0)
		return
	}
	c.JSON(http.StatusOK, info.RowsAffected)
}

// GetMsgs 获取消息列表（对应 Java MessageSer.getmsgs）。
func (m *MessageController) GetMsgs(c *gin.Context) {
	openid := c.Query("openid")
	var msgs []model.UsermessageTbl
	if err := conf.Db.Where("openid = ?", openid).Order("msgTime desc").Find(&msgs).Error; err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, []UsermessageDTO{})
		return
	}
	result := make([]UsermessageDTO, 0, len(msgs))
	for _, msg := range msgs {
		result = append(result, UsermessageDTO{
			ID:      msg.ID,
			Openid:  msg.Openid,
			Message: msg.Message,
			MsgTime: msg.MsgTime,
			Status:  msg.Status,
		})
	}
	c.JSON(http.StatusOK, result)
}

// YezhuController 对应 Java YezhuCon（/chezhu）。
type YezhuController struct{}

// zhuceReq 业主注册车位请求参数（对齐 Java domain.Place）。
type zhuceReq struct {
	Ownerid         string `form:"ownerid" json:"ownerid"`
	Province        string `form:"province" json:"province"`
	City            string `form:"city" json:"city"`
	District        string `form:"district" json:"district"`
	Streer          string `form:"streer" json:"streer"`
	RelatedBuilding string `form:"relatedBuilding" json:"relatedBuilding"`
	Longitude       string `form:"longitude" json:"longitude"`
	Latitude        string `form:"latitude" json:"latitude"`
	Rate            string `form:"rate" json:"rate"`
	State           string `form:"state" json:"state"`
	FixType         string `form:"fixType" json:"fixType"`
	LedID           string `form:"ledId" json:"ledId"`
}

// Zhuce 业主注册车位（对应 Java YeZhuSer.zhuce）。
func (y *YezhuController) Zhuce(c *gin.Context) {
	ctx := c.Request.Context()
	var req zhuceReq
	_ = c.ShouldBind(&req)

	cityName := req.Province + req.City + req.District
	no := getCityNo(cityName)
	if no == "" {
		no = "888888"
	}

	// selectIdById: 取该城市编号下最大的 placeid
	var placeids []string
	if err := conf.Db.Raw("SELECT placeid FROM place_tbl WHERE placeid LIKE ? ORDER BY placeid DESC", no+"%").Scan(&placeids).Error; err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.String(http.StatusOK, "注册失败，ser返回可能为null")
		return
	}
	if len(placeids) == 0 {
		no = no + "00001"
	} else {
		xianzaiID := placeids[0]
		if len(xianzaiID) > 6 {
			n, _ := strconv.Atoi(xianzaiID[6:])
			no = no + fmt.Sprintf("%05d", n+1)
		} else {
			no = no + "00001"
		}
	}

	place := model.PlaceTbl{
		Placeid:         no,
		Ownerid:         req.Ownerid,
		Province:        req.Province,
		City:            req.City,
		District:        req.District,
		Streer:          req.Streer,
		RelatedBuilding: req.RelatedBuilding,
		Longitude:       req.Longitude,
		Latitude:        req.Latitude,
		Rate:            req.Rate,
		State:           req.State,
		FixType:         req.FixType,
		LedID:           req.LedID,
	}
	if err := objects.PlaceTbl.WithContext(ctx).Create(&place); err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.String(http.StatusOK, "注册失败，ser返回可能为null")
		return
	}
	c.String(http.StatusOK, "注册成功")
}

// GetLockList 获取所有车位锁（对应 Java YeZhuSer.getLockList）。
func (y *YezhuController) GetLockList(c *gin.Context) {
	var list []map[string]string
	if err := conf.Db.Raw("SELECT lockid, lockmac FROM lock_tbl").Scan(&list).Error; err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, []map[string]string{})
		return
	}
	c.JSON(http.StatusOK, list)
}

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

// 城市编号映射（diqubianhao.txt）懒加载。
var (
	cityMapOnce sync.Once
	cityMap     map[string]string
)

// getCityNo 根据「省+市+区」拼接的城市名查询行政区划编号（对应 Java diquToBianhao.getCityNo）。
func getCityNo(cityName string) string {
	cityMapOnce.Do(func() {
		cityMap = make(map[string]string)
		data, err := os.ReadFile("conf/diqubianhao.txt")
		if err != nil {
			logger.Mylog.Err(err).Caller().Send()
			return
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				cityMap[parts[1]] = parts[0]
			}
		}
	})
	return cityMap[cityName]
}
