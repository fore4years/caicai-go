package main

import (
	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/middleware"
	"caicai-go/router"
	"strconv"

	"github.com/gin-gonic/gin"
)

var serverPort = ":" + strconv.Itoa(int(conf.Cfg.App.Port))

func main() {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.MiddleLog(logger.Mylog))

	router.RouterInit(r)

	r.Run(serverPort)
}
gentool -db 'mysql' -dsn 'root:lanlan1313!@#@tcp(81.69.46.160:3306)/parking?charset=utf8mb4&parseTime=True&loc=Local' -tables 'account_private_tbl,account_public_tbl,appointment_tbl,charge_message_tbl,charge_order_tbl,charge_tbl,charging_price_tbl,charging_state_tbl,charging_station_management_tbl,charging_station_tbl,community_apply_tbl,config_tbl,device_data_tbl,driveruse_tbl,external_api_app,fault_tbl,free_charging_users_tbl,hot_search_item_tbl,hotel_product_tbl,hotel_staff_tbl,hotel_tbl,imei_tab,invoice,invoice_title,led_mac_tbl,led_tbl,localchepai_tbl,lock_tbl,lock_use_tbl,message_tbl,neighbor_share_user_tbl,om_employee,openLock_tbl,order_private_tbl,order_tbl,order_twice_tbl,owner_tbl,package_record_tbl,place_tbl,placeapply_tbl,platform_tbl,point_tbl,private_charging_price_tbl,private_charging_tbl,private_place_tbl,purchase_pole_application_tbl,re_order_electronic_tbl,receptacle_power_tbl,receptacle_tbl,recharge_order_tbl,recharge_order_tbl_sorted,saveyytime_tbl,shop_staff_tbl,shop_tbl,staff_tbl,system_tbl,tab_balance_records,tab_charging_gun,tab_gateway,tab_image,tab_order_electronic,tab_parking_spaces,tab_price,tab_price_time,tab_price_time_spring,tab_price_time_summer,tab_price_time_winter,tab_product,tab_profit_sharing_config,tab_video,tab_video_watch_record,user_bank_card_tbl,user_bluetooth_bindings_tbl,user_shop_tbl,user_tbl,usercar_tbl,usermessage_tbl,version_tbl,wangguan_tbl,withdrawal_record_tbl,zbb_area' -onlyModel -outPath './model' -modelPkgName 'model'
