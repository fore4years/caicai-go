package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// AdminRevenueController 对齐 Java System.controller.MoneyGetCon（/system 收益统计）。
// 注意：MoneyGetCon 是遗留控制器，多数方法返回原始 Map/String/List（不包 Result）。
type AdminRevenueController struct{}

// revenueSpaces 根据 type 解析消费类型与车位 id：
//   - "驿享充电" 查 tab_parking_spaces
//   - "邻享充电" 查 private_place_tbl
//
// 返回 spacesID、consumptionType、是否命中；未命中时返回 false。
func revenueSpaces(spacesCode, typ string) (int32, string, bool) {
	switch typ {
	case "驿享充电":
		var s model.TabParkingSpace
		if err := conf.Db.Where("spaces_code = ?", spacesCode).First(&s).Error; err != nil {
			return 0, "", false
		}
		return s.ID, "驿享充电", true
	case "邻享充电":
		var s model.PrivatePlaceTbl
		if err := conf.Db.Where("spaces_code = ?", spacesCode).First(&s).Error; err != nil {
			return 0, "", false
		}
		return s.ID, "邻享充电", true
	default:
		return 0, "", false
	}
}

// sumBasicConsumption 对给定订单查询条件求和 basic_consumption（对齐 Java getAll）。
func sumBasicConsumption(db *gorm.DB) decimal.Decimal {
	var v decimal.Decimal
	if err := db.Select("COALESCE(SUM(basic_consumption), 0)").Row().Scan(&v); err != nil {
		return decimal.Zero
	}
	return v
}

// MoneyPlaces /system/moneyPlaces → {max, list}（原始 Map）
func (a *AdminRevenueController) MoneyPlaces(c *gin.Context) {
	current, size := parsePage(c)

	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Where("state != ?", "待加锁").Count(&max)

	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Where("state != ?", "待加锁").
		Offset(pageOffset(current, size)).Limit(size).Find(&places)

	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
}

// MoneyOwner /system/moneyOwner?ownerid= → owner 实体（原始对象）
func (a *AdminRevenueController) MoneyOwner(c *gin.Context) {
	var owner model.OwnerTbl
	if err := conf.Db.Where("ownerid = ?", c.Query("ownerid")).First(&owner).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, owner)
}

// GetMoney /system/getMoney → {all, today}（原始 Map，值为 basic_consumption 总和）
func (a *AdminRevenueController) GetMoney(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	all := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).Where("state = ?", "已完成"))
	todaySum := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("state = ?", "已完成").Where("over_time LIKE ?", today+"%"))

	c.JSON(http.StatusOK, gin.H{"all": all, "today": todaySum})
}

// GetOrders /system/getOrders?page=&length=&lockid= → {max, list}（原始 Map）
func (a *AdminRevenueController) GetOrders(c *gin.Context) {
	current, size := parsePage(c)
	lockid := c.Query("lockid")

	var max int64
	conf.Db.Model(&model.OrderTbl{}).Where("state = ? AND lockid = ?", "已完成", lockid).Count(&max)

	var orders []model.OrderTbl
	conf.Db.Model(&model.OrderTbl{}).Where("state = ? AND lockid = ?", "已完成", lockid).
		Offset(pageOffset(current, size)).Limit(size).Find(&orders)

	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
}

// GetNickname /system/getNickname?openid= → 昵称（原始 String）
func (a *AdminRevenueController) GetNickname(c *gin.Context) {
	var u model.UserTbl
	if err := conf.Db.Where("openid = ?", c.Query("openid")).First(&u).Error; err != nil {
		c.String(http.StatusOK, "")
		return
	}
	c.String(http.StatusOK, u.NickName)
}

// AllAndToday /system/allAndToday?lockid= → {all, today}（原始 Map）
func (a *AdminRevenueController) AllAndToday(c *gin.Context) {
	lockid := c.Query("lockid")
	today := time.Now().Format("2006-01-02")

	all := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).Where("state = ? AND lockid = ?", "已完成", lockid))
	todaySum := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("state = ? AND lockid = ?", "已完成", lockid).Where("over_time LIKE ?", today+"%"))

	c.JSON(http.StatusOK, gin.H{"all": all, "today": todaySum})
}

