package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
)

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
