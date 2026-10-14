package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============ CosController（/cos） ============

type CosController struct{}

func (c *CosController) Sts(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{})
}
