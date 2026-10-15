package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/logger"
	"caicai-go/repository"
)

// MessageController 对应 Java MessageCon。
type MessageController struct{}

// GetMsgStatus 获取未读消息数（对应 Java MessageSer.getMsgStatus）。
func (m *MessageController) GetMsgStatus(c *gin.Context) {
	openid := c.Query("openid")
	count, err := repository.CountUnreadMessages(openid)
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
	}
	c.JSON(http.StatusOK, count)
}

// UpdateMsgStatus 将消息标记为已读（对应 Java MessageSer.updateMsgStatus）。
func (m *MessageController) UpdateMsgStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	rowsAffected, err := repository.MarkMessageRead(c.Request.Context(), int32(id))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, 0)
		return
	}
	c.JSON(http.StatusOK, rowsAffected)
}

// GetMsgs 获取消息列表（对应 Java MessageSer.getmsgs）。
func (m *MessageController) GetMsgs(c *gin.Context) {
	openid := c.Query("openid")
	msgs, err := repository.MessagesByOpenid(openid)
	if err != nil {
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
