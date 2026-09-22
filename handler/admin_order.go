package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// AdminOrderController 对齐 Java System.controller.OrderManageCon / OrderCon。
type AdminOrderController struct{}

// orderListQuery 构建订单列表查询条件（订单管理列表）。
func orderListQuery(orderid, openid, startDate, endDate, state, consumptionType, address string) *gorm.DB {
	db := conf.Db.Model(&model.OrderTbl{})
	if orderid != "" {
		db = db.Where("orderid = ?", orderid)
	}
	if openid != "" {
		db = db.Where("openid = ?", openid)
	}
	if startDate != "" {
		db = db.Where("begin_time >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("over_time <= ?", endDate+" 23:59:59")
	}
	if state != "" {
		db = db.Where("state = ?", state)
	}
	if consumptionType != "" {
		db = db.Where("consumption_type = ?", consumptionType)
	}
	if address != "" {
		db = db.Where("consumption_location LIKE ?", "%"+address+"%")
	}
	return db
}

// orderPidQuery 构建某 pid 下已完成订单的查询条件（充电记录）。
func orderPidQuery(pid, startDate, endDate string) *gorm.DB {
	db := conf.Db.Model(&model.OrderTbl{}).Where("pid = ?", pid).Where("state = ?", "已完成")
	if startDate != "" {
		db = db.Where("over_time >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("over_time <= ?", endDate)
	}
	return db
}

// GetOrderList GET /system/orderManage/list
func (a *AdminOrderController) GetOrderList(c *gin.Context) {
	current, size := parsePage(c)
	orderid := c.Query("orderid")
	openid := c.Query("openid")
	phone := c.Query("phone")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	state := c.Query("state")
	consumptionType := c.Query("consumptionType")
	address := c.Query("address")

	// 提供手机号时先查用户 openid；查不到返回空结果。
	if phone != "" {
		var u model.UserTbl
		if err := conf.Db.Where("phone = ?", phone).First(&u).Error; err != nil {
			c.JSON(http.StatusOK, ResultSuccessPage(0, []OrderDTO{}))
			return
		}
		openid = u.Openid
	}

	var total int64
	orderListQuery(orderid, openid, startDate, endDate, state, consumptionType, address).Count(&total)

	var orders []model.OrderTbl
	orderListQuery(orderid, openid, startDate, endDate, state, consumptionType, address).
		Order("reserve_time desc").
		Offset(pageOffset(current, size)).Limit(size).Find(&orders)

	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetOrderStatistics GET /system/orderManage/statistics
func (a *AdminOrderController) GetOrderStatistics(c *gin.Context) {
	pid := c.Query("pid")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	statsQuery := func(today bool) *gorm.DB {
		db := conf.Db.Model(&model.OrderTbl{}).Where("state = ?", "已完成")
		if pid != "" {
			db = db.Where("pid = ?", pid)
		}
		if today {
			t := time.Now().Format("2006-01-02")
			db = db.Where("begin_time >= ?", t+" 00:00:00").Where("over_time <= ?", t+" 23:59:59")
		} else {
			if startDate != "" {
				db = db.Where("begin_time >= ?", startDate)
			}
			if endDate != "" {
				db = db.Where("over_time <= ?", endDate+" 23:59:59")
			}
		}
		return db
	}

	type agg struct {
		Degree float64         `gorm:"column:d"`
		Fee    decimal.Decimal `gorm:"column:f"`
	}

	var totalOrders, todayOrders int64
	statsQuery(false).Count(&totalOrders)
	statsQuery(true).Count(&todayOrders)

	var totalAgg, todayAgg agg
	statsQuery(false).Select("COALESCE(SUM(charging_degree),0) AS d, COALESCE(SUM(total_price),0) AS f").Scan(&totalAgg)
	statsQuery(true).Select("COALESCE(SUM(charging_degree),0) AS d, COALESCE(SUM(total_price),0) AS f").Scan(&todayAgg)

	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalOrders": totalOrders,
		"totalDegree": totalAgg.Degree,
		"totalFee":    totalAgg.Fee,
		"todayOrders": todayOrders,
		"todayDegree": todayAgg.Degree,
		"todayFee":    todayAgg.Fee,
	}))
}

