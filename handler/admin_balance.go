package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
	"caicai-go/repository"
)

// AdminBalanceController 对齐 Java System.controller.BalanceDetailCon。
type AdminBalanceController struct{}

// List GET /system/balanceDetail/list
// 对齐 Java BalanceDetailSerImpl.getList：过滤 openid/phone 为空的数据，按 openid 升序。
func (a *AdminBalanceController) List(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	phone := c.Query("phone")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.UserTbl{}).
			Where("openid IS NOT NULL AND openid <> '' AND phone IS NOT NULL AND phone <> ''")
		if openid != "" {
			db = db.Where("openid LIKE ?", "%"+openid+"%")
		}
		if phone != "" {
			db = db.Where("phone LIKE ?", "%"+phone+"%")
		}
		return db
	}

	var total int64
	query().Count(&total)
	var users []model.UserTbl
	query().Order("openid asc").Offset(pageOffset(current, size)).Limit(size).Find(&users)

	list := make([]UserDTO, 0, len(users))
	for _, u := range users {
		list = append(list, userToDTO(u))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// BalanceSource GET /system/balanceDetail/balanceSource?openid=
// 对齐 Java BalanceDetailSerImpl.getBalanceSource：当前余额 = 充值来源(可退) + 其他来源(不可退)。
func (a *AdminBalanceController) BalanceSource(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	var u model.UserTbl
	if err := conf.Db.Where("openid = ?", openid).First(&u).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "获取余额来源失败: 用户不存在"))
		return
	}
	currentBalance := u.Balans

	// 查询"已支付"且 remaining_refund_amount > 0 的充值订单，汇总为充值来源（可退）
	var orders []model.RechargeOrderTbl
	conf.Db.Where("openid = ? AND order_status = ? AND remaining_refund_amount > ?", openid, "已支付", decimal.Zero).
		Find(&orders)

	rechargeSource := decimal.Zero
	for _, o := range orders {
		rechargeSource = rechargeSource.Add(o.RemainingRefundAmount)
	}
	otherSource := currentBalance.Sub(rechargeSource)

	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"currentBalance":     currentBalance,
		"rechargeSource":     rechargeSource,
		"otherSource":        otherSource,
		"rechargeOrderCount": len(orders),
	}))
}

// Recharge POST /system/balanceDetail/recharge?openid=&amount=
// 对齐 Java BalanceDetailSerImpl.rechargeBalance：用户存在则 balans += amount。
// 金额用 decimal 解析并传字符串，配合 CAST(? AS DECIMAL(10,2)) 避免 MySQL DECIMAL→DOUBLE 精度丢失。
func (a *AdminBalanceController) Recharge(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	amount, err := decimal.NewFromString(c.Query("amount"))
	if err != nil || !amount.IsPositive() {
		c.JSON(http.StatusOK, ResultError(400, "充值金额必须大于0"))
		return
	}
	var u model.UserTbl
	if err := conf.Db.Where("openid = ?", openid).First(&u).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "充值余额失败: 用户不存在"))
		return
	}
	amt := amount.StringFixed(2)
	if err := conf.Db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(
			"UPDATE user_tbl SET balans = balans + CAST(? AS DECIMAL(10,2)) WHERE openid = ?",
			amt, openid,
		).Error
	}); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "充值余额失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// Deduct POST /system/balanceDetail/deduct?openid=&amount=
// 对齐 Java BalanceDetailSerImpl.deductBalance：走统一余额消费逻辑
// （优先扣其他来源，不足部分按 FIFO 扣充值订单 remaining_refund_amount，并释放冻结余额）。
func (a *AdminBalanceController) Deduct(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	amount, err := decimal.NewFromString(c.Query("amount"))
	if err != nil || !amount.IsPositive() {
		c.JSON(http.StatusOK, ResultError(400, "扣除金额必须大于0"))
		return
	}
	if !repository.ConsumeBalance(openid, amount) {
		c.JSON(http.StatusOK, ResultError(500, "扣除余额失败，用户openid: "+openid+"，扣除金额: "+amount.String()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}
