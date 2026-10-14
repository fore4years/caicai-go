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

// Add POST /image/add 对齐 Java ImageController.add：JSON 提交图片字节落库。
func (i *ImageController) Add(c *gin.Context) {
	var req model.TabImage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Create(&req)
	c.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

// GetAll GET /image/getAll 对齐 Java ImageController.getAll：分页查询所有图片。
func (i *ImageController) GetAll(c *gin.Context) {
	current, size := parsePage(c)
	var total int64
	conf.Db.Model(&model.TabImage{}).Count(&total)
	var rows []model.TabImage
	conf.Db.Model(&model.TabImage{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}
