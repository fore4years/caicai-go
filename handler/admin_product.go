package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// productGatewayDTO 对应 Java model.GatewayBean 的序列化字段（驼峰）。
type productGatewayDTO struct {
	ID           int32          `json:"id,omitempty"`
	Mac          string         `json:"mac,omitempty"`
	Iccid        string         `json:"iccid,omitempty"`
	Name         string         `json:"name,omitempty"`
	InstallPlace string         `json:"installPlace,omitempty"`
	InstallTime  *LocalDateTime `json:"installTime,omitempty"`
}

// productDTO 对应 Java model.ProductBean 的序列化字段（驼峰）。
type productDTO struct {
	Pid            string             `json:"pid"`
	Nid            string             `json:"nid,omitempty"`
	Imei           string             `json:"imei,omitempty"`
	GatewayID      int32              `json:"gatewayId,omitempty"`
	Environment    string             `json:"environment,omitempty"`
	ProductionTime *LocalDateTime     `json:"productionTime,omitempty"`
	IsTwicecar     int32              `json:"isTwicecar,omitempty"`
	HasCpLine      int32              `json:"hasCpLine,omitempty"`
	Gateway        *productGatewayDTO `json:"gateway,omitempty"`
}

func productToDTO(p model.TabProduct) productDTO {
	dto := productDTO{
		Pid:            p.Pid,
		Nid:            p.Nid,
		Imei:           p.Imei,
		GatewayID:      p.GatewayID,
		Environment:    p.Environment,
		ProductionTime: timeToLocal(p.ProductionTime),
		IsTwicecar:     p.IsTwicecar,
		HasCpLine:      p.HasCpLine,
	}
	// 对齐 Java ProductBean.resultMap 中 <collection property="gateway" column="gateway_id">。
	if g := repository.GatewayByID(p.GatewayID); g != nil {
		dto.Gateway = &productGatewayDTO{
			ID:           g.ID,
			Mac:          g.Mac,
			Iccid:        g.Iccid,
			Name:         g.Name,
			InstallPlace: g.InstallPlace,
			InstallTime:  timeToLocal(g.InstallTime),
		}
	}
	return dto
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

// productPage 分页查询产品；ordered 为 true 时附加 Java 的自定义排序。
// getAll/search 使用自定义排序，searchByGatewayId 不带排序（对齐 Java 各方法的 .last(...) 差异）。
func productPage(c *gin.Context, gatewayID, tiaojian string, ordered bool) {
	current, size := parsePage(c)
	total, products := repository.ListProducts(gatewayID, tiaojian, pageOffset(current, size), size, ordered)

	list := make([]productDTO, 0, len(products))
	for _, p := range products {
		list = append(list, productToDTO(p))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetAll GET /system/product/getAll
func (a *AdminProductController) GetAll(c *gin.Context) {
	productPage(c, "", "", true)
}

// Search GET /system/product/search?tiaojian=
func (a *AdminProductController) Search(c *gin.Context) {
	productPage(c, "", c.Query("tiaojian"), true)
}

// SearchByGatewayId GET /system/product/searchByGatewayId?gatewayId=
func (a *AdminProductController) SearchByGatewayId(c *gin.Context) {
	productPage(c, c.Query("gatewayId"), "", false)
}

// GetByProductId GET /system/product/getByProductId?productId=
func (a *AdminProductController) GetByProductId(c *gin.Context) {
	p := repository.ProductByPid(c.Query("productId"))
	if p == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(*p)))
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
		Pid:            repository.NextProductID(header),
		Nid:            req.Nid,
		Imei:           req.Imei,
		GatewayID:      req.GatewayID,
		Environment:    env,
		ProductionTime: timePtr(time.Now()),
		IsTwicecar:     isTwicecar,
	}
	if err := repository.Create(&p).Error; err != nil {
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
	if repository.ProductByPid(req.Pid) == nil {
		c.JSON(http.StatusOK, ResultError(400, "产品不存在"))
		return
	}
	rowsAffected := repository.UpdateProductByPid(req.Pid, map[string]interface{}{
		"nid":         req.Nid,
		"Imei":        req.Imei,
		"environment": req.Environment,
		"gateway_id":  req.GatewayID,
	})
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// DeleteById DELETE /system/product/deleteById/:pid
func (a *AdminProductController) DeleteById(c *gin.Context) {
	pid := c.Param("pid")
	if repository.ProductByPid(pid) == nil {
		c.JSON(http.StatusOK, ResultError(400, "产品不存在"))
		return
	}
	repository.DeleteProductByPid(pid)
	c.JSON(http.StatusOK, ResultSuccess(true))
}
