package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

func RouterInit(r *gin.Engine) {
	userController := new(handler.UserController)
	shouyeController := new(handler.ShouyeController)
	serviceNumberController := new(handler.ServiceNumberController)
	wxFujinController := new(handler.WxFujinController)
	messageController := new(handler.MessageController)
	yezhuController := new(handler.YezhuController)
	yunController := new(handler.YunController)
	wodeController := new(handler.WodeController)
	weChatController := new(handler.WeChatController)
	orderController := new(handler.OrderController)
	imageController := new(handler.ImageController)
	directiveController := new(handler.DirectiveController)
	deviceDataController := new(handler.DeviceDataController)
	externalController := new(handler.ExternalController)
	adminAuthController := new(handler.AdminAuthController)
	gatewayController := new(handler.GatewayController)
	lockController := new(handler.LockController)
	chargingGunController := new(handler.ChargingGunController)
	chargingStateController := new(handler.ChargingStateController)
	stationManagementController := new(handler.StationManagementController)
	ledController := new(handler.LedController)
	receptacleController := new(handler.ReceptacleController)
	imeiController := new(handler.ImeiController)
	productController := new(handler.ProductController)
	orderElectronicController := new(handler.OrderElectronicController)
	orderPrivateController := new(handler.OrderPrivateController)
	orderTwiceController := new(handler.OrderTwiceController)
	rechargeOrderController := new(handler.RechargeOrderController)
	withdrawalController := new(handler.WithdrawalController)
	userBankCardController := new(handler.UserBankCardController)
	userBluetoothBindingsController := new(handler.UserBluetoothBindingsController)
	accountPublicController := new(handler.AccountPublicController)
	accountPrivateController := new(handler.AccountPrivateController)
	balanceRecordController := new(handler.BalanceRecordController)
	placeController := new(handler.PlaceController)
	privatePlaceController := new(handler.PrivatePlaceController)
	freeChargingUserController := new(handler.FreeChargingUserController)
	neighborShareUserController := new(handler.NeighborShareUserController)
	configController := new(handler.ConfigController)
	versionController := new(handler.VersionController)
	openLockController := new(handler.OpenLockController)
	driverUseController := new(handler.DriverUseController)
	billingRulesController := new(handler.BillingRulesController)
	videoController := new(handler.VideoController)
	videoWatchRecordController := new(handler.VideoWatchRecordController)
	packageRecordController := new(handler.PackageRecordController)
	communityApplyController := new(handler.CommunityApplyController)
	receptaclePowerController := new(handler.ReceptaclePowerController)
	cosController := new(handler.CosController)
	userInvoiceController := new(handler.UserInvoiceController)
	purchasePoleApplicationController := new(handler.PurchasePoleApplicationController)
	parkingSpacesController := new(handler.ParkingSpacesController)
	privateChargingController := new(handler.PrivateChargingController)
	subscribeMessageController := new(handler.SubscribeMessageController)
	hotelSelfController := new(handler.HotelSelfController)
	shopController := new(handler.ShopController)
	ownerController := new(handler.OwnerController)
	adminUserController := new(handler.AdminUserController)
	adminOrderController := new(handler.AdminOrderController)
	adminPlaceController := new(handler.AdminPlaceController)
	adminParkingSpaceController := new(handler.AdminParkingSpaceController)
	adminProductController := new(handler.AdminProductController)
	adminEmployeeController := new(handler.AdminEmployeeController)
	adminRevenueController := new(handler.AdminRevenueController)
	adminLoRaController := new(handler.AdminLoRaController)
	adminLockController := new(handler.AdminLockController)
	adminFaultController := new(handler.AdminFaultController)
	adminChargeStateController := new(handler.AdminChargeStateController)
	adminChargeStationController := new(handler.AdminChargeStationController)
	adminChargeStationFourController := new(handler.AdminChargeStationFourController)
	adminInvoiceController := new(handler.AdminInvoiceController)
	adminBalanceController := new(handler.AdminBalanceController)
	adminHotelController := new(handler.AdminHotelController)
	adminProfitSharingController := new(handler.AdminProfitSharingController)
	adminPurchasePoleController := new(handler.AdminPurchasePoleController)
	adminPrivatePileController := new(handler.AdminPrivatePileController)

	// 对齐 Java WxShouyeCon 的 /wx/* 端点
	wx := r.Group("/wx")
	{
		wx.POST("/login", userController.WxLogin)
		wx.POST("/login_owner", userController.LoginOwner)
		wx.POST("/getPhoneNumber", userController.GetPhoneNumber)
		wx.POST("/getOwnerPhoneNumber", userController.GetOwnerPhoneNumber)
		wx.POST("/submit", userController.Submit)

		// 首页相关端点（Java 使用 @RequestMapping 未限定 method，此处用 Any 对齐）
		wx.Any("/getMessage", shouyeController.GetMessage)
		wx.Any("/getPlaces", shouyeController.GetPlaces)
		wx.Any("/saoma", shouyeController.Saoma)
		wx.Any("/saomaForCharge", shouyeController.SaomaForCharge)
		wx.Any("/hasOrder", shouyeController.HasOrder)
		wx.Any("/hasChargeOrder", shouyeController.HasChargeOrder)

		// 服务号 / 附近 / 消息
		wx.GET("/checkSignature", serviceNumberController.CheckSignature)
		wx.Any("/getLockid", wxFujinController.GetLockid)
		wx.Any("/getMsgStatus", messageController.GetMsgStatus)
		wx.Any("/updateMsgStatus", messageController.UpdateMsgStatus)
		wx.Any("/getmsgs", messageController.GetMsgs)
	}

	// 业主侧
	chezhu := r.Group("/chezhu")
	{
		chezhu.Any("/zhuce", yezhuController.Zhuce)
		chezhu.Any("/getLockList", yezhuController.GetLockList)
	}

	// 我的（订单）
	wode := r.Group("/wx/wode")
	{
		wode.Any("/getOrders", wodeController.GetOrders)
		wode.Any("/timeDifference", wodeController.TimeDifference)
		wode.Any("/getOrderXq", wodeController.GetOrderXq)
		wode.Any("/searchmsg", wodeController.Searchmsg)
	}

	// 微信登录/注册（对应 Java WeChatController）
	wechat := r.Group("/wechat")
	{
		wechat.GET("/getPhone", weChatController.GetPhone)
		wechat.POST("/register", weChatController.Register)
		wechat.POST("/om/register", weChatController.OmRegister)
		wechat.GET("/login", weChatController.Login)
		wechat.GET("/om/login", weChatController.OmLogin)
		wechat.POST("/yiParL/register", weChatController.YiParLRegister)
		wechat.GET("/yiParL/login", weChatController.YiParLLogin)
	}

	// 设备数据回调
	r.Any("/yun", yunController.Yun)

	// 图片（Java ImageController；/image/getById 已在 Auth 白名单内）
	r.GET("/image/getById", imageController.GetById)

	// 网关（Java GatewayController）
	gateway := r.Group("/gateway")
	{
		gateway.GET("/getById", gatewayController.GetById)
		gateway.POST("/add", gatewayController.Add)
		gateway.GET("/getAll", gatewayController.GetAll)
	}

	// 车位锁（Java LockController）
	lock := r.Group("/lock")
	{
		lock.GET("/getById", lockController.GetById)
		lock.GET("/getByLockId", lockController.GetByLockId)
		lock.POST("/add", lockController.Add)
		lock.GET("/getAll", lockController.GetAll)
		lock.GET("/getByPid", lockController.GetByPid)
		lock.GET("/getByOrder", lockController.GetByOrder)
	}

	// 充电枪（Java ChargingGunController）
	chargingGun := r.Group("/chargingGun")
	{
		chargingGun.GET("/getById", chargingGunController.GetById)
		chargingGun.POST("/add", chargingGunController.Add)
		chargingGun.POST("/addPrivateGun", chargingGunController.AddPrivateGun)
		chargingGun.GET("/getAll", chargingGunController.GetAll)
		chargingGun.GET("/getByPidAndDirection", chargingGunController.GetByPidAndDirection)
		chargingGun.POST("/updatePwm", chargingGunController.UpdatePwm)
	}

	// 充电桩状态（Java ChargingStateController）
	chargingState := r.Group("/chargingState")
	{
		chargingState.GET("/list", chargingStateController.List)
		chargingState.GET("/page", chargingStateController.Page)
		chargingState.GET("/getById", chargingStateController.GetById)
		chargingState.POST("/save", chargingStateController.Save)
		chargingState.PUT("/update", chargingStateController.Update)
		chargingState.DELETE("/deleteById", chargingStateController.DeleteById)
	}

	// 充电站人员管理（Java ChargingStationManagementController）
	stationManagement := r.Group("/chargingStations/management")
	{
		stationManagement.GET("/getAll", stationManagementController.GetAll)
		stationManagement.GET("/getById/:id", stationManagementController.GetById)
		stationManagement.GET("/getByOpenId/:openid", stationManagementController.GetByOpenId)
		stationManagement.POST("/save", stationManagementController.Save)
		stationManagement.PUT("/update/:id", stationManagementController.Update)
		stationManagement.DELETE("/delete/:id", stationManagementController.Delete)
	}

	// LED（Java LedController）
	r.GET("/led/:id", ledController.GetById)

	// 二轮车插座（Java ReceptacleController）
	receptacle := r.Group("/receptacle")
	{
		receptacle.GET("/getReceptacleAll", receptacleController.GetReceptacleAll)
		receptacle.POST("/add", receptacleController.Add)
		receptacle.POST("/batchAdd", receptacleController.BatchAdd)
		receptacle.DELETE("/delete/:id", receptacleController.DeleteById)
		receptacle.DELETE("/batchDelete/:pid", receptacleController.BatchDelete)
	}

	// IMEI 绑定（Java ImeiController）
	imei := r.Group("/imei")
	{
		imei.GET("/binding", imeiController.Binding)
		imei.GET("/cancelBinding", imeiController.CancelBinding)
		imei.GET("/listBinding", imeiController.ListBinding)
		imei.GET("/updateBinding", imeiController.UpdateBinding)
	}

	// 产品（Java ProductController，小程序侧）
	product := r.Group("/product")
	{
		product.GET("/getById", productController.GetById)
		product.GET("/getByProductId", productController.GetByProductId)
		product.POST("/addLockProduct", productController.AddLockProduct)
		product.POST("/addChargingProduct", productController.AddChargingProduct)
		product.POST("/addTwiceChargingProduct", productController.AddTwiceChargingProduct)
		product.POST("/addPrivateChargingProduct", productController.AddPrivateChargingProduct)
		product.GET("/getAll", productController.GetAll)
		product.GET("/getByImeiLike", productController.GetByImeiLike)
		product.GET("/getNidAndGatewayByPid", productController.GetNidAndGatewayByPid)
	}

	// 订单电量（Java OrderElectronicController）
	orderElectronic := r.Group("/orderElectronic")
	{
		orderElectronic.GET("/getById", orderElectronicController.GetById)
		orderElectronic.POST("/add", orderElectronicController.Add)
		orderElectronic.GET("/getAll", orderElectronicController.GetAll)
		orderElectronic.GET("/getValueByOrderid", orderElectronicController.GetValueByOrderid)
		orderElectronic.GET("/getFeesByOrderid", orderElectronicController.GetFeesByOrderid)
		orderElectronic.GET("/getPrivateFeesByOrderid", orderElectronicController.GetPrivateFeesByOrderid)
		orderElectronic.GET("/getPrivateFeesByOrderidAndPrivateUser", orderElectronicController.GetPrivateFeesByOrderidAndPrivateUser)
		orderElectronic.GET("/getPrivateFeesByPrivateOrderid", orderElectronicController.GetPrivateFeesByPrivateOrderid)
		orderElectronic.GET("/getElectronicByOrderid", orderElectronicController.GetElectronicByOrderid)
		orderElectronic.GET("/getElectronicByPrivateOrderid", orderElectronicController.GetElectronicByPrivateOrderid)
		orderElectronic.GET("/getElectronicByPrivateOrderidAndPrivateUser", orderElectronicController.GetElectronicByPrivateOrderidAndPrivateUser)
		orderElectronic.GET("/getPeriodFeesByOrderid", orderElectronicController.GetPeriodFeesByOrderid)
	}

	// 订单私桩关联（Java OrderPrivateController）
	orderPrivate := r.Group("/orderPrivate")
	{
		orderPrivate.GET("/:orderid", orderPrivateController.GetById)
		orderPrivate.POST("", orderPrivateController.Add)
		orderPrivate.GET("/getOrderState", orderPrivateController.GetOrderState)
		orderPrivate.GET("/getOrderRechargeAmountByOwner", orderPrivateController.GetOrderRechargeAmountByOwner)
		orderPrivate.GET("/getOrderRechargeAmountByNeighbor", orderPrivateController.GetOrderRechargeAmountByNeighbor)
	}

	// 二轮车订单计费（Java OrderTwiceController）
	orderTwice := r.Group("/orderTwice")
	{
		orderTwice.GET("/getByOpenid/:orderid", orderTwiceController.GetByOrderid)
		orderTwice.POST("/save", orderTwiceController.Save)
	}

	// 充值订单（Java RechargeOrderController）
	recharge := r.Group("/recharge")
	{
		recharge.GET("/list/:openid", rechargeOrderController.List)
		recharge.GET("/detail/:orderId", rechargeOrderController.Detail)
		recharge.GET("/statistics/:openid", rechargeOrderController.Statistics)
		recharge.PUT("/updateStatus", rechargeOrderController.UpdateStatus)
	}

	// 提现/退款（Java WithdrawalController）
	withdrawal := r.Group("/withdrawal")
	{
		withdrawal.POST("/apply", withdrawalController.Apply)
		withdrawal.POST("/refund", withdrawalController.Refund)
		withdrawal.POST("/refund-and-process", withdrawalController.RefundAndProcess)
		withdrawal.POST("/process-refund", withdrawalController.ProcessRefund)
		withdrawal.GET("/list/:openid", withdrawalController.List)
		withdrawal.GET("/detail/:recordId", withdrawalController.Detail)
		withdrawal.PUT("/updateStatus", withdrawalController.UpdateStatus)
	}

	// 用户银行卡（Java UserBankCardController）
	userBankCard := r.Group("/userBankCard")
	{
		userBankCard.GET("/list", userBankCardController.List)
		userBankCard.GET("/page", userBankCardController.Page)
		userBankCard.GET("/getById/:id", userBankCardController.GetById)
		userBankCard.GET("/getByOpenId/:openid", userBankCardController.GetByOpenId)
		userBankCard.POST("/save", userBankCardController.Save)
		userBankCard.PUT("/update", userBankCardController.Update)
		userBankCard.DELETE("/deleteById/:id", userBankCardController.DeleteById)
	}

	// 用户蓝牙绑定（Java UserBluetoothBindingsController）
	userBluetooth := r.Group("/userBluetoothBindings")
	{
		userBluetooth.GET("/list", userBluetoothBindingsController.List)
		userBluetooth.GET("/page", userBluetoothBindingsController.Page)
		userBluetooth.GET("/getById/:id", userBluetoothBindingsController.GetById)
		userBluetooth.GET("/getByOpenId/:openid", userBluetoothBindingsController.GetByOpenId)
		userBluetooth.POST("/save", userBluetoothBindingsController.Save)
		userBluetooth.PUT("/update", userBluetoothBindingsController.Update)
		userBluetooth.DELETE("/deleteById/:id", userBluetoothBindingsController.DeleteById)
	}

	// 公司/私人账户（Java AccountPublicController / AccountPrivateController）
	r.POST("/account/public/add", accountPublicController.Add)
	r.GET("/account/public/getByOrderId/:orderId", accountPublicController.GetByOrderId)
	r.POST("/account/private/add", accountPrivateController.Add)
	r.GET("/account/private/getByOrderId/:orderId", accountPrivateController.GetByOrderId)

	// 余额明细（Java BalanceRecordController）
	balanceRecords := r.Group("/balanceRecords")
	{
		balanceRecords.GET("/getAll", balanceRecordController.GetAll)
		balanceRecords.GET("/getByPage", balanceRecordController.GetByPage)
		balanceRecords.GET("/getById/:id", balanceRecordController.GetById)
		balanceRecords.POST("/save", balanceRecordController.Save)
		balanceRecords.PUT("/update", balanceRecordController.Update)
		balanceRecords.DELETE("/deleteById/:id", balanceRecordController.DeleteById)
	}

	// 业主车位（Java PlaceController）
	place := r.Group("/place")
	{
		place.GET("/getByOpenid/:openid", placeController.GetByOpenid)
		place.PUT("/updateById", placeController.UpdateById)
	}

	// 私桩车位（Java PrivatePlaceController）
	privatePlace := r.Group("/privatePlace")
	{
		privatePlace.GET("/getAll", privatePlaceController.GetAll)
		privatePlace.GET("/getById/:id", privatePlaceController.GetById)
		privatePlace.POST("/save", privatePlaceController.Save)
		privatePlace.PUT("/updateById/:id", privatePlaceController.UpdateById)
		privatePlace.DELETE("/deleteById/:id", privatePlaceController.DeleteById)
		privatePlace.GET("/getBySpacesCode/:spacesCode", privatePlaceController.GetBySpacesCode)
		privatePlace.GET("/getByPlaceId/:placeId", privatePlaceController.GetByPlaceId)
		privatePlace.GET("/getByOrder/:orderId", privatePlaceController.GetByOrder)
	}

	// 免费充电人员（Java FreeChargingUserController）
	freeCharging := r.Group("/freeChargingUsers")
	{
		freeCharging.GET("/getAll", freeChargingUserController.GetAll)
		freeCharging.GET("/getById/:id", freeChargingUserController.GetById)
		freeCharging.GET("/getByStationId/:stationId", freeChargingUserController.GetByStationId)
		freeCharging.GET("/getByOpenId/:openid", freeChargingUserController.GetByOpenId)
		freeCharging.POST("/save", freeChargingUserController.Save)
		freeCharging.PUT("/update/:id", freeChargingUserController.Update)
		freeCharging.PUT("/updateByOpenId/:openid/:stationId", freeChargingUserController.UpdateByOpenId)
		freeCharging.DELETE("/delete/:openid", freeChargingUserController.Delete)
	}

	// 邻享充共享用户（Java NeighborShareUserController）
	neighborShare := r.Group("/neighborShareUser")
	{
		neighborShare.POST("/save", neighborShareUserController.Save)
		neighborShare.DELETE("/delete/:id", neighborShareUserController.Delete)
		neighborShare.PUT("/updateById/:id", neighborShareUserController.Update)
		neighborShare.PUT("/refuseShareById/:id", neighborShareUserController.RefuseShareById)
		neighborShare.PUT("/cancelShareById/:id", neighborShareUserController.CancelShareById)
		neighborShare.GET("/getById/:id", neighborShareUserController.GetById)
		neighborShare.GET("/listByOwner/:ownerId", neighborShareUserController.ListByOwner)
		neighborShare.GET("/getNoSharelistByOwner/:ownerId", neighborShareUserController.GetNoSharelistByOwner)
		neighborShare.GET("/getSharelistByOwner/:ownerId", neighborShareUserController.GetSharelistByOwner)
		neighborShare.GET("/getByPidAndOpenId", neighborShareUserController.GetByPidAndOpenId)
	}

	// 参数（Java ConfigController）
	r.GET("/config/save", configController.Save)

	// 版本（Java VersionController）
	version := r.Group("/version")
	{
		version.GET("/list", versionController.List)
		version.GET("/page", versionController.Page)
		version.GET("/getById/:id", versionController.GetById)
		version.POST("/save", versionController.Save)
		version.PUT("/update", versionController.Update)
		version.DELETE("/deleteById/:id", versionController.DeleteById)
	}

	// 开箱锁记录（Java OpenLockController）
	openLock := r.Group("/openLock")
	{
		openLock.GET("/list", openLockController.List)
		openLock.GET("/page", openLockController.Page)
		openLock.GET("/getById/:id", openLockController.GetById)
		openLock.POST("/save", openLockController.Save)
		openLock.PUT("/update", openLockController.Update)
		openLock.DELETE("/deleteById/:id", openLockController.DeleteById)
	}

	// 司机使用记录（Java DriverUseController）
	r.GET("/driverUse/save", driverUseController.Save)

	// 计价规则（Java BillingRulesController，base /wx）
	r.GET("/wx/getAllBillingRules", billingRulesController.GetAllBillingRules)
	r.GET("/wx/getBillingRules", billingRulesController.GetBillingRules)

	// 视频（Java VideoController / VideoWatchRecordController）
	video := r.Group("/video")
	{
		video.POST("/save", videoController.Save)
		video.GET("/list", videoController.List)
	}
	videoWatchRecord := r.Group("/videoWatchRecord")
	{
		videoWatchRecord.POST("/save", videoWatchRecordController.Save)
		videoWatchRecord.GET("/list", videoWatchRecordController.List)
		videoWatchRecord.GET("/listByOpenid", videoWatchRecordController.ListByOpenid)
		videoWatchRecord.GET("/countByFileId", videoWatchRecordController.CountByFileId)
	}

	// 二轮车套餐（Java PackageRecordController）
	packageRecord := r.Group("/packageRecord")
	{
		packageRecord.POST("/save", packageRecordController.Save)
		packageRecord.POST("/balancePay", packageRecordController.BalancePay)
		packageRecord.POST("/refund", packageRecordController.Refund)
		packageRecord.GET("/getLatestValidPackage", packageRecordController.GetLatestValidPackage)
		packageRecord.GET("/getAllValidPackages", packageRecordController.GetAllValidPackages)
		packageRecord.GET("/getMaxPowerFromRecentOrders", packageRecordController.GetMaxPowerFromRecentOrders)
	}

	// 小区申请（Java CommunityApplyController）
	r.POST("/communityApply/add", communityApplyController.Add)

	// 二轮车插座电量（Java ReceptaclePowerController）
	receptaclePower := r.Group("/receptacle/power")
	{
		receptaclePower.GET("/getPowerByPid", receptaclePowerController.GetPowerByPid)
		receptaclePower.GET("/getPowerByReceptacleId", receptaclePowerController.GetPowerByReceptacleId)
	}

	// COS 临时密钥（Java CosController）
	r.GET("/cos/sts", cosController.Sts)

	// 车位（Java ParkingSpacesController，base /spaces）
	spaces := r.Group("/spaces")
	{
		spaces.GET("/getById", parkingSpacesController.GetById)
		spaces.GET("/getByLock", parkingSpacesController.GetByLock)
		spaces.GET("/getByOpenid", parkingSpacesController.GetByOpenid)
		spaces.POST("/add", parkingSpacesController.Add)
		spaces.GET("/getAll", parkingSpacesController.GetAll)
		spaces.GET("/getByEnable", parkingSpacesController.GetByEnable)
		spaces.GET("/setEnableONid", parkingSpacesController.SetEnableONid)
		spaces.GET("/searchAll", parkingSpacesController.SearchAll)
		spaces.GET("/getByPidAndDirection", parkingSpacesController.GetByPidAndDirection)
		spaces.GET("/getByPoint", parkingSpacesController.GetByPoint)
		spaces.GET("/getByUserOrderAndLocation", parkingSpacesController.GetByUserOrderAndLocation)
		spaces.GET("/getByDistance", parkingSpacesController.GetByDistance)
		spaces.GET("/getBySpacesCode", parkingSpacesController.GetBySpacesCode)
		spaces.GET("/search", parkingSpacesController.Search)
		spaces.GET("/isOneLockPerSpot", parkingSpacesController.IsOneLockPerSpot)
		spaces.GET("/getOtherSpacesIdBySpacesId", parkingSpacesController.GetOtherSpacesIdBySpacesId)
	}

	// 私桩（Java PrivateChargingController）
	privateCharging := r.Group("/privateCharging")
	{
		privateCharging.POST("/save", privateChargingController.Save)
		privateCharging.POST("/register", privateChargingController.Register)
		privateCharging.DELETE("/deleteById/:id", privateChargingController.DeleteById)
		privateCharging.GET("/getById/:id", privateChargingController.GetById)
		privateCharging.GET("/getByOpenId", privateChargingController.GetByOpenId)
		privateCharging.GET("/getByOpenIdAndChargeId", privateChargingController.GetByOpenIdAndChargeId)
		privateCharging.GET("/getListByOpenId", privateChargingController.GetListByOpenId)
		privateCharging.GET("/getByPidAndOpenId", privateChargingController.GetByPidAndOpenId)
		privateCharging.GET("/getUserInfoByPid", privateChargingController.GetUserInfoByPid)
		privateCharging.GET("/getAll", privateChargingController.GetAll)
		privateCharging.GET("/getBySpacesNum", privateChargingController.GetBySpacesNum)
		privateCharging.GET("/open", privateChargingController.Open)
		privateCharging.GET("/updateShareTime", privateChargingController.UpdateShareTime)
		privateCharging.GET("/close", privateChargingController.Close)
		privateCharging.GET("/getAllSharePlace", privateChargingController.GetAllSharePlace)
		privateCharging.GET("/getSharePlaceByLocation", privateChargingController.GetSharePlaceByLocation)
		privateCharging.GET("/getSharePlaceByName", privateChargingController.GetSharePlaceByName)
		privateCharging.GET("/getSharePlaceCountByName", privateChargingController.GetSharePlaceCountByName)
		privateCharging.POST("/setInfoToPrivateUser", privateChargingController.SetInfoToPrivateUser)
		privateCharging.POST("/sendNewRelocateCarMessage", privateChargingController.SendNewRelocateCarMessage)
		privateCharging.POST("/updateByProductId", privateChargingController.UpdateByProductId)
	}

	// 订阅消息（Java SubscribeMessageController）
	subscribeMessage := r.Group("/subscribeMessage")
	{
		subscribeMessage.POST("/selectRid", subscribeMessageController.SelectRid)
		subscribeMessage.POST("/sendParkingMessage", subscribeMessageController.SendParkingMessage)
		subscribeMessage.POST("/sendRelocateCarMessage", subscribeMessageController.SendRelocateCarMessage)
		subscribeMessage.POST("/sendParkingTicketMessage", subscribeMessageController.SendParkingTicketMessage)
		subscribeMessage.POST("/sendChargingMessage", subscribeMessageController.SendChargingMessage)
		subscribeMessage.POST("/sendChargingSettleMessage", subscribeMessageController.SendChargingSettleMessage)
		subscribeMessage.POST("/sendChargingExceptionMessage", subscribeMessageController.SendChargingExceptionMessage)
		subscribeMessage.POST("/sendOrderOverMessage", subscribeMessageController.SendOrderOverMessage)
		subscribeMessage.POST("/sendMoveCarMessage", subscribeMessageController.SendMoveCarMessage)
	}

	// 酒店自助（Java HotelController）
	hotelSelf := r.Group("/hotel")
	{
		hotelSelf.POST("/add", hotelSelfController.Add)
		hotelSelf.GET("/getByOpenid", hotelSelfController.GetByOpenid)
		hotelSelf.GET("/getById", hotelSelfController.GetById)
		hotelSelf.GET("/openLock", hotelSelfController.OpenLock)
		hotelSelf.GET("/closeLock", hotelSelfController.CloseLock)
		hotelSelf.GET("/getStaffInfo", hotelSelfController.GetStaffInfo)
	}

	// 商户自助（Java ShopController，base /wx）
	r.GET("/wx/getShopByOpenId", shopController.GetShopByOpenId)
	r.GET("/wx/getShopInfoByShopId", shopController.GetShopInfoByShopId)
	r.GET("/wx/getShopName", shopController.GetShopName)
	r.GET("/wx/searchShop", shopController.SearchShop)

	// 业主小程序（Java owner/controller/*）
	r.GET("/wx/owner/getPlaceAll", ownerController.GetPlaceAll)
	r.GET("/wx/owner/getPlace", ownerController.GetPlace)
	r.GET("/wx/owner/getOrders", ownerController.GetOrders)
	r.POST("/wx/owner/placeAdd", ownerController.PlaceAdd)
	r.POST("/wx/owner/placeUpdate", ownerController.PlaceUpdate)
	r.POST("/wx/owner/lockAdd", ownerController.LockAdd)
	r.GET("/wx/owner/getByLockMac", ownerController.GetByLockMac)

	// 设备控制指令（Java DirectiveController；/directive 需 JWT，对齐 @OpenRequired）
	directive := r.Group("/directive")
	{
		directive.GET("/sendHex", directiveController.SendHex)
		directive.GET("/openLock", directiveController.OpenLock)
		directive.GET("/openPlaceLock", directiveController.OpenPlaceLock)
		directive.GET("/closePlaceLock", directiveController.ClosePlaceLock)
		directive.GET("/openCharge", directiveController.OpenCharge)
		directive.GET("/getInfo", directiveController.GetInfo)
		directive.GET("/getPower", directiveController.GetPower)
		directive.GET("/getElePower", directiveController.GetElePower)
		directive.GET("/getEleCurrent", directiveController.GetEleCurrent)
		directive.GET("/getEleVoltage", directiveController.GetEleVoltage)
		directive.GET("/getEleFrequency", directiveController.GetEleFrequency)
		directive.GET("/getEleData", directiveController.GetEleData)
		directive.GET("/setGatewayAddress", directiveController.SetGatewayAddress)
		directive.GET("/setPwm", directiveController.SetPwm)
		directive.GET("/getGatewayList", directiveController.GetGatewayList)
		directive.GET("/setOpenTime", directiveController.SetOpenTime)
		directive.GET("/CalibrationPower", directiveController.CalibrationPower)
		directive.GET("/openGauge", directiveController.OpenGauge)
		directive.GET("/closeGauge", directiveController.CloseGauge)
		directive.GET("/OpenEle", directiveController.OpenEle)
		directive.GET("/closeEle", directiveController.CloseEle)
		directive.GET("/openVoice", directiveController.OpenVoice)
		directive.GET("/rotationPlatform", directiveController.RotationPlatform)
		directive.GET("/chargingEndDetection", directiveController.ChargingEndDetection)
		directive.GET("/sendBluetoothMac", directiveController.SendBluetoothMac)
		directive.GET("/BluetoothMacCancel", directiveController.BluetoothMacCancel)
		directive.GET("/setRedisValue", directiveController.SetRedisValue)
		directive.GET("/getChargeGunStatus", directiveController.GetChargeGunStatus)
		directive.GET("/changeOpenTime", directiveController.ChangeOpenTime)
		directive.GET("/closeOpenTime", directiveController.CloseOpenTime)
		directive.GET("/startOpenTime", directiveController.StartOpenTime)
		directive.GET("/getGunHolderStatus", directiveController.GetGunHolderStatus)
		directive.GET("/openGunHolderSensor", directiveController.OpenGunHolderSensor)
	}

	// 设备数据接收（Java DeviceDataController；/api/device 已在 Auth 白名单内）
	r.POST("/api/device/data", deviceDataController.Receive)

	// 外部 API（Java ExternalApiController；/external 已在 Auth 白名单内）
	r.GET("/external/openGunHolderSensor", externalController.OpenGunHolderSensor)

	// 订单 / 用户信息（鉴权由全局 Auth 中间件统一处理）
	order := r.Group("/order")
	{
		order.GET("/getNotFinish", orderController.GetNotFinish)
		order.GET("/add", orderController.Add)
		order.GET("/getByOrderId", orderController.GetByOrderId)
		order.GET("/addByLock", orderController.AddByLock)
		order.GET("/addByCharge", orderController.AddByCharge)
		order.GET("/addByTwiceCharge", orderController.AddByTwiceCharge)
		order.GET("/addByPrivateCharge", orderController.AddByPrivateCharge)
		order.GET("/closeTwiceOrder", orderController.CloseTwiceOrder)
		order.GET("/asyncCloseTwiceGauge", orderController.AsyncCloseTwiceGauge)
		order.GET("/closeTwiceOrderWithoutGauge", orderController.CloseTwiceOrderWithoutGauge)
		order.GET("/payScore", orderController.PayScore)
		order.GET("/queryPayScore", orderController.QueryPayScore)
		order.GET("/updateOrderState", orderController.UpdateOrderState)
		order.GET("/openRelay", orderController.OpenRelay)
		order.GET("/cancelOrder", orderController.CancelOrder)
		order.GET("/cancelPayScore", orderController.CancelPayScore)
		order.GET("/start", orderController.Start)
		order.GET("/stop", orderController.Stop)
		order.GET("/startLock", orderController.StartLock)
		order.GET("/stopLock", orderController.StopLock)
		order.GET("/booking", orderController.Booking)
		order.GET("/bookingByYiXiang", orderController.BookingByYiXiang)
		order.GET("/newBookingByYiXiang", orderController.NewBookingByYiXiang)
		order.GET("/getNotFinishBySpacesId", orderController.GetNotFinishBySpacesId)
		order.GET("/close", orderController.Close)
		order.GET("/finish", orderController.Finish)
		order.GET("/finishBalans", orderController.FinishBalans)
		order.GET("/refunds", orderController.Refunds)
		order.POST("/wechat/callback", orderController.WechatCallback)
		order.POST("/getOrderPay", orderController.GetOrderPay)
		order.GET("/getAllOrders", orderController.GetAllOrders)
		order.GET("/getOrdersByPage", orderController.GetOrdersByPage)
		order.GET("/getOngoingOrders", orderController.GetOngoingOrders)
		order.GET("/getDoneOrders", orderController.GetDoneOrders)
		order.GET("/getWaitPayOrders", orderController.GetWaitPayOrders)
		order.GET("/getOrdersByPageForCaiCai", orderController.GetOrdersByPageForCaiCai)
		order.GET("/getMeterValue", orderController.GetMeterValue)
		order.GET("/getCurrentValue", orderController.GetCurrentValue)
		order.GET("/getVoltageValue", orderController.GetVoltageValue)
		order.GET("/getOrderRechargeAmount", orderController.GetOrderRechargeAmount)
		order.GET("/getMeterValueByPrivate", orderController.GetMeterValueByPrivate)
		order.GET("/getOrderState", orderController.GetOrderState)
		order.GET("/getplaceUsable", orderController.GetplaceUsable)
		order.GET("/getExpectDuration", orderController.GetExpectDuration)
		order.GET("/setOrderStateIsWaitReturnGun", orderController.SetOrderStateIsWaitReturnGun)
		order.GET("/manualDetectionIsReturnGun", orderController.ManualDetectionIsReturnGun)
		order.GET("/getWaitReturnGunOrderBySpacesId", orderController.GetWaitReturnGunOrderBySpacesId)
		order.GET("/ByOrderIdSendCloseEleAndMessage", orderController.ByOrderIdSendCloseEleAndMessage)
		order.POST("/addImage", orderController.AddImage)
		order.GET("/unfreezeOrder", orderController.UnfreezeOrder)
	}
	user := r.Group("/user")
	{
		user.GET("/getUserInfo", orderController.GetUserInfo)
		user.GET("/getUserInfoByOpenId/:openid", orderController.GetUserInfoByOpenId)
		user.POST("/updateUserInfo", orderController.UpdateUserInfo)
		user.POST("/updatePhone", orderController.UpdatePhone)
		user.GET("/refToken", orderController.RefToken)
		user.GET("/updateUserIdEntity", orderController.UpdateUserIdEntity)
		user.GET("/updateUserStatus", orderController.UpdateUserStatus)
	}

	// 发票（用户侧，Java InvoiceController）
	invoiceUser := r.Group("/invoice")
	{
		invoiceUser.GET("/getByInvoiceId", userInvoiceController.GetByInvoiceId)
		invoiceUser.GET("/updateById", userInvoiceController.UpdateById)
		invoiceUser.GET("/getInvoiceByOpenid", userInvoiceController.GetInvoiceByOpenid)
		invoiceUser.POST("/add", userInvoiceController.Add)
		invoiceUser.GET("/getByOpenid", userInvoiceController.GetByOpenid)
		invoiceUser.GET("/getById", userInvoiceController.GetById)
		invoiceUser.GET("/deleteById", userInvoiceController.DeleteById)
		invoiceUser.POST("/updetaById", userInvoiceController.UpdetaById)
		invoiceUser.POST("/addInvoice", userInvoiceController.AddInvoice)
	}

	// 购桩申请（Java PurchasePoleApplicationController）
	purchasePoleApp := r.Group("/purchasePoleApplication")
	{
		purchasePoleApp.POST("/updateOrderStatus", purchasePoleApplicationController.UpdateOrderStatus)
		purchasePoleApp.GET("/getByOrderStatus/:orderStatus", purchasePoleApplicationController.GetByOrderStatus)
		purchasePoleApp.POST("/create", purchasePoleApplicationController.Create)
		purchasePoleApp.GET("/getCurrentUserApplication", purchasePoleApplicationController.GetCurrentUserApplication)
		purchasePoleApp.GET("/getByOpenid/:openid", purchasePoleApplicationController.GetByOpenid)
		purchasePoleApp.DELETE("/deleteByOrderNumber/:orderNumber", purchasePoleApplicationController.DeleteByOrderNumber)
		purchasePoleApp.GET("/pay/:orderNumber", purchasePoleApplicationController.Pay)
		purchasePoleApp.POST("/updatePaymentInfo", purchasePoleApplicationController.UpdatePaymentInfo)
		purchasePoleApp.GET("/getByOrderNumber/:orderNumber", purchasePoleApplicationController.GetByOrderNumber)
		purchasePoleApp.POST("/applyAndProcessRefund", purchasePoleApplicationController.ApplyAndProcessRefund)
	}

	// 运维职工管理（Java OmEmployeeController；/om/employee 前缀已在 Auth 白名单内）
	omEmployee := r.Group("/om/employee")
	{
		omEmployee.POST("/add", adminEmployeeController.Add)
		omEmployee.GET("/getByName", adminEmployeeController.GetByName)
		omEmployee.GET("/getAll", adminEmployeeController.GetAll)
		omEmployee.DELETE("/deleteById", adminEmployeeController.DeleteById)
		omEmployee.GET("/updateStatus", adminEmployeeController.UpdateStatus)
	}

	// 后台管理（对齐 Java System.controller.*；/system 前缀已在 Auth 白名单内）
	system := r.Group("/system")
	{
		system.POST("/login", adminAuthController.Login)
		system.POST("/pwdChange", adminAuthController.PwdChange)
		system.GET("/checkAdmin", adminAuthController.CheckAdmin)
		system.GET("/getAllUsers", adminAuthController.GetAllUsers)
		system.POST("/updateUserAdminStatus", adminAuthController.UpdateUserAdminStatus)

		// 运维用户审核（Java OmUserCon）
		systemUser := system.Group("/user")
		{
			systemUser.GET("/getOmUserByPhone", adminUserController.GetOmUserByPhone)
			systemUser.GET("/getOmUserInfoAll", adminUserController.GetOmUserInfoAll)
			systemUser.GET("/updateOmEnable", adminUserController.UpdateOmEnable)
		}

		// 订单管理（Java OrderManageCon）
		orderManage := system.Group("/orderManage")
		{
			orderManage.GET("/list", adminOrderController.GetOrderList)
			orderManage.GET("/statistics", adminOrderController.GetOrderStatistics)
			orderManage.GET("/detail", adminOrderController.GetOrderDetail)
			orderManage.GET("/export", adminOrderController.ExportOrderData)
			orderManage.DELETE("/batchDelete", adminOrderController.BatchDeleteOrders)
		}

		// 电量/金额汇总（Java OrderCon）
		systemOrder := system.Group("/order")
		{
			systemOrder.GET("/getOrderDegree", adminOrderController.GetOrderDegree)
			systemOrder.GET("/getOrderDegreeByToday", adminOrderController.GetOrderDegreeByToday)
			systemOrder.GET("/getOrderFee", adminOrderController.GetOrderFee)
			systemOrder.GET("/getOrderFeeByToday", adminOrderController.GetOrderFeeByToday)
			systemOrder.GET("/getOrdersByPidAndDate", adminOrderController.GetOrdersByPidAndDate)
		}

		// 车位管理（Java PlaceListCon）
		system.Any("/allPlace", adminPlaceController.AllPlace)
		system.GET("/searchByCity", adminPlaceController.SearchByCity)
		system.GET("/filterPlacesByState", adminPlaceController.FilterPlacesByState)
		system.Any("/getOwner", adminPlaceController.GetOwner)
		system.Any("/getUserPhone", adminPlaceController.GetUserPhone)
		system.Any("/placesDel", adminPlaceController.PlacesDel)
		system.Any("/placeGet", adminPlaceController.GetPlace)
		system.GET("/getByPlaceId", adminPlaceController.GetByPlaceId)
		system.Any("/lockGet", adminPlaceController.LockGet)
		system.Any("/chargeGet", adminPlaceController.ChargeGet)
		system.Any("/place_list/sousuo", adminPlaceController.Sousuo)
		system.POST("/updatePlaceStatus", adminPlaceController.UpdatePlaceStatus)

		// 驿享充车位（Java ParkingSpacesCon）
		parkingSpaces := system.Group("/parkingSpaces")
		{
			parkingSpaces.GET("/getAll", adminParkingSpaceController.GetAll)
			parkingSpaces.POST("/updateServiceAndFeeById", adminParkingSpaceController.UpdateServiceAndFeeById)
			parkingSpaces.GET("/getMonthlyChargingStatistics", adminParkingSpaceController.GetMonthlyChargingStatistics)
			parkingSpaces.GET("/getWeeklyChargingStatistics", adminParkingSpaceController.GetWeeklyChargingStatistics)
		}

		// 产品管理（Java ProductCon）
		product := system.Group("/product")
		{
			product.GET("/getAll", adminProductController.GetAll)
			product.GET("/search", adminProductController.Search)
			product.GET("/searchByGatewayId", adminProductController.SearchByGatewayId)
			product.GET("/getByProductId", adminProductController.GetByProductId)
			product.POST("/addLockProduct", adminProductController.AddLockProduct)
			product.POST("/addChargingProduct", adminProductController.AddChargingProduct)
			product.POST("/addTwiceChargingProduct", adminProductController.AddTwiceChargingProduct)
			product.POST("/addPrivateChargingProduct", adminProductController.AddPrivateChargingProduct)
			product.POST("/update", adminProductController.Update)
			product.DELETE("/deleteById/:pid", adminProductController.DeleteById)
		}

		// 收益统计（Java MoneyGetCon）
		system.Any("/moneyPlaces", adminRevenueController.MoneyPlaces)
		system.Any("/moneyOwner", adminRevenueController.MoneyOwner)
		system.Any("/getMoney", adminRevenueController.GetMoney)
		system.Any("/getOrders", adminRevenueController.GetOrders)
		system.Any("/getNickname", adminRevenueController.GetNickname)
		system.Any("/allAndToday", adminRevenueController.AllAndToday)
		system.Any("/orderDel", adminRevenueController.OrderDel)
		system.Any("/getTotalIncome", adminRevenueController.GetTotalIncome)
		system.Any("/getTodayIncome", adminRevenueController.GetTodayIncome)
		system.Any("/getOrderDetails", adminRevenueController.GetOrderDetails)
		system.Any("/getTodayOrders", adminRevenueController.GetTodayOrders)
		system.GET("/excel_export", adminRevenueController.ExcelExport)

		// 网关管理（Java LoRacon）
		system.Any("/getLoRaMax", adminLoRaController.GetLoRaMax)
		system.Any("/getLoRapage", adminLoRaController.GetLoRapage)
		system.Any("/getGatewayById", adminLoRaController.GetGatewayById)
		system.Any("/upbyid/:id", adminLoRaController.Upbyid)
		system.Any("/debyid/:id", adminLoRaController.Debyid)
		system.Any("/LoRaDel", adminLoRaController.LoRaDel)
		system.Any("/LoRasousuo", adminLoRaController.LoRasousuo)
		system.Any("/searchByMac", adminLoRaController.SearchByMac)
		system.Any("/raChange", adminLoRaController.RaChange)

		// 车位锁管理（Java SystemLockCon）
		system.Any("/getlockPage", adminLockController.GetlockPage)
		system.Any("/getChargePage", adminLockController.GetChargePage)
		system.Any("/getlockMax", adminLockController.GetlockMax)
		system.Any("/getPlace", adminLockController.GetPlace)
		system.Any("/disable", adminLockController.Disable)
		system.Any("/lockChange", adminLockController.LockChange)
		system.Any("/mushDis", adminLockController.MushDis)
		system.Any("/lock/sousuo", adminLockController.Sousuo)
		system.Any("/accessupdatestate", adminLockController.Accessupdatestate)

		// 故障上报（Java faultCon）
		system.Any("/getFault", adminFaultController.GetFault)
		system.Any("/faultMushDel", adminFaultController.FaultMushDel)
		system.Any("/getFaultById", adminFaultController.GetFaultById)
		system.Any("/updateState", adminFaultController.UpdateState)
		system.Any("/fault/sousuo", adminFaultController.Sousuo)

		// 充电桩实时状态（Java ChargeStateCon）
		system.GET("/chargeState/list", adminChargeStateController.List)

		// 二轮车充电站（Java ChargeStationCon）
		chargeStation := system.Group("/chargeStation")
		{
			chargeStation.GET("/getStationListByPage", adminChargeStationController.GetStationListByPage)
			chargeStation.GET("/getStationByPid", adminChargeStationController.GetStationByPid)
			chargeStation.POST("/add", adminChargeStationController.Add)
			chargeStation.POST("/updateByPid", adminChargeStationController.UpdateByPid)
			chargeStation.DELETE("/deleteById/:id", adminChargeStationController.DeleteById)
		}

		// 四轮车（Java ChargeStationFourCon）
		chargeStationFour := system.Group("/chargeStationFour")
		{
			chargeStationFour.GET("/getStationListByPage", adminChargeStationFourController.GetStationListByPage)
			chargeStationFour.GET("/getBySpacesCode", adminChargeStationFourController.GetBySpacesCode)
			chargeStationFour.GET("/getStationById", adminChargeStationFourController.GetStationById)
			chargeStationFour.POST("/updateById", adminChargeStationFourController.UpdateById)
			chargeStationFour.DELETE("/deleteById/:id", adminChargeStationFourController.DeleteById)
		}

		// 发票明细（Java InvoiceDetailCon）
		invoice := system.Group("/invoice")
		{
			invoice.GET("/list", adminInvoiceController.List)
			invoice.PUT("/process", adminInvoiceController.Process)
			invoice.POST("/batchDelete", adminInvoiceController.BatchDelete)
			invoice.GET("/checkRecentApply", adminInvoiceController.CheckRecentApply)
		}

		// 余额明细（Java BalanceDetailCon）
		balanceDetail := system.Group("/balanceDetail")
		{
			balanceDetail.GET("/list", adminBalanceController.List)
			balanceDetail.GET("/balanceSource", adminBalanceController.BalanceSource)
			balanceDetail.POST("/recharge", adminBalanceController.Recharge)
			balanceDetail.POST("/deduct", adminBalanceController.Deduct)
		}

		// 商户管理（Java HotelCon）
		hotel := system.Group("/hotel")
		{
			hotel.GET("/list", adminHotelController.List)
			hotel.PUT("/updateReview", adminHotelController.UpdateReview)
			hotel.POST("/batchDelete", adminHotelController.BatchDelete)
		}

		// 分账管理（Java ProfitSharingConfigCon）
		profitSharing := system.Group("/profitSharingConfig")
		{
			profitSharing.GET("/list", adminProfitSharingController.List)
			profitSharing.GET("/detail/:id", adminProfitSharingController.Detail)
			profitSharing.POST("/add", adminProfitSharingController.Add)
			profitSharing.PUT("/update", adminProfitSharingController.Update)
			profitSharing.DELETE("/delete/:id", adminProfitSharingController.Delete)
			profitSharing.DELETE("/batchDelete", adminProfitSharingController.BatchDelete)
		}

		// 购桩管理（Java PurchasePoleManageCon）
		purchasePole := system.Group("/purchasePoleManage")
		{
			purchasePole.GET("/list", adminPurchasePoleController.List)
			purchasePole.GET("/detail", adminPurchasePoleController.Detail)
			purchasePole.POST("/updateStatus", adminPurchasePoleController.UpdateStatus)
			purchasePole.POST("/batchDelete", adminPurchasePoleController.BatchDelete)
		}

		// 私桩审核（Java PrivatePileAuditController）
		privatePile := system.Group("/privatePile")
		{
			privatePile.GET("/getAuditList", adminPrivatePileController.GetAuditList)
			privatePile.GET("/searchByPhone", adminPrivatePileController.SearchByPhone)
			privatePile.POST("/updateStatus", adminPrivatePileController.UpdateStatus)
			privatePile.POST("/batchUpdateStatus", adminPrivatePileController.BatchUpdateStatus)
		}
	}

	// 静态资源：前端以根绝对路径引用（/js/、/page/、/img/、/agreement/ 等）。
	// 不能用 r.Static("/", ...)（会注册根 catch-all /*filepath，与 /system、/om 等 API 前缀冲突导致启动 panic），
	// 故按子目录分别挂载，根级文件用 StaticFile 精确注册。
	r.Static("/js", "./static/js")
	r.Static("/page", "./static/page")
	r.Static("/img", "./static/img")
	r.Static("/agreement", "./static/agreement")

	r.StaticFile("/", "./static/index.html")
	r.StaticFile("/index.html", "./static/index.html")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.StaticFile("/favicon1.png", "./static/favicon1.png")
	r.StaticFile("/MP_verify_rPRKIe9EPX5e9ccx.txt", "./static/MP_verify_rPRKIe9EPX5e9ccx.txt")
	r.StaticFile("/RZgTjYwAOm.txt", "./static/RZgTjYwAOm.txt")
	r.StaticFile("/QfIw9ZHwHp.txt", "./static/QfIw9ZHwHp.txt")
}
