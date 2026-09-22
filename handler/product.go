package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ProductController 对齐 Java controller.ProductController（小程序侧产品，base /product）。
type ProductController struct{}

// GetById GET /product/getById?id=（tab_product 主键为 pid，此处按 pid 查询）
func (p *ProductController) GetById(c *gin.Context) {
	var prod model.TabProduct
	if err := conf.Db.Where("pid = ?", c.Query("id")).First(&prod).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(prod)))
}

// GetByProductId GET /product/getByProductId?productId=
func (p *ProductController) GetByProductId(c *gin.Context) {
	var prod model.TabProduct
	if err := conf.Db.Where("pid = ?", c.Query("productId")).First(&prod).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(prod)))
}

// AddLockProduct POST /product/addLockProduct
func (p *ProductController) AddLockProduct(c *gin.Context) {
	(&AdminProductController{}).AddLockProduct(c)
}

// AddChargingProduct POST /product/addChargingProduct
func (p *ProductController) AddChargingProduct(c *gin.Context) {
	(&AdminProductController{}).AddChargingProduct(c)
}

// AddTwiceChargingProduct POST /product/addTwiceChargingProduct
func (p *ProductController) AddTwiceChargingProduct(c *gin.Context) {
	(&AdminProductController{}).AddTwiceChargingProduct(c)
}

// AddPrivateChargingProduct POST /product/addPrivateChargingProduct
func (p *ProductController) AddPrivateChargingProduct(c *gin.Context) {
	(&AdminProductController{}).AddPrivateChargingProduct(c)
}

// GetAll GET /product/getAll?current=&size=
func (p *ProductController) GetAll(c *gin.Context) {
	(&AdminProductController{}).GetAll(c)
}

// GetByImeiLike GET /product/getByImeiLike?imei=
func (p *ProductController) GetByImeiLike(c *gin.Context) {
	var rows []model.TabProduct
	conf.Db.Where("Imei LIKE ?", "%"+c.Query("imei")+"%").Find(&rows)
	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// GetNidAndGatewayByPid GET /product/getNidAndGatewayByPid?pid=
func (p *ProductController) GetNidAndGatewayByPid(c *gin.Context) {
	var prod model.TabProduct
	if err := conf.Db.Where("pid = ?", c.Query("pid")).First(&prod).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	var gw model.TabGateway
	conf.Db.Where("id = ?", prod.GatewayID).First(&gw)
	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"nid":        prod.Nid,
		"imei":       prod.Imei,
		"gatewayId":  prod.GatewayID,
		"gatewayMac": gw.Mac,
	}))
}