// OrderDel /system/orderDel，body {orderid} → "ok"（原始 String）
func (a *AdminRevenueController) OrderDel(c *gin.Context) {
	var req struct {
		Orderid string `json:"orderid"`
	}
	_ = c.ShouldBindJSON(&req)
	conf.Db.Where("orderid = ?", req.Orderid).Delete(&model.OrderTbl{})
	c.String(http.StatusOK, "ok")
}

// GetTotalIncome /system/getTotalIncome?spacesCode=&type= → {totalIncome}（原始 Map）
func (a *AdminRevenueController) GetTotalIncome(c *gin.Context) {
	spacesID, consumptionType, ok := revenueSpaces(c.Query("spacesCode"), c.Query("type"))
	if !ok {
		c.JSON(http.StatusOK, gin.H{"totalIncome": 0})
		return
	}

	total := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("spaces_id = ? AND consumption_type = ? AND state = ?", spacesID, consumptionType, "已完成"))

	c.JSON(http.StatusOK, gin.H{"totalIncome": total})
}

// GetTodayIncome /system/getTodayIncome?spacesCode=&type= → {todayIncome}（原始 Map）
func (a *AdminRevenueController) GetTodayIncome(c *gin.Context) {
	spacesID, consumptionType, ok := revenueSpaces(c.Query("spacesCode"), c.Query("type"))
	if !ok {
		c.JSON(http.StatusOK, gin.H{"todayIncome": 0})
		return
	}

	today := time.Now().Format("2006-01-02")
	todayIncome := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("spaces_id = ? AND consumption_type = ? AND state = ?", spacesID, consumptionType, "已完成").
		Where("over_time LIKE ?", today+"%"))

	c.JSON(http.StatusOK, gin.H{"todayIncome": todayIncome})
}

// GetOrderDetails /system/getOrderDetails?spacesCode=&type= → 订单明细列表（原始 List<Map>）
func (a *AdminRevenueController) GetOrderDetails(c *gin.Context) {
	spacesID, consumptionType, ok := revenueSpaces(c.Query("spacesCode"), c.Query("type"))
	if !ok {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	var orders []model.OrderTbl
	conf.Db.Model(&model.OrderTbl{}).
		Where("spaces_id = ? AND consumption_type = ? AND state = ?", spacesID, consumptionType, "已完成").
		Order("begin_time desc").Find(&orders)

	result := make([]gin.H, 0, len(orders))
	for _, o := range orders {
		result = append(result, gin.H{
			"orderid":          o.Orderid,
			"beginTime":        timeToLocal(o.BeginTime),
			"overTime":         timeToLocal(o.OverTime),
			"openid":           o.Openid,
			"totalPrice":       o.TotalPrice,
			"basicConsumption": o.BasicConsumption,
			"chargingDegree":   o.ChargingDegree,
			"consumptionType":  o.ConsumptionType,
		})
	}
	c.JSON(http.StatusOK, result)
}

// GetTodayOrders /system/getTodayOrders?page=&length= → {max, list, todayTotalIncome}（原始 Map）
func (a *AdminRevenueController) GetTodayOrders(c *gin.Context) {
	page := queryInt(c, "page", 0)
	length := queryInt(c, "length", 0)
	today := time.Now().Format("2006-01-02")

	todayQuery := func() *gorm.DB {
		return conf.Db.Model(&model.OrderTbl{}).
			Where("over_time LIKE ? AND state = ?", today+"%", "已完成").
			Order("over_time desc")
	}

	var max int64
	todayQuery().Count(&max)

	var orders []model.OrderTbl
	db := todayQuery()
	if page > 0 && length > 0 {
		db = db.Offset((page - 1) * length).Limit(length)
	}
	db.Find(&orders)

	todayTotalIncome := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("over_time LIKE ? AND state = ?", today+"%", "已完成"))

	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list, "todayTotalIncome": todayTotalIncome})
}

// ExcelExport GET /system/excel_export（Java 为二进制 Excel 流，这里按计划返回 JSON 桩）
func (a *AdminRevenueController) ExcelExport(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess("导出成功"))
}