// GetOrderDetail GET /system/orderManage/detail?orderId=
func (a *AdminOrderController) GetOrderDetail(c *gin.Context) {
	orderID := c.Query("orderId")
	var order model.OrderTbl
	if err := conf.Db.Where("orderid = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(order)))
}

// ExportOrderData GET /system/orderManage/export（对齐 Java：仅返回导出成功提示，不产生真实文件）
func (a *AdminOrderController) ExportOrderData(c *gin.Context) {
	pid := c.Query("pid")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	state := c.Query("state")

	db := conf.Db.Model(&model.OrderTbl{})
	if pid != "" {
		db = db.Where("pid = ?", pid)
	}
	if startDate != "" {
		db = db.Where("begin_time >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("over_time <= ?", endDate+" 23:59:59")
	}
	if state != "" {
		db = db.Where("state = ?", state)
	}

	var count int64
	db.Count(&count)
	c.JSON(http.StatusOK, ResultSuccess("订单数据导出成功，共"+strconv.FormatInt(count, 10)+"条记录"))
}

// BatchDeleteOrders DELETE /system/orderManage/batchDelete，body 为 orderid 数组。
func (a *AdminOrderController) BatchDeleteOrders(c *gin.Context) {
	var orderIds []string
	if err := c.ShouldBindJSON(&orderIds); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除订单失败"))
		return
	}
	if len(orderIds) == 0 {
		c.JSON(http.StatusOK, ResultSuccess(true))
		return
	}
	if err := conf.Db.Where("orderid IN ?", orderIds).Delete(&model.OrderTbl{}).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除订单失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// orderSum 汇总某列（charging_degree / basic_consumption），返回字符串；无数据返回 nil。
func orderSum(pid, startDate, endDate, column string, today bool) *string {
	q := "SELECT COALESCE(SUM(" + column + "),0) FROM order_tbl WHERE state = '已完成'"
	var args []interface{}
	if pid != "" {
		q += " AND pid = ?"
		args = append(args, pid)
	}
	if today {
		t := time.Now().Format("2006-01-02")
		q += " AND over_time >= ? AND over_time <= ?"
		args = append(args, t+" 00:00:00", t+" 23:59:59")
	} else {
		if startDate != "" {
			q += " AND over_time >= ?"
			args = append(args, startDate)
		}
		if endDate != "" {
			q += " AND over_time <= ?"
			args = append(args, endDate)
		}
	}

	var v decimal.Decimal
	if err := conf.Db.Raw(q, args...).Row().Scan(&v); err != nil {
		return nil
	}
	s := v.String()
	return &s
}

// GetOrderDegree GET /system/order/getOrderDegree
func (a *AdminOrderController) GetOrderDegree(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(orderSum(c.Query("pid"), c.Query("startDate"), c.Query("endDate"), "charging_degree", false)))
}

// GetOrderDegreeByToday GET /system/order/getOrderDegreeByToday
func (a *AdminOrderController) GetOrderDegreeByToday(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(orderSum(c.Query("pid"), "", "", "charging_degree", true)))
}

// GetOrderFee GET /system/order/getOrderFee
func (a *AdminOrderController) GetOrderFee(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(orderSum(c.Query("pid"), c.Query("startDate"), c.Query("endDate"), "basic_consumption", false)))
}

// GetOrderFeeByToday GET /system/order/getOrderFeeByToday
func (a *AdminOrderController) GetOrderFeeByToday(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(orderSum(c.Query("pid"), "", "", "basic_consumption", true)))
}

// GetOrdersByPidAndDate GET /system/order/getOrdersByPidAndDate
func (a *AdminOrderController) GetOrdersByPidAndDate(c *gin.Context) {
	current, size := parsePage(c)
	pid := c.Query("pid")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	var total int64
	orderPidQuery(pid, startDate, endDate).Count(&total)

	var orders []model.OrderTbl
	orderPidQuery(pid, startDate, endDate).
		Order("begin_time desc").
		Offset(pageOffset(current, size)).Limit(size).Find(&orders)

	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}
