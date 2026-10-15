package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// AdminOrderController 对齐 Java System.controller.OrderManageCon / OrderCon。
type AdminOrderController struct{}

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
		u, err := repository.UserByPhone(c.Request.Context(), phone)
		if err != nil {
			c.JSON(http.StatusOK, ResultSuccessPage(0, []OrderDTO{}))
			return
		}
		openid = u.Openid
	}

	total, orders := repository.ListOrders(orderid, openid, startDate, endDate, state, consumptionType, address, pageOffset(current, size), size)

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

	totalOrders, todayOrders, totalDegree, totalFee, todayDegree, todayFee := repository.OrderStatistics(pid, startDate, endDate)

	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalOrders": totalOrders,
		"totalDegree": totalDegree,
		"totalFee":    totalFee,
		"todayOrders": todayOrders,
		"todayDegree": todayDegree,
		"todayFee":    todayFee,
	}))
}

// GetOrderDetail GET /system/orderManage/detail?orderId=
func (a *AdminOrderController) GetOrderDetail(c *gin.Context) {
	orderID := c.Query("orderId")
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(*order)))
}

// ExportOrderData GET /system/orderManage/export（对齐 Java：仅返回导出成功提示，不产生真实文件）
func (a *AdminOrderController) ExportOrderData(c *gin.Context) {
	pid := c.Query("pid")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	state := c.Query("state")

	count := repository.CountOrdersByFilters(pid, startDate, endDate, state)
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
	if err := repository.DeleteOrdersByIDs(orderIds); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除订单失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// GetOrderDegree GET /system/order/getOrderDegree
func (a *AdminOrderController) GetOrderDegree(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(repository.OrderSum(c.Query("pid"), c.Query("startDate"), c.Query("endDate"), "charging_degree", false)))
}

// GetOrderDegreeByToday GET /system/order/getOrderDegreeByToday
func (a *AdminOrderController) GetOrderDegreeByToday(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(repository.OrderSum(c.Query("pid"), "", "", "charging_degree", true)))
}

// GetOrderFee GET /system/order/getOrderFee
func (a *AdminOrderController) GetOrderFee(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(repository.OrderSum(c.Query("pid"), c.Query("startDate"), c.Query("endDate"), "basic_consumption", false)))
}

// GetOrderFeeByToday GET /system/order/getOrderFeeByToday
func (a *AdminOrderController) GetOrderFeeByToday(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(repository.OrderSum(c.Query("pid"), "", "", "basic_consumption", true)))
}

// GetOrdersByPidAndDate GET /system/order/getOrdersByPidAndDate
func (a *AdminOrderController) GetOrdersByPidAndDate(c *gin.Context) {
	current, size := parsePage(c)
	pid := c.Query("pid")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	total, orders := repository.ListOrdersByPid(pid, startDate, endDate, pageOffset(current, size), size)

	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}
