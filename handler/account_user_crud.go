package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ UserBankCardController（/userBankCard） ============

type UserBankCardController struct{}

func (c *UserBankCardController) List(ctx *gin.Context) {
	var rows []model.UserBankCardTbl
	repository.FindAll(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *UserBankCardController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.UserBankCardTbl{}, &total)
	var rows []model.UserBankCardTbl
	repository.Page(&model.UserBankCardTbl{}, "create_time desc", pageOffset(current, size), size, &rows)
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

func (c *UserBankCardController) GetById(ctx *gin.Context) {
	var m model.UserBankCardTbl
	if err := repository.FindByID(&m, ctx.Param("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(m))
}

func (c *UserBankCardController) GetByOpenId(ctx *gin.Context) {
	m, err := repository.UserBankCardByOpenid(ctx.Param("openid"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(m))
}

func (c *UserBankCardController) Save(ctx *gin.Context) {
	var m model.UserBankCardTbl
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	m.Openid = ctx.GetString("openid")
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&m).Error == nil))
}

func (c *UserBankCardController) Update(ctx *gin.Context) {
	var m model.UserBankCardTbl
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := repository.UpdateByID(&model.UserBankCardTbl{}, m.ID, m)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *UserBankCardController) DeleteById(ctx *gin.Context) {
	res := repository.DeleteByID(&model.UserBankCardTbl{}, ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ UserBluetoothBindingsController（/userBluetoothBindings） ============

type UserBluetoothBindingsController struct{}

func (c *UserBluetoothBindingsController) List(ctx *gin.Context) {
	var rows []model.UserBluetoothBindingsTbl
	repository.FindAll(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *UserBluetoothBindingsController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.UserBluetoothBindingsTbl{}, &total)
	var rows []model.UserBluetoothBindingsTbl
	repository.Page(&model.UserBluetoothBindingsTbl{}, "create_time desc", pageOffset(current, size), size, &rows)
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

func (c *UserBluetoothBindingsController) GetById(ctx *gin.Context) {
	var m model.UserBluetoothBindingsTbl
	if err := repository.FindByID(&m, ctx.Param("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(m))
}

func (c *UserBluetoothBindingsController) GetByOpenId(ctx *gin.Context) {
	m, err := repository.UserBluetoothBindingsByOpenid(ctx.Param("openid"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(m))
}

func (c *UserBluetoothBindingsController) Save(ctx *gin.Context) {
	var m model.UserBluetoothBindingsTbl
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	m.BluetoothMac = strings.ReplaceAll(m.BluetoothMac, ":", "")
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&m).Error == nil))
}

func (c *UserBluetoothBindingsController) Update(ctx *gin.Context) {
	var m model.UserBluetoothBindingsTbl
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := repository.UpdateByID(&model.UserBluetoothBindingsTbl{}, m.ID, m)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *UserBluetoothBindingsController) DeleteById(ctx *gin.Context) {
	res := repository.DeleteByID(&model.UserBluetoothBindingsTbl{}, ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ AccountPublicController（/account/public） ============

type AccountPublicController struct{}

func (c *AccountPublicController) Add(ctx *gin.Context) {
	var a model.AccountPublicTbl
	if err := ctx.ShouldBindJSON(&a); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "公司账户信息新增失败"))
		return
	}
	if err := repository.Create(&a).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "公司账户信息新增失败"))
		return
	}
	ctx.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "公司账户信息新增成功", Data: true})
}

func (c *AccountPublicController) GetByOrderId(ctx *gin.Context) {
	a, err := repository.AccountPublicByOrderID(ctx.Param("orderId"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(404, "未找到对应的公司账户信息"))
		return
	}
	ctx.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "公司账户信息查询成功", Data: a})
}

// ============ AccountPrivateController（/account/private） ============

type AccountPrivateController struct{}

func (c *AccountPrivateController) Add(ctx *gin.Context) {
	var a model.AccountPrivateTbl
	if err := ctx.ShouldBindJSON(&a); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "私人账户信息新增失败"))
		return
	}
	if err := repository.Create(&a).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "私人账户信息新增失败"))
		return
	}
	ctx.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "私人账户信息新增成功", Data: true})
}

func (c *AccountPrivateController) GetByOrderId(ctx *gin.Context) {
	a, err := repository.AccountPrivateByOrderID(ctx.Param("orderId"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(404, "未找到对应的私人账户信息"))
		return
	}
	ctx.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "私人账户信息查询成功", Data: a})
}

// ============ BalanceRecordController（/balanceRecords） ============

type BalanceRecordController struct{}

func (c *BalanceRecordController) GetAll(ctx *gin.Context) {
	var rows []model.TabBalanceRecord
	repository.FindAll(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *BalanceRecordController) GetByPage(ctx *gin.Context) {
	current, size := parsePage(ctx)
	openid := ctx.GetString("openid")
	total, rows := repository.BalanceRecordsByOpenid(openid, pageOffset(current, size), size)
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

func (c *BalanceRecordController) GetById(ctx *gin.Context) {
	var m model.TabBalanceRecord
	if err := repository.FindByID(&m, ctx.Param("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(m))
}

func (c *BalanceRecordController) Save(ctx *gin.Context) {
	var m model.TabBalanceRecord
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&m).Error == nil))
}

func (c *BalanceRecordController) Update(ctx *gin.Context) {
	var m model.TabBalanceRecord
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := repository.UpdateByID(&model.TabBalanceRecord{}, m.ID, m)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *BalanceRecordController) DeleteById(ctx *gin.Context) {
	res := repository.DeleteByID(&model.TabBalanceRecord{}, ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
