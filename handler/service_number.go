package handler

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
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
