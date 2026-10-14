package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ ShopController（/wx 商户） ============

// shopDTO 对应 Java domain.Shop（驼峰）。
type shopDTO struct {
	ID         int32  `json:"id,omitempty"`
	Icon       string `json:"icon,omitempty"`
	ShopID     string `json:"shopId,omitempty"`
	Name       string `json:"name,omitempty"`
	Address    string `json:"address,omitempty"`
	WorkState  string `json:"workState,omitempty"`
	WorkTime   string `json:"workTime,omitempty"`
	CreatTime  string `json:"creatTime,omitempty"`
	UpdateTime string `json:"updateTime,omitempty"`
}

func shopToDTO(s model.ShopTbl) shopDTO {
	return shopDTO{
		ID: s.ID, Icon: s.Icon, ShopID: s.ShopID, Name: s.Name, Address: s.Address,
		WorkState: s.WorkState, WorkTime: s.WorkTime, CreatTime: s.CreatTime, UpdateTime: s.UpdateTime,
	}
}

// shopInfoDTO 对应 Java dto.ShopDto。
type shopInfoDTO struct {
	ShopID    string `json:"shopId,omitempty"`
	URL       string `json:"url,omitempty"`
	Name      string `json:"name,omitempty"`
	State     string `json:"state,omitempty"`
	Address   string `json:"address,omitempty"`
	WorkTime  string `json:"workTime,omitempty"`
	CheweiNum string `json:"cheweiNum,omitempty"`
}

// shopToShopInfo 对应 Java shopToShopDto。
func shopToShopInfo(shop model.ShopTbl) shopInfoDTO {
	state := "休息中"
	if shop.WorkState == "1" {
		state = "营业中"
	}
	address := ""
	cheweiNum := "0"
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", shop.Address).First(&place).Error; err == nil {
		address = place.Province + place.City + place.District + place.Streer + shop.Name
		var locks []model.LockTbl
		conf.Db.Where("placeid = ?", place.Placeid).Find(&locks)
		num := 0
		for _, l := range locks {
			var lui model.LockUseTbl
			if conf.Db.Where("lockid = ?", l.Lockid).First(&lui).Error == nil && lui.UseState == "0" {
				num++
			}
		}
		cheweiNum = strconv.Itoa(num)
	}
	return shopInfoDTO{
		ShopID: shop.ShopID, URL: shop.Icon, Name: shop.Name, State: state,
		Address: address, WorkTime: shop.WorkTime, CheweiNum: cheweiNum,
	}
}

type ShopController struct{}

// GetShopByOpenId 对齐 Java getShopByOpenId（返回 List<ShopDto>，非 Result 包装）。
func (c *ShopController) GetShopByOpenId(ctx *gin.Context) {
	openID := ctx.Query("openId")
	var userShops []model.UserShopTbl
	conf.Db.Where("open_id = ?", openID).Find(&userShops)
	list := make([]shopInfoDTO, 0, len(userShops))
	for _, us := range userShops {
		var shop model.ShopTbl
		if err := conf.Db.Where("shop_id = ?", us.ShopID).First(&shop).Error; err == nil {
			list = append(list, shopToShopInfo(shop))
		}
	}
	ctx.JSON(http.StatusOK, list)
}

// GetShopInfoByShopId 对齐 Java getShopInfoByShopId（返回 ShopDto）。
func (c *ShopController) GetShopInfoByShopId(ctx *gin.Context) {
	var shop model.ShopTbl
	if err := conf.Db.Where("shop_id = ?", ctx.Query("shopId")).First(&shop).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	ctx.JSON(http.StatusOK, shopToShopInfo(shop))
}

// GetShopName 对齐 Java getShopName（返回 String）。
func (c *ShopController) GetShopName(ctx *gin.Context) {
	var shop model.ShopTbl
	if err := conf.Db.Where("shop_id = ?", ctx.Query("shopId")).First(&shop).Error; err != nil {
		ctx.JSON(http.StatusOK, "")
		return
	}
	ctx.JSON(http.StatusOK, shop.Name)
}

// SearchShop 对齐 Java searchShop（按名称模糊查询，返回 List<Shop>）。
func (c *ShopController) SearchShop(ctx *gin.Context) {
	var rows []model.ShopTbl
	conf.Db.Where("name LIKE ?", "%"+ctx.Query("str")+"%").Find(&rows)
	list := make([]shopDTO, 0, len(rows))
	for _, s := range rows {
		list = append(list, shopToDTO(s))
	}
	ctx.JSON(http.StatusOK, list)
}
