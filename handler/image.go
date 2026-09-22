package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ImageController 对齐 Java controller.ImageController（图片表 tab_image）。
type ImageController struct{}

// GetById GET /image/getById?id= → 返回原始图片字节（image/jpeg，对齐 Java produces=IMAGE_JPEG_VALUE）。
// 图片不存在时返回空 200（对齐 Java 返回 null）。
func (i *ImageController) GetById(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	var img model.TabImage
	if err := conf.Db.Where("id = ?", id).First(&img).Error; err != nil {
		c.Data(http.StatusOK, "image/jpeg", nil)
		return
	}
	c.Data(http.StatusOK, "image/jpeg", img.Image)
}
