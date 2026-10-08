package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

// registerPartialGaps 注册对既有控制器补齐的缺失端点路由。
func registerPartialGaps(r *gin.Engine) {
	hotelSelfController := new(handler.HotelSelfController)
	shopController := new(handler.ShopController)
	imageController := new(handler.ImageController)
	productController := new(handler.ProductController)
	orderController := new(handler.OrderController)
	wodeController := new(handler.WodeController)
	ownerController := new(handler.OwnerController)

	// 酒店自助（Java HotelController）
	hotel := r.Group("/hotel")
	{
		hotel.POST("/updateHotelImg", hotelSelfController.UpdateHotelImg)
		hotel.POST("/addParkingNumber", hotelSelfController.AddParkingNumber)
		hotel.POST("/addProduct", hotelSelfController.AddProduct)
		hotel.POST("/deleteByProductId", hotelSelfController.DeleteByProductId)
		hotel.POST("/batchOpenLock", hotelSelfController.BatchOpenLock)
		hotel.GET("/staffSendApply", hotelSelfController.StaffSendApply)
		hotel.GET("/getStaffApply", hotelSelfController.GetStaffApply)
		hotel.GET("/PassApply", hotelSelfController.PassApply)
		hotel.GET("/getPassApply", hotelSelfController.GetPassApply)
		hotel.GET("/deleteStaff", hotelSelfController.DeleteStaff)
		hotel.GET("/updateLockInfo", hotelSelfController.UpdateLockInfo)
		hotel.GET("/getLockInfo", hotelSelfController.GetLockInfo)
		hotel.GET("/getAllLockInfo", hotelSelfController.GetAllLockInfo)
	}

	// 商户自助（Java ShopController，base /wx，@RequestMapping 任意方法）
	r.Any("/wx/getShopLockInfoByShopId", shopController.GetShopLockInfoByShopId)
	r.Any("/wx/getShopLockListByShopId", shopController.GetShopLockListByShopId)
	r.Any("/wx/updateLockInfo", shopController.UpdateLockInfo)
	r.Any("/wx/getShopStaff", shopController.GetShopStaff)
	r.Any("/wx/deleteShopStaff", shopController.DeleteShopStaff)
	r.Any("/wx/getShopApplyStaff", shopController.GetShopApplyStaff)
	r.Any("/wx/agreeShopApplyStaff", shopController.AgreeShopApplyStaff)
	r.Any("/wx/staffSendApply", shopController.StaffSendApply)

	// 图片（Java ImageController）
	r.POST("/image/add", imageController.Add)
	r.GET("/image/getAll", imageController.GetAll)

	// 产品（Java ProductController）
	r.GET("/product/getNotUse", productController.GetNotUse)
	r.GET("/product/getNotUseByStations", productController.GetNotUseByStations)
	r.GET("/product/getPidBySpacesId", productController.GetPidBySpacesId)

	// 用户（Java UserController，挂 OrderController）
	r.POST("/user/updateAvatar", orderController.UpdateAvatar)
	r.POST("/user/updateEmblem", orderController.UpdateEmblem)
	r.POST("/user/updateIdCardAvatar", orderController.UpdateIdCardAvatar)

	// 我的（Java wxWodeCon，@RequestMapping 任意方法）
	r.Any("/wx/wode/overOrder", wodeController.OverOrder)
	r.Any("/wx/wode/chargeOrderPay", wodeController.ChargeOrderPay)

	// 业主（Java owner/indexCon + owner/myCon）
	r.Any("/wx/owner/getByIdPlace", ownerController.GetByIdPlace)
	r.POST("/wx/owner/save", ownerController.Save)
	r.Any("/wx/owner/placeAdd_idCard_r", ownerController.PlaceAddIdCardR)
	r.Any("/wx/owner/placeAdd_prove", ownerController.PlaceAddProve)
	r.Any("/wx/owner/getmsgs", ownerController.GetMsgs)
}
