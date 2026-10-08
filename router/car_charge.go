package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

// registerCarCharge 注册车辆预约/车牌（CarDetailsCon）与充电列表/二维码（ChargeListCon）路由。
func registerCarCharge(r *gin.Engine) {
	car := new(handler.CarDetailsController)
	charge := new(handler.ChargeListController)

	// 车辆预约 / 车牌（CarDetailsCon）
	r.Any("/wx/yuyue", car.Yuyue)
	r.Any("/wx/cancelyuyue", car.Cancelyuyue)
	r.Any("/wx/searchplate", car.Searchplate)
	r.Any("/wx/saveyyTime", car.SaveyyTime)
	r.Any("/wx/getyyTime", car.GetyyTime)
	r.Any("/wx/addSaveyyTime", car.AddSaveyyTime)
	r.Any("/wx/addCarPlate_num", car.AddCarPlateNum)
	r.Any("/wx/getUserCar", car.GetUserCar)
	r.Any("/wx/delCarMsg", car.DelCarMsg)

	// 充电列表 / 二维码（ChargeListCon）
	r.Any("/wx/getChargePlaces", charge.GetChargePlaces)
	r.Any("/wx/getDev_carlock", charge.GetDevCarlock)
	r.Any("/wx/getDevcharge", charge.GetDevcharge)
	r.Any("/wx/getSearchcharge", charge.GetSearchcharge)
	r.Any("/wx/getSearchcarlock", charge.GetSearchcarlock)
	r.Any("/wx/getiscarlock", charge.Getiscarlock)
	r.Any("/wx/selectAllcharge", charge.SelectAllcharge)
	r.Any("/wx/getinstallTime", charge.GetinstallTime)
	r.Any("/wx/updateiscaolock", charge.Updateiscaolock)
	r.Any("/wx/getByLockid", charge.GetByLockid)
	r.Any("/wx/getpByLockid", charge.GetpByLockid)
	r.Any("/wx/getProductIDBylockid", charge.GetProductIDBylockid)
	r.Any("/wx/getAllByProductID", charge.GetAllByProductID)
	r.Any("/wx/wode/coverOrder", charge.CoverOrder)
	r.Any("/wx/getChargePayMsg", charge.GetChargePayMsg)
	r.Any("/wx/getPlaceByLockid", charge.GetPlaceByLockid)
}
