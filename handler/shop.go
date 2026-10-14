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

// shopLockUseInfoDTO 对应 Java domain.ShopLockUseInfoDto（字段 Numb/adr/state/useTime）。
type shopLockUseInfoDTO struct {
	Numb    string `json:"numb"`
	Adr     string `json:"adr"`
	State   string `json:"state"`
	UseTime string `json:"useTime"`
}

// GetShopLockInfoByShopId GET /wx/getShopLockInfoByShopId 对齐 Java getShopLockInfoByShopId：
// shop → place → 该车位所有锁 → 每个锁的使用信息 DTO。
func (c *ShopController) GetShopLockInfoByShopId(ctx *gin.Context) {
	shopID := ctx.Query("shopId")
	var shop model.ShopTbl
	if err := conf.Db.Where("shop_id = ?", shopID).First(&shop).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", shop.Address).First(&place).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	var locks []model.LockTbl
	conf.Db.Where("placeid = ?", place.Placeid).Find(&locks)

	res := make([]shopLockUseInfoDTO, 0, len(locks))
	for _, l := range locks {
		var lui model.LockUseTbl
		if conf.Db.Where("lockid = ?", l.Lockid).First(&lui).Error != nil {
			continue
		}
		if lui.UseState == "1" {
			lui.UseTime = strconv.Itoa(sysTimeDiffMs(lui.StartTime, sysTime()) / 60000)
		}
		adr := "dm"
		if l.FixType == "地下" {
			adr = "dx"
		}
		state := "使用中"
		if lui.UseState == "0" {
			state = "空闲中"
		}
		res = append(res, shopLockUseInfoDTO{Numb: lui.Lockid, Adr: adr, State: state, UseTime: lui.UseTime})
	}
	ctx.JSON(http.StatusOK, res)
}

// GetShopLockListByShopId GET /wx/getShopLockListByShopId 对齐 Java getShopLockListByShopId：
// 返回商铺关联车位下的所有车位锁（lock_tbl）。
func (c *ShopController) GetShopLockListByShopId(ctx *gin.Context) {
	shopID := ctx.Query("shopId")
	var shop model.ShopTbl
	if err := conf.Db.Where("shop_id = ?", shopID).First(&shop).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", shop.Address).First(&place).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	var locks []model.LockTbl
	conf.Db.Where("placeid = ?", place.Placeid).Find(&locks)
	ctx.JSON(http.StatusOK, locks)
}

// UpdateLockInfo GET /wx/updateLockInfo 对齐 Java ShopServerImpl.updateLockInfo。
// 注：Java 原逻辑无论原状态如何都会置 useState=1 并刷新 startTime（else 分支为死代码）。
func (c *ShopController) UpdateLockInfo(ctx *gin.Context) {
	lockID := ctx.Query("lockId")
	var lui model.LockUseTbl
	if err := conf.Db.Where("lockid = ?", lockID).First(&lui).Error; err != nil {
		ctx.JSON(http.StatusOK, false)
		return
	}
	now := sysTime()
	res := conf.Db.Model(&model.LockUseTbl{}).Where("lockid = ?", lockID).
		Updates(map[string]interface{}{"useState": "1", "startTime": now, "updateTime": now})
	ctx.JSON(http.StatusOK, res.RowsAffected > 0)
}

// GetShopStaff GET /wx/getShopStaff 对齐 Java getShopStaff：查询有权限的员工（lockPermissions=1）。
func (c *ShopController) GetShopStaff(ctx *gin.Context) {
	var staffs []model.ShopStaffTbl
	conf.Db.Where("shopId = ? AND lockPermissions = ?", ctx.Query("shopId"), "1").Find(&staffs)
	ctx.JSON(http.StatusOK, staffs)
}

// DeleteShopStaff GET /wx/deleteShopStaff 对齐 Java deleteShopStaff：取消员工权限（=0）。
func (c *ShopController) DeleteShopStaff(ctx *gin.Context) {
	res := conf.Db.Model(&model.ShopStaffTbl{}).
		Where("number = ?", ctx.Query("staffId")).
		Update("lockPermissions", "0")
	ctx.JSON(http.StatusOK, res.RowsAffected > 0)
}

// GetShopApplyStaff GET /wx/getShopApplyStaff 对齐 Java getShopApplyStaff：查询无权限员工（=0）。
func (c *ShopController) GetShopApplyStaff(ctx *gin.Context) {
	var staffs []model.ShopStaffTbl
	conf.Db.Where("shopId = ? AND lockPermissions = ?", ctx.Query("shopId"), "0").Find(&staffs)
	ctx.JSON(http.StatusOK, staffs)
}

// AgreeShopApplyStaff GET /wx/agreeShopApplyStaff 对齐 Java agreeShopApplyStaff：给予权限（=1）。
func (c *ShopController) AgreeShopApplyStaff(ctx *gin.Context) {
	res := conf.Db.Model(&model.ShopStaffTbl{}).
		Where("number = ?", ctx.Query("staffId")).
		Update("lockPermissions", "1")
	ctx.JSON(http.StatusOK, res.RowsAffected > 0)
}

// StaffSendApply GET /wx/staffSendApply 对齐 Java ShopStaffServerImpl.staffSendApply：提交权限申请。
func (c *ShopController) StaffSendApply(ctx *gin.Context) {
	openID := ctx.Query("openId")
	shopID := ctx.Query("shopId")

	var staffs []model.ShopStaffTbl
	conf.Db.Where("shopId = ?", shopID).Find(&staffs)
	for _, s := range staffs {
		if s.OpenID == openID {
			ctx.JSON(http.StatusOK, false)
			return
		}
	}

	staff := c.creatShopStaff(openID, shopID, staffs)
	res := conf.Db.Select("openId", "shopId", "dust", "name", "age", "number", "creatTime", "phone", "icon", "lockPermissions").
		Create(&staff)
	ctx.JSON(http.StatusOK, res.Error == nil)
}

// creatShopStaff 对齐 Java ShopStaffServerImpl.creatNewStaff：查询 user 封装员工信息（dust=普通员工）。
func (c *ShopController) creatShopStaff(openID, shopID string, staffs []model.ShopStaffTbl) model.ShopStaffTbl {
	var user model.UserTbl
	conf.Db.Where("openid = ?", openID).First(&user)

	lastNumber := "000001"
	for _, s := range staffs {
		lastNumber = s.Number
	}
	lastNumber = nextStaffNumber(lastNumber)

	return model.ShopStaffTbl{
		OpenID:          openID,
		ShopID:          shopID,
		Dust:            "普通员工",
		Name:            user.NickName,
		Age:             "20",
		Number:          lastNumber,
		CreatTime:       sysTime(),
		Phone:           user.Phone,
		Icon:            "", // Java 取 user.avatarUrl，Go 模型无此列（只有 avatar BLOB），置空
		LockPermissions: "0",
	}
}
