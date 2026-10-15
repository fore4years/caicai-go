package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/repository"
)

// AdminRevenueController 对齐 Java System.controller.MoneyGetCon（/system 收益统计）。
// 注意：MoneyGetCon 是遗留控制器，多数方法返回原始 Map/String/List（不包 Result）。
type AdminRevenueController struct{}

// queryDecimal 读取查询参数并解析为 decimal.Decimal，缺失或解析失败返回 0。
func queryDecimal(c *gin.Context, key string) decimal.Decimal {
	if v := c.Query(key); v != "" {
		if d, err := decimal.NewFromString(v); err == nil {
			return d
		}
	}
	return decimal.Zero
}

// MoneyPlaces /system/moneyPlaces → {max, list}（原始 Map）
func (a *AdminRevenueController) MoneyPlaces(c *gin.Context) {
	current, size := parsePage(c)

	max := repository.CountPlacesNotState("待加锁")
	places := repository.PagePlacesNotState("待加锁", pageOffset(current, size), size)

	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
}

// MoneyOwner /system/moneyOwner?ownerid= → owner 实体（原始对象）
func (a *AdminRevenueController) MoneyOwner(c *gin.Context) {
	owner, err := repository.OwnerByOwnerid(c.Request.Context(), c.Query("ownerid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, owner)
}

// GetMoney /system/getMoney → {all, today}（原始 Map，值为 basic_consumption 总和）
func (a *AdminRevenueController) GetMoney(c *gin.Context) {
	all := repository.SumFinishedOrdersBasicConsumption()
	todaySum := repository.SumFinishedOrdersTodayBasicConsumption()

	c.JSON(http.StatusOK, gin.H{"all": all, "today": todaySum})
}

// GetOrders /system/getOrders?page=&length=&lockid= → {max, list}（原始 Map）
func (a *AdminRevenueController) GetOrders(c *gin.Context) {
	current, size := parsePage(c)
	lockid := c.Query("lockid")

	max, orders := repository.FinishedOrdersByLockid(lockid, pageOffset(current, size), size)

	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
}

// GetNickname /system/getNickname?openid= → 昵称（原始 String）
func (a *AdminRevenueController) GetNickname(c *gin.Context) {
	u, err := repository.UserByOpenid(c.Request.Context(), c.Query("openid"))
	if err != nil {
		c.String(http.StatusOK, "")
		return
	}
	c.String(http.StatusOK, u.NickName)
}

// AllAndToday /system/allAndToday?lockid= → {all, today}（原始 Map）
func (a *AdminRevenueController) AllAndToday(c *gin.Context) {
	lockid := c.Query("lockid")
	all := repository.SumFinishedOrdersByLockidBasicConsumption(lockid)
	todaySum := repository.SumFinishedOrdersByLockidTodayBasicConsumption(lockid)

	c.JSON(http.StatusOK, gin.H{"all": all, "today": todaySum})
}

// OrderDel /system/orderDel，body {orderid} → "ok"（原始 String）
func (a *AdminRevenueController) OrderDel(c *gin.Context) {
	var req struct {
		Orderid string `json:"orderid"`
	}
	_ = c.ShouldBindJSON(&req)
	repository.DeleteOrderByOrderid(req.Orderid)
	c.String(http.StatusOK, "ok")
}

// GetTotalIncome /system/getTotalIncome?spacesCode=&type= → {totalIncome}（原始 Map）
func (a *AdminRevenueController) GetTotalIncome(c *gin.Context) {
	spacesID, consumptionType, ok := repository.RevenueSpaces(c.Query("spacesCode"), c.Query("type"))
	if !ok {
		c.JSON(http.StatusOK, gin.H{"totalIncome": 0})
		return
	}

	total := repository.SumFinishedOrdersBySpacesBasicConsumption(spacesID, consumptionType)

	c.JSON(http.StatusOK, gin.H{"totalIncome": total})
}

// GetTodayIncome /system/getTodayIncome?spacesCode=&type= → {todayIncome}（原始 Map）
func (a *AdminRevenueController) GetTodayIncome(c *gin.Context) {
	spacesID, consumptionType, ok := repository.RevenueSpaces(c.Query("spacesCode"), c.Query("type"))
	if !ok {
		c.JSON(http.StatusOK, gin.H{"todayIncome": 0})
		return
	}

	todayIncome := repository.SumFinishedOrdersBySpacesTodayBasicConsumption(spacesID, consumptionType)

	c.JSON(http.StatusOK, gin.H{"todayIncome": todayIncome})
}

// GetOrderDetails /system/getOrderDetails?spacesCode=&type= → 订单明细列表（原始 List<Map>）
func (a *AdminRevenueController) GetOrderDetails(c *gin.Context) {
	orders, _, ok := repository.RevenueOrders(c.Query("spacesCode"), c.Query("type"))
	if !ok {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

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
	max, orders, todayTotalIncome := repository.TodayFinishedOrders(page, length)

	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list, "todayTotalIncome": todayTotalIncome})
}

// excelOverview 对齐 Java excelImpl.buildData（概览 Sheet 单行汇总）。
type excelOverview struct {
	StartTime      *LocalDateTime  `json:"startTime"`
	EndTime        *LocalDateTime  `json:"endTime"`
	ElectronNum    decimal.Decimal `json:"electronNum"`
	ElectronFee    decimal.Decimal `json:"electronFee"`
	ServiceFee     decimal.Decimal `json:"serviceFee"`
	OvertimeUseFee decimal.Decimal `json:"overtimeUseFee"`
}

// excelDetailRow 对齐 Java excelImpl.buildDetailData（收益明细 Sheet 每行 9 列）。
type excelDetailRow struct {
	OrderNo        string          `json:"orderNo"`
	StartTime      *LocalDateTime  `json:"startTime"`
	EndTime        *LocalDateTime  `json:"endTime"`
	UserOpenID     string          `json:"userOpenId"`
	Amount         decimal.Decimal `json:"amount"`
	ElectronNum    decimal.Decimal `json:"electronNum"`
	ElectronFee    decimal.Decimal `json:"electronFee"`
	ServiceFee     decimal.Decimal `json:"serviceFee"`
	OvertimeUseFee decimal.Decimal `json:"overtimeUseFee"`
}

// parseExcelDate 将 yyyy-MM-dd 转为 LocalDateTime；startOfDay 为 true 取 00:00:00，否则 23:59:59。
// 空或非法日期返回 nil（对齐 Java parseDate）。
func parseExcelDate(date string, startOfDay bool) *LocalDateTime {
	if strings.TrimSpace(date) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil
	}
	if !startOfDay {
		t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	v := LocalDateTime(t)
	return &v
}

// ExcelExport GET /system/excel_export
// Java 使用 FesodSheet 输出二进制 Excel（概览 + 收益明细两个 Sheet）。Go 无 Excel 依赖，
// 此处按相同业务逻辑组装概览与明细数据并以 JSON 返回（对齐 excelImpl.buildData/buildDetailData）。
func (a *AdminRevenueController) ExcelExport(c *gin.Context) {
	overview := excelOverview{
		StartTime:      parseExcelDate(c.Query("startDate"), true),
		EndTime:        parseExcelDate(c.Query("endDate"), false),
		ElectronNum:    queryDecimal(c, "chargingDegree"),
		ElectronFee:    queryDecimal(c, "powerRate"),
		ServiceFee:     queryDecimal(c, "serviceRate"),
		OvertimeUseFee: queryDecimal(c, "timeoutCar"),
	}

	orders, consumptionType, _ := repository.RevenueOrders(c.Query("spacesCode"), c.Query("type"))
	details := make([]excelDetailRow, 0, len(orders))
	for _, o := range orders {
		var electronFee, serviceFee decimal.Decimal
		if consumptionType == "邻享充电" {
			electronFee, serviceFee = repository.PrivateElectronicByOrderidAndPrivateUser(o.Orderid)
		} else {
			electronFee, serviceFee = electronicFeesByOrderid(o.Orderid)
		}
		overtimeUseFee := o.BasicConsumption.Sub(electronFee).Sub(serviceFee)
		if overtimeUseFee.IsNegative() {
			overtimeUseFee = decimal.Zero
		}
		details = append(details, excelDetailRow{
			OrderNo:        o.Orderid,
			StartTime:      timeToLocal(o.BeginTime),
			EndTime:        timeToLocal(o.OverTime),
			UserOpenID:     o.Openid,
			Amount:         o.BasicConsumption,
			ElectronNum:    o.ChargingDegree,
			ElectronFee:    electronFee,
			ServiceFee:     serviceFee,
			OvertimeUseFee: overtimeUseFee,
		})
	}

	c.JSON(http.StatusOK, gin.H{"overview": overview, "details": details})
}
