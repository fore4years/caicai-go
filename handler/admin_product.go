package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// productDTO 对应 Java model.ProductBean 的序列化字段（驼峰）。
type productDTO struct {
	Pid            string         `json:"pid"`
	Nid            string         `json:"nid,omitempty"`
	Imei           string         `json:"imei,omitempty"`
	GatewayID      int32          `json:"gatewayId,omitempty"`
	Environment    string         `json:"environment,omitempty"`
	ProductionTime *LocalDateTime `json:"productionTime,omitempty"`
	IsTwicecar     int32          `json:"isTwicecar,omitempty"`
	HasCpLine      int32          `json:"hasCpLine,omitempty"`
}

func productToDTO(p model.TabProduct) productDTO {
	return productDTO{
		Pid:            p.Pid,
		Nid:            p.Nid,
		Imei:           p.Imei,
		GatewayID:      p.GatewayID,
		Environment:    p.Environment,
		ProductionTime: timeToLocal(p.ProductionTime),
		IsTwicecar:     p.IsTwicecar,
		HasCpLine:      p.HasCpLine,
	}
}

// productReq 新增/更新请求体（对齐 Java ProductDTO，字段名用前端使用的驼峰）。
type productReq struct {
	Nid         string `json:"nid"`
	Imei        string `json:"imei"`
	GatewayID   int32  `json:"gatewayId"`
	Environment string `json:"environment"`
	Pid         string `json:"pid"`
}

// AdminProductController 对齐 Java System.controller.ProductCon。
type AdminProductController struct{}

func productListQuery(gatewayID, tiaojian string) *gorm.DB {
	db := conf.Db.Model(&model.TabProduct{})
	if gatewayID != "" {
		db = db.Where("gateway_id = ?", gatewayID)
	}
	if tiaojian != "" {
		db = db.Where("pid LIKE ?", "%"+tiaojian+"%")
	}
	return db
}

func productPage(c *gin.Context, gatewayID, tiaojian string) {
	current, size := parsePage(c)
	var total int64
	productListQuery(gatewayID, tiaojian).Count(&total)

	var products []model.TabProduct
	productListQuery(gatewayID, tiaojian).
		Order("SUBSTRING(pid, 1, 3) ASC, CAST(SUBSTRING(pid, 4) AS UNSIGNED) DESC").
		Offset(pageOffset(current, size)).Limit(size).Find(&products)

	list := make([]productDTO, 0, len(products))
	for _, p := range products {
		list = append(list, productToDTO(p))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetAll GET /system/product/getAll
func (a *AdminProductController) GetAll(c *gin.Context) {
	productPage(c, "", "")
}

// Search GET /system/product/search?tiaojian=
func (a *AdminProductController) Search(c *gin.Context) {
	productPage(c, "", c.Query("tiaojian"))
}

// SearchByGatewayId GET /system/product/searchByGatewayId?gatewayId=
func (a *AdminProductController) SearchByGatewayId(c *gin.Context) {
	productPage(c, c.Query("gatewayId"), "")
}

// GetByProductId GET /system/product/getByProductId?productId=
func (a *AdminProductController) GetByProductId(c *gin.Context) {
	var p model.TabProduct
	if err := conf.Db.Where("pid = ?", c.Query("productId")).First(&p).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(p)))
}

// nextProductID 生成下一个产品编号：header + 8 位递增序号（对齐 Java getNextProductId）。
func nextProductID(header string) string {
	var maxPid sql.NullString
	conf.Db.Raw("SELECT MAX(pid) FROM tab_product WHERE pid LIKE ?", header+"%").Row().Scan(&maxPid)
	if !maxPid.Valid || maxPid.String == "" {
		return header + "00000001"
	}
	pid := strings.TrimPrefix(maxPid.String, header)
	n, err := strconv.Atoi(pid)
	if err != nil {
		return header + "00000001"
	}
	return header + fmt.Sprintf("%08d", n+1)
}

func (a *AdminProductController) addProduct(c *gin.Context, header string, isTwicecar int32, envOverride string) {
	var req productReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	env := req.Environment
	if envOverride != "" {
		env = envOverride
	}
	p := model.TabProduct{
		Pid:            nextProductID(header),
		Nid:            req.Nid,
		Imei:           req.Imei,
		GatewayID:      req.GatewayID,
		Environment:    env,
		ProductionTime: time.Now(),
		IsTwicecar:     isTwicecar,
	}
	if err := conf.Db.Create(&p).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// AddLockProduct POST /system/product/addLockProduct
func (a *AdminProductController) AddLockProduct(c *gin.Context) {
	a.addProduct(c, "SCL", 0, "")
}

// AddChargingProduct POST /system/product/addChargingProduct
func (a *AdminProductController) AddChargingProduct(c *gin.Context) {
	a.addProduct(c, "SCC", 0, "")
}

// AddTwiceChargingProduct POST /system/product/addTwiceChargingProduct
func (a *AdminProductController) AddTwiceChargingProduct(c *gin.Context) {
	a.addProduct(c, "SCC", 1, "")
}

// AddPrivateChargingProduct POST /system/product/addPrivateChargingProduct
func (a *AdminProductController) AddPrivateChargingProduct(c *gin.Context) {
	a.addProduct(c, "SCC", 0, "私桩")
}

// Update POST /system/product/update
func (a *AdminProductController) Update(c *gin.Context) {
	var req productReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	var p model.TabProduct
	if err := conf.Db.Where("pid = ?", req.Pid).First(&p).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(400, "产品不存在"))
		return
	}
	res := conf.Db.Model(&model.TabProduct{}).Where("pid = ?", req.Pid).Updates(map[string]interface{}{
		"nid":         req.Nid,
		"Imei":        req.Imei,
		"environment": req.Environment,
		"gateway_id":  req.GatewayID,
	})
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// DeleteById DELETE /system/product/deleteById/:pid
func (a *AdminProductController) DeleteById(c *gin.Context) {
	pid := c.Param("pid")
	var p model.TabProduct
	if err := conf.Db.Where("pid = ?", pid).First(&p).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(400, "产品不存在"))
		return
	}
	conf.Db.Where("pid = ?", pid).Delete(&model.TabProduct{})
	c.JSON(http.StatusOK, ResultSuccess(true))
}
