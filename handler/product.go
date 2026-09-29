package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ProductController 对齐 Java controller.ProductController（小程序侧产品，base /product）。
type ProductController struct{}

// productNidGatewayDTO 对齐 Java domain.vo.ProductNidGatewayVO。
type productNidGatewayDTO struct {
	Nid     string `json:"nid"`
	Gateway string `json:"gateway"`
}

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

// addProduct 小程序侧新增产品：按 imei 作为网关 mac 查询网关，取网关 id 写入 gateway_id。
// 对齐 Java ProductServiceImpl.getGateway + addXxxProduct。
// isTwicecar 为 nil 时该列不写入（Java builder 未设置，落库为 NULL）。
func (p *ProductController) addProduct(c *gin.Context, header string, isTwicecar *int32, envOverride string) {
	var req productReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}

	// Java getGateway：imei 非空时按 mac 查询网关，查不到抛 CustomException("该网关不存在,请检查")。
	var gw model.TabGateway
	if req.Imei == "" {
		c.JSON(http.StatusOK, ResultError(400, "该网关不存在,请检查"))
		return
	}
	if err := conf.Db.Where("mac = ?", req.Imei).First(&gw).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(400, "该网关不存在,请检查"))
		return
	}

	env := req.Environment
	if envOverride != "" {
		env = envOverride
	}

	prod := model.TabProduct{
		Pid:         nextProductID(header),
		Nid:         req.Nid,
		Imei:        req.Imei,
		GatewayID:   gw.ID,
		Environment: env,
	}
	fields := []string{"pid", "nid", "Imei", "gateway_id", "environment"}
	if isTwicecar != nil {
		prod.IsTwicecar = *isTwicecar
		fields = append(fields, "is_twicecar")
	}
	if err := conf.Db.Select(fields).Create(&prod).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// AddLockProduct POST /product/addLockProduct
func (p *ProductController) AddLockProduct(c *gin.Context) {
	p.addProduct(c, "SCL", nil, "")
}

// AddChargingProduct POST /product/addChargingProduct
func (p *ProductController) AddChargingProduct(c *gin.Context) {
	p.addProduct(c, "SCC", nil, "")
}

// AddTwiceChargingProduct POST /product/addTwiceChargingProduct
func (p *ProductController) AddTwiceChargingProduct(c *gin.Context) {
	one := int32(1)
	p.addProduct(c, "SCC", &one, "")
}

// AddPrivateChargingProduct POST /product/addPrivateChargingProduct
func (p *ProductController) AddPrivateChargingProduct(c *gin.Context) {
	zero := int32(0)
	p.addProduct(c, "SCC", &zero, "私桩")
}

// GetAll GET /product/getAll?current=&size=（Java 无自定义排序，仅分页）
func (p *ProductController) GetAll(c *gin.Context) {
	current, size := parsePage(c)
	var total int64
	conf.Db.Model(&model.TabProduct{}).Count(&total)

	var rows []model.TabProduct
	conf.Db.Model(&model.TabProduct{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetByImeiLike GET /product/getByImeiLike?imei=（空则返回全部，否则按 Imei 模糊查询）
func (p *ProductController) GetByImeiLike(c *gin.Context) {
	imei := strings.TrimSpace(c.Query("imei"))
	var rows []model.TabProduct
	if imei == "" {
		conf.Db.Find(&rows)
	} else {
		conf.Db.Where("Imei LIKE ?", "%"+imei+"%").Find(&rows)
	}
	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// GetNidAndGatewayByPid GET /product/getNidAndGatewayByPid?pid=
func (p *ProductController) GetNidAndGatewayByPid(c *gin.Context) {
	prod := productByPid(c.Query("pid"))
	if prod == nil {
		c.JSON(http.StatusOK, ResultError(400, "产品不存在"))
		return
	}

	var nid, gateway string
	if prod.Imei != "" {
		nid = prod.Imei
		gateway = prod.Imei
	} else {
		nid = prod.Nid
		gw := gatewayByID(prod.GatewayID)
		if gw == nil {
			c.JSON(http.StatusOK, ResultError(400, "网关不存在"))
			return
		}
		gateway = gw.Mac
	}
	c.JSON(http.StatusOK, ResultSuccess(productNidGatewayDTO{Nid: nid, Gateway: gateway}))
}
