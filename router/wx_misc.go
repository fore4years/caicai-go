package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

// registerWxMisc 注册员工 / 故障上报 / 车位锁 / 自建 / 业主充电相关路由。
func registerWxMisc(r *gin.Engine) {
	staff := new(handler.StaffController)
	r.GET("/wx/selectname", staff.Selectname)
	r.GET("/wx/getState", staff.GetState)

	guzhang := new(handler.WxGuzhangController)
	r.POST("/wx/guzhangtijiao", guzhang.GuzhangTijiao)
	r.GET("/wx/getAllfault", guzhang.GetAllFault)
	r.GET("/wx/updateState", guzhang.UpdateState)
	r.GET("/wx/getMaintenance_time", guzhang.GetMaintenanceTime)

	cheweisuo := new(handler.CheweisuoController)
	r.GET("/cheweisuo/kaisuo", cheweisuo.Kaisuo)

	lockc := new(handler.HardwareLockController)
	r.POST("/wx/lock/open/scan/phone", lockc.OpenScanPhone)
	r.POST("/wx/lock/close/scan/phone", lockc.CloseScanPhone)
	r.POST("/wx/lock/open/scan/lanya", lockc.OpenScanLanya)

	zijian := new(handler.ZijianController)
	r.GET("/zijian/shibai", zijian.Shibai)
	r.GET("/zijian/chenggong", zijian.Chenggong)

	charge := new(handler.OwnerChargeController)
	r.GET("/wx/addCharge", charge.AddCharge)
	r.GET("/wx/addChargeTest", charge.AddChargeTest)
	r.GET("/wx/isAddCharge", charge.IsAddCharge)
}
