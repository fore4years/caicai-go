package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

// registerChargeStation 注册充电站（ChargeStation）与二轮车订单电量（ReOrderPower）路由。
func registerChargeStation(r *gin.Engine) {
	chargeStation := new(handler.ChargeStationController)
	reOrderPower := new(handler.ReOrderPowerController)

	cs := r.Group("/ChargeStation")
	{
		cs.GET("/getChargeStation", chargeStation.GetChargeStation)
		cs.GET("/getById", chargeStation.GetById)
		cs.GET("/getByOpenid", chargeStation.GetByOpenid)
		cs.POST("/add", chargeStation.Add)
		cs.GET("/getByPoint", chargeStation.GetByPoint)
		cs.GET("/getStationList", chargeStation.GetStationList)
	}

	rop := r.Group("/receptacle/order/power")
	{
		rop.GET("/add", reOrderPower.Add)
		rop.GET("/getById", reOrderPower.GetById)
		rop.GET("/getAll", reOrderPower.GetAll)
		rop.GET("/getValueByOrderid", reOrderPower.GetValueByOrderid)
		rop.GET("/getFeesByOrderid", reOrderPower.GetFeesByOrderid)
		rop.GET("/getElectronicByOrderid", reOrderPower.GetElectronicByOrderid)
	}
}
