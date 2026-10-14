package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ OwnerChargeController（base /wx，owner/chargeCon） ============

// 对齐 Java ReturnConstantEnum 的返回码。
const (
	chargeObjectNull   = 10000 // 对象为空
	chargeRepeatCharge = -9    // 充电桩不能重复安装
	chargeRepeatPlace  = -10   // 车位不能重复安装充电桩
	chargeInstalled    = 10001 // 车位已安装充电桩
	chargeUninstalled  = 10002 // 车位未安装充电桩
)

// OwnerChargeController 对齐 Java owner.controller.chargeCon。
type OwnerChargeController struct{}

// AddCharge /wx/addCharge?placeid=&ownerid= —— 对齐 chargeAddSerImpl.addCharge。
func (c *OwnerChargeController) AddCharge(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.addCharge(ctx.Query("placeid"), ctx.Query("ownerid"))))
}

func (c *OwnerChargeController) addCharge(placeid, ownerid string) int {
	var existing model.ChargeTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&existing).Error; err == nil {
		return chargeRepeatPlace // -10
	}
	str := "您为车位：" + placeid + "提交的安装充电桩申请正在审核，请保持电话畅通，以便于我们的工作人员与您取得联系~~~"
	conf.Db.Create(&model.MessageTbl{
		Ownerid: ownerid,
		Message: str,
		Msgtime: time.Now().Format("2006-01-02 15:04:05"),
		Status:  "未读",
	})
	res := conf.Db.Create(&model.ChargeTbl{Placeid: placeid, Charge: "待审核"})
	return int(res.RowsAffected)
}

// AddChargeTest /wx/addChargeTest?placeid=&ownerid=&nid=&direction=&ammeter= —— 扫码添加充电桩。
func (c *OwnerChargeController) AddChargeTest(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.addChargeTest(
		ctx.Query("placeid"), ctx.Query("ownerid"), ctx.Query("nid"), ctx.Query("direction"), ctx.Query("ammeter"),
	)))
}

func (c *OwnerChargeController) addChargeTest(placeid, ownerid, nid, direction, ammeter string) int {
	// 业主和车位是否匹配
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ? AND ownerid = ?", placeid, ownerid).First(&place).Error; err != nil {
		return chargeObjectNull // 10000
	}
	// 充电桩是否重复添加（nid + direction 且 chargeid 已分配）
	var cm model.ChargeMessageTbl
	if err := conf.Db.Where("nid = ? AND direction = ? AND chargeid IS NOT NULL", nid, direction).First(&cm).Error; err == nil {
		return chargeRepeatCharge // -9
	}
	// 车位是否重复添加
	var cm2 model.ChargeMessageTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&cm2).Error; err == nil {
		return chargeRepeatPlace // -10
	}
	// 生成充电桩编号（对齐 incrementAndFormatNumber：去掉前 2 位后 +1，再格式化为 SC%06d）
	chargeid := ""
	var maxChargeID string
	if err := conf.Db.Raw("SELECT MAX(chargeid) FROM charge_message_tbl WHERE LENGTH(chargeid) = 10").Row().Scan(&maxChargeID); err == nil && maxChargeID != "" {
		if len(maxChargeID) >= 2 {
			if n, e := strconv.Atoi(maxChargeID[2:]); e == nil {
				chargeid = fmt.Sprintf("SC%06d", n+1)
			}
		}
	}
	res := conf.Db.Exec(
		"INSERT INTO charge_message_tbl (chargeid, placeid, nid, direction, ammeter) VALUES (?, ?, ?, ?, ?)",
		chargeid, placeid, nid, direction, ammeter,
	)
	return int(res.RowsAffected)
}

// IsAddCharge /wx/isAddCharge?placeid= —— 对齐 chargeAddSerImpl.isAddCharge。
func (c *OwnerChargeController) IsAddCharge(ctx *gin.Context) {
	var cm model.ChargeMessageTbl
	if err := conf.Db.Where("placeid = ?", ctx.Query("placeid")).First(&cm).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(chargeUninstalled)) // 10002
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargeInstalled)) // 10001
}
