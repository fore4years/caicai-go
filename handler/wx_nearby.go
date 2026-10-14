package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/objects"
)

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
