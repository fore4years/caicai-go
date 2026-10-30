package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ProductController 对齐 Java controller.ProductController（小程序侧产品，base /product）。
type ProductController struct{}

// productNidGatewayDTO 对齐 Java domain.vo.ProductNidGatewayVO。
type productNidGatewayDTO struct {
	Nid     string `json:"nid"`
	Gateway string `json:"gateway"`
}

// productVoDTO 对齐 Java domain.vo.ProductVo（getAllRegisteredIds 的返回）。
type productVoDTO struct {
	RegisteredIds []string `json:"registeredIds"`
	LatestNumber  string   `json:"latestNumber"`
}

// GetById GET /product/getById?id=（tab_product 主键为 pid，此处按 pid 查询）
func (p *ProductController) GetById(c *gin.Context) {
	prod := repository.ProductByPid(c.Query("id"))
	if prod == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(*prod)))
}

// GetByProductId GET /product/getByProductId?productId=
func (p *ProductController) GetByProductId(c *gin.Context) {
	prod := repository.ProductByPid(c.Query("productId"))
	if prod == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(*prod)))
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
	if req.Imei == "" {
		c.JSON(http.StatusOK, ResultError(400, "该网关不存在,请检查"))
		return
	}
	gw := repository.GatewayByMac(req.Imei)
	if gw == nil {
		c.JSON(http.StatusOK, ResultError(400, "该网关不存在,请检查"))
		return
	}

	env := req.Environment
	if envOverride != "" {
		env = envOverride
	}

	prod := model.TabProduct{
		Pid:         repository.NextProductID(header),
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
	if err := repository.CreateProductFields(&prod, fields); err != nil {
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
	zero := int32(0)
	p.addProduct(c, "SCC", &zero, "")
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
	repository.Count(&model.TabProduct{}, &total)

	var rows []model.TabProduct
	repository.Page(&model.TabProduct{}, "", pageOffset(current, size), size, &rows)

	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetByImeiLike GET /product/getByImeiLike?imei=（空则返回全部，否则按 Imei 模糊查询）
func (p *ProductController) GetByImeiLike(c *gin.Context) {
	imei := strings.TrimSpace(c.Query("imei"))
	rows := repository.ProductsByImeiLike(imei)
	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// GetNidAndGatewayByPid GET /product/getNidAndGatewayByPid?pid=
func (p *ProductController) GetNidAndGatewayByPid(c *gin.Context) {
	prod := repository.ProductByPid(c.Query("pid"))
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
		gw := repository.GatewayByID(prod.GatewayID)
		if gw == nil {
			c.JSON(http.StatusOK, ResultError(400, "网关不存在"))
			return
		}
		gateway = gw.Mac
	}
	c.JSON(http.StatusOK, ResultSuccess(productNidGatewayDTO{Nid: nid, Gateway: gateway}))
}

// GetAllRegisteredIds GET /product/getAllRegisteredIds
// 返回已使用的充电桩编号列表（pid 前缀 SCC）与最新编号，对齐 Java ProductServiceImpl.getAllRegisteredIds。
func (p *ProductController) GetAllRegisteredIds(c *gin.Context) {
	latestNumber := repository.MaxSCCPid()
	registeredIds := repository.SCCPids()

	c.JSON(http.StatusOK, ResultSuccess(productVoDTO{RegisteredIds: registeredIds, LatestNumber: latestNumber}))
}

// GetNotUse GET /product/getNotUse 对齐 Java getNotUse：未被充电枪/车位锁使用且非二轮车的产品。
func (p *ProductController) GetNotUse(c *gin.Context) {
	rows := repository.NotUseProducts()
	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// GetNotUseByStations GET /product/getNotUseByStations 对齐 Java getNotUseByStations：二轮车充电站产品。
func (p *ProductController) GetNotUseByStations(c *gin.Context) {
	rows := repository.TwiceCarProducts()
	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// GetPidBySpacesId GET /product/getPidBySpacesId 对齐 Java getPidBySpacesId：
// 车位 → 充电枪 → 产品。
func (p *ProductController) GetPidBySpacesId(c *gin.Context) {
	prod, err := repository.ProductBySpacesID(c.Query("spacesId"))
	if err != nil || prod.Pid == "" {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(*prod)))
}
