package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

// registerPricing 注册充电价格 / 价格体系路由。
func registerPricing(r *gin.Engine) {
	chargingPrice := new(handler.ChargingPriceController)
	chargingPriceTime := new(handler.ChargingPriceTimeController)
	price := new(handler.PriceController)
	priceTime := new(handler.PriceTimeController)

	cp := r.Group("/ChargingPrice")
	{
		cp.GET("/getById", chargingPrice.GetById)
		cp.POST("/add", chargingPrice.Add)
		cp.GET("/getAll", chargingPrice.GetAll)
		cp.GET("/getMix", chargingPrice.GetMix)
		cp.GET("/getByTime", chargingPrice.GetByTime)
	}

	cpt := r.Group("/ChargingPriceTime")
	{
		cpt.GET("/getById", chargingPriceTime.GetById)
		cpt.POST("/add", chargingPriceTime.Add)
		cpt.GET("/getAll", chargingPriceTime.GetAll)
		cpt.GET("/getByTime", chargingPriceTime.GetByTime)
		cpt.GET("/getNow", chargingPriceTime.GetNow)
	}

	p := r.Group("/price")
	{
		p.GET("/getById", price.GetById)
		p.POST("/add", price.Add)
		p.GET("/getAll", price.GetAll)
		p.GET("/getMix", price.GetMix)
		p.GET("/getByTime", price.GetByTime)
	}

	pt := r.Group("/priceTime")
	{
		pt.GET("/getById", priceTime.GetById)
		pt.POST("/add", priceTime.Add)
		pt.GET("/getAll", priceTime.GetAll)
		pt.GET("/getByTime", priceTime.GetByTime)
		pt.GET("/getNow", priceTime.GetNow)
		pt.GET("/getAllNow", priceTime.GetAllNow)
	}
}

// registerZbbArea 注册区域表（ZbbArea）路由。
func registerZbbArea(r *gin.Engine) {
	zbbArea := new(handler.ZbbAreaController)
	g := r.Group("/ZbbAreaBean")
	{
		g.GET("/getById", zbbArea.GetById)
		g.POST("/add", zbbArea.Add)
		g.GET("/getAll", zbbArea.GetAll)
	}
}
