package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// partial_gaps.go —— 补齐 Java 有、Go 缺的控制器端点。
// 所有新增方法挂到既有控制器结构体上，路由统一由 RegisterPartialGaps 注册。
// 硬件指令 / 微信订阅消息 / COS 上传等外部调用若无对等基础设施，此处省略并注释，
// 仅保留 DB 写入。
// ============================================================================

// ============================ 通用小工具 ============================

// saveNamedImage 读取 multipart 的指定字段并写入 tab_image，返回图片 id。
func saveNamedImage(ctx *gin.Context, field string) (int32, bool) {
	fh, err := ctx.FormFile(field)
	if err != nil {
		return 0, false
	}
	f, err := fh.Open()
	if err != nil {
		return 0, false
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return 0, false
	}
	img := model.TabImage{Image: data}
	if conf.Db.Create(&img).Error != nil {
		return 0, false
	}
	return img.ID, true
}

// nextStaffNumber 对齐 Java creatNewStaff 的员工编号生成：从首个非 0 字符位开始 +1，
// 前面补 0（Java 原逻辑，无员工时保留默认值）。
func nextStaffNumber(lastNumber string) string {
	for i := 0; i < len(lastNumber); i++ {
		if lastNumber[i] != '0' {
			num, err := strconv.Atoi(lastNumber)
			if err == nil {
				num++
				s := strconv.Itoa(num)
				k := ""
				for j := i; j > 0; j-- {
					k += "0"
				}
				lastNumber = k + s
			}
			break
		}
	}
	return lastNumber
}

// nextPlaceSpacesCode 对齐 Java indexSerImpl.getNextSpacesCode：code + 5 位递增编号。
func nextPlaceSpacesCode(code string) string {
	if code == "" {
		return ""
	}
	var maxCode string
	conf.Db.Raw("SELECT MAX(placeid) FROM place_tbl WHERE placeid LIKE ?", code+"%").Scan(&maxCode)
	if maxCode == "" {
		maxCode = "0"
	}
	remainder := maxCode
	if len(maxCode) >= len(code) && maxCode[:len(code)] == code {
		remainder = maxCode[len(code):]
	}
	n, err := strconv.Atoi(remainder)
	if err != nil {
		return ""
	}
	return code + fmt.Sprintf("%05d", n+1)
}

// ============================ HotelSelfController（/hotel） ============================

// UpdateHotelImg POST /hotel/updateHotelImg 对齐 Java HotelServiceImpl.updateHotelImg。
// 图片字节写入 tab_image，再以 openid + hotel_img_id 新增一条 hotel_tbl，返回新酒店 id。
func (c *HotelSelfController) UpdateHotelImg(ctx *gin.Context) {
	imageID, ok := saveImageFromCtx(ctx)
	if !ok {
		ctx.JSON(http.StatusOK, ResultError(500, "保存图片失败"))
		return
	}
	openid := ctx.GetString("openid")
	if openid == "" {
		openid = ctx.PostForm("openid")
	}
	hotel := model.HotelTbl{Openid: openid, HotelImgID: imageID}
	if err := conf.Db.Create(&hotel).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(int(hotel.ID)))
}

// AddParkingNumber POST /hotel/addParkingNumber 对齐 Java addParkingNumber：
// 在已有酒店的车位数量上累加。
func (c *HotelSelfController) AddParkingNumber(ctx *gin.Context) {
	var h model.HotelTbl
	if err := ctx.ShouldBindJSON(&h); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	var existing model.HotelTbl
	if err := conf.Db.Where("id = ?", h.ID).First(&existing).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", existing.ID).
		Update("parking_number", existing.ParkingNumber+h.ParkingNumber)
	ctx.JSON(http.StatusOK, ResultSuccess(int(res.RowsAffected)))
}

// AddProduct POST /hotel/addProduct 对齐 Java addProduct：校验 SCL 前缀、去重、新增车位锁关联。
func (c *HotelSelfController) AddProduct(ctx *gin.Context) {
	var p model.HotelProductTbl
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	if len(p.ProductID) < 3 || p.ProductID[:3] != "SCL" {
		ctx.JSON(http.StatusOK, ResultError(400, "二维码错误"))
		return
	}
	var existing model.HotelProductTbl
	if err := conf.Db.Where("hotelId = ? AND productId = ?", p.HotelID, p.ProductID).First(&existing).Error; err == nil {
		ctx.JSON(http.StatusOK, ResultError(400, "该车位锁已存在"))
		return
	}
	p.UseState = "0"
	res := conf.Db.Select("hotelId", "productId", "useState").Create(&p)
	ctx.JSON(http.StatusOK, ResultSuccess(int(res.RowsAffected)))
}

// DeleteByProductId POST /hotel/deleteByProductId 对齐 Java deleteByProductIds：
// productIds 中为空的项使 hotel_tbl.parking_number - 1，否则删除对应 hotel_product_tbl 记录。
func (c *HotelSelfController) DeleteByProductId(ctx *gin.Context) {
	var req struct {
		ProductIDs []string `json:"productIds"`
		HotelID    string   `json:"hotelId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess("ok"))
		return
	}
	for _, pid := range req.ProductIDs {
		if pid == "" {
			conf.Db.Exec("UPDATE hotel_tbl SET parking_number = parking_number - 1 WHERE id = ?", req.HotelID)
		} else {
			conf.Db.Where("productId = ? AND hotelId = ?", pid, req.HotelID).Delete(&model.HotelProductTbl{})
		}
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// BatchOpenLock POST /hotel/batchOpenLock 对齐 Java batchOpenLock：
// 逐锁更新使用信息（硬件开锁指令省略），完成后 sleep 2s。
func (c *HotelSelfController) BatchOpenLock(ctx *gin.Context) {
	var req struct {
		LockIDs []string `json:"lockIds"`
		HotelID string   `json:"hotelId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess("开锁成功"))
		return
	}
	for _, lockID := range req.LockIDs {
		// Java 此处调用 hotelService.openLock(lockId) 下发硬件开锁指令，省略。
		c.updateLockInfoInternal(lockID, req.HotelID)
	}
	time.Sleep(2 * time.Second)
	ctx.JSON(http.StatusOK, ResultSuccess("开锁成功"))
}

// StaffSendApply GET /hotel/staffSendApply 对齐 Java staffSendApply：0 已存在 / 1 成功 / -1 失败。
func (c *HotelSelfController) StaffSendApply(ctx *gin.Context) {
	openID := ctx.Query("openId")
	hotelID := ctx.Query("hotelId")

	var staffs []model.HotelStaffTbl
	conf.Db.Where("hotelId = ?", hotelID).Find(&staffs)
	for _, s := range staffs {
		if s.OpenID == openID {
			ctx.JSON(http.StatusOK, ResultSuccess(0))
			return
		}
	}

	staff := c.creatHotelStaff(openID, hotelID, staffs)
	res := conf.Db.Select("openId", "hotelId", "dust", "name", "age", "number", "creatTime", "phone", "icon", "lockPermissions").
		Create(&staff)
	if res.Error != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(-1))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(1))
}

// creatHotelStaff 对齐 Java creatNewStaff：查询 user 封装员工信息（dust=保安，权限 0）。
func (c *HotelSelfController) creatHotelStaff(openID, hotelID string, staffs []model.HotelStaffTbl) model.HotelStaffTbl {
	var user model.UserTbl
	conf.Db.Where("openid = ?", openID).First(&user)

	lastNumber := "000000"
	for _, s := range staffs {
		lastNumber = s.Number
	}
	lastNumber = nextStaffNumber(lastNumber)

	return model.HotelStaffTbl{
		OpenID:          openID,
		HotelID:         hotelID,
		Dust:            "保安",
		Name:            user.NickName,
		Age:             "20",
		Number:          lastNumber,
		CreatTime:       sysTime(),
		Phone:           user.Phone,
		Icon:            "", // Java 取 user.avatarUrl，Go 模型无此列（只有 avatar BLOB），置空
		LockPermissions: "0",
	}
}

// GetStaffApply GET /hotel/getStaffApply 对齐 Java getStaffApply：查询酒店开锁员申请表。
func (c *HotelSelfController) GetStaffApply(ctx *gin.Context) {
	var staffs []model.HotelStaffTbl
	conf.Db.Where("hotelId = ?", ctx.Query("hotelId")).Find(&staffs)
	ctx.JSON(http.StatusOK, ResultSuccess(staffs))
}

// PassApply GET /hotel/PassApply 对齐 Java PassApply：通过申请（lockPermissions=1），
// 微信订阅消息推送省略。
func (c *HotelSelfController) PassApply(ctx *gin.Context) {
	res := conf.Db.Model(&model.HotelStaffTbl{}).
		Where("hotelId = ? AND openId = ?", ctx.Query("hotelId"), ctx.Query("openId")).
		Update("lockPermissions", "1")
	ctx.JSON(http.StatusOK, ResultSuccess(int(res.RowsAffected)))
}

// GetPassApply GET /hotel/getPassApply 对齐 Java getPassApply：查询当前用户已通过申请的开锁权限。
func (c *HotelSelfController) GetPassApply(ctx *gin.Context) {
	openID := ctx.GetString("openid")
	var staffs []model.HotelStaffTbl
	conf.Db.Where("openId = ? AND lockPermissions = ?", openID, "1").Find(&staffs)
	ctx.JSON(http.StatusOK, ResultSuccess(staffs))
}

// DeleteStaff GET /hotel/deleteStaff 对齐 Java deleteStaff：删除开锁员，0 行返回 "false"。
func (c *HotelSelfController) DeleteStaff(ctx *gin.Context) {
	res := conf.Db.Where("hotelId = ? AND openId = ?", ctx.Query("hotelId"), ctx.Query("openId")).
		Delete(&model.HotelStaffTbl{})
	if res.RowsAffected == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess("false"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// UpdateLockInfo GET /hotel/updateLockInfo 对齐 Java updateLockInfo：切换锁使用状态。
func (c *HotelSelfController) UpdateLockInfo(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.updateLockInfoInternal(ctx.Query("lockId"), ctx.Query("hotelId"))))
}

// updateLockInfoInternal 对齐 Java HotelServiceImpl.updateLockInfo：
// 0→1 记开锁时间，1→0 记关锁时间并覆盖使用时长（分钟）。
func (c *HotelSelfController) updateLockInfoInternal(lockID, hotelID string) bool {
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ? AND hotelId = ?", lockID, hotelID).First(&hp).Error; err != nil {
		return false
	}
	now := sysTime()
	if hp.UseState == "0" {
		hp.UseState = "1"
		hp.StartTime = now
	} else {
		hp.UseState = "0"
		hp.EndTime = now
		hp.UseTime = strconv.Itoa(sysTimeDiffMs(hp.StartTime, now) / 60000)
	}
	hp.UpdateTime = now
	res := conf.Db.Model(&model.HotelProductTbl{}).Where("productId = ? AND hotelId = ?", lockID, hotelID).
		Updates(map[string]interface{}{
			"useState":   hp.UseState,
			"useTime":    hp.UseTime,
			"updateTime": hp.UpdateTime,
			"startTime":  hp.StartTime,
			"endTime":    hp.EndTime,
		})
	return res.RowsAffected > 0
}

// GetLockInfo GET /hotel/getLockInfo 对齐 Java getLockInfo：查询单个锁的使用信息。
func (c *HotelSelfController) GetLockInfo(ctx *gin.Context) {
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ? AND hotelId = ?", ctx.Query("lockId"), ctx.Query("hotelId")).First(&hp).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(hp))
}

// GetAllLockInfo GET /hotel/getAllLockInfo 对齐 Java getAllLockInfo：查询酒店所有锁的使用信息。
func (c *HotelSelfController) GetAllLockInfo(ctx *gin.Context) {
	var rows []model.HotelProductTbl
	conf.Db.Where("hotelId = ?", ctx.Query("hotelId")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// ============================ ShopController（/wx 商户） ============================

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

// ============================ ImageController（/image） ============================

// Add POST /image/add 对齐 Java ImageController.add：JSON 提交图片字节落库。
func (i *ImageController) Add(c *gin.Context) {
	var req model.TabImage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Create(&req)
	c.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

// GetAll GET /image/getAll 对齐 Java ImageController.getAll：分页查询所有图片。
func (i *ImageController) GetAll(c *gin.Context) {
	current, size := parsePage(c)
	var total int64
	conf.Db.Model(&model.TabImage{}).Count(&total)
	var rows []model.TabImage
	conf.Db.Model(&model.TabImage{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// ============================ ProductController（/product） ============================

// GetNotUse GET /product/getNotUse 对齐 Java getNotUse：未被充电枪/车位锁使用且非二轮车的产品。
func (p *ProductController) GetNotUse(c *gin.Context) {
	var rows []model.TabProduct
	conf.Db.Raw(`SELECT tab_product.* FROM tab_product
		LEFT JOIN tab_charging_gun ON tab_charging_gun.product_id = tab_product.pid
		LEFT JOIN lock_tbl ON lock_tbl.product_id = tab_product.pid
		WHERE tab_charging_gun.product_id IS NULL AND lock_tbl.product_id IS NULL AND tab_product.is_twicecar = 0`).Scan(&rows)
	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// GetNotUseByStations GET /product/getNotUseByStations 对齐 Java getNotUseByStations：二轮车充电站产品。
func (p *ProductController) GetNotUseByStations(c *gin.Context) {
	var rows []model.TabProduct
	conf.Db.Raw(`SELECT tab_product.* FROM tab_product WHERE tab_product.is_twicecar = 1`).Scan(&rows)
	list := make([]productDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, productToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// GetPidBySpacesId GET /product/getPidBySpacesId 对齐 Java getPidBySpacesId：
// 车位 → 充电枪 → 产品。
func (p *ProductController) GetPidBySpacesId(c *gin.Context) {
	var prod model.TabProduct
	err := conf.Db.Raw(`SELECT tab_product.* FROM tab_parking_spaces
		LEFT JOIN tab_charging_gun ON tab_parking_spaces.charging_gun_id = tab_charging_gun.id
		LEFT JOIN tab_product ON tab_charging_gun.product_id = tab_product.pid
		WHERE tab_parking_spaces.id = ?`, c.Query("spacesId")).Scan(&prod).Error
	if err != nil || prod.Pid == "" {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(productToDTO(prod)))
}

// ============================ UserController（挂在 OrderController 上，/user） ============================

// UpdateAvatar POST /user/updateAvatar 对齐 Java updateAvatar：头像字节直接写入 user_tbl.avatar。
func (o *OrderController) UpdateAvatar(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	openid := c.GetString("openid")
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("avatar", data)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// UpdateEmblem POST /user/updateEmblem 对齐 Java updateEmblem：国徽面图片写入 tab_image，回写 id_card_emblem。
func (o *OrderController) UpdateEmblem(c *gin.Context) {
	imageID, ok := saveImageFromCtx(c)
	if !ok {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	openid := c.GetString("openid")
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("id_card_emblem", imageID)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// UpdateIdCardAvatar POST /user/updateIdCardAvatar 对齐 Java updateIdCardAvatar：头像面图片写入 tab_image，回写 id_card_avatar。
func (o *OrderController) UpdateIdCardAvatar(c *gin.Context) {
	imageID, ok := saveImageFromCtx(c)
	if !ok {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	openid := c.GetString("openid")
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("id_card_avatar", imageID)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// ============================ WodeController（/wx/wode） ============================

// OverOrder POST /wx/wode/overOrder 对齐 Java wxWodeSerImpl.overOrder：
// 结束停车订单，按停车费率（30 分钟免费、按 30 分钟档计费 + 超时计费）计算金额，置为待支付。
func (w *WodeController) OverOrder(c *gin.Context) {
	var req struct {
		Orderid string `json:"orderid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, "ok")
		return
	}
	var order model.OrderTbl
	if err := conf.Db.Where("orderid = ?", req.Orderid).First(&order).Error; err != nil {
		c.JSON(http.StatusOK, "ok")
		return
	}
	place := placeByLockid(order.Lockid)
	if place == nil {
		c.JSON(http.StatusOK, "ok")
		return
	}

	now := time.Now()
	var totalPrice decimal.Decimal
	if !order.BeginTime.IsZero() {
		useTime := now.Sub(order.BeginTime)
		minutes := int(useTime.Minutes())
		if minutes <= 30 {
			totalPrice = decimal.Zero
		} else {
			extraMinutes := minutes - 30
			intervals := extraMinutes / 30
			if extraMinutes%30 > 0 {
				intervals++
			}
			rate, _ := decimal.NewFromString(place.Rate)
			totalPrice = rate.Mul(decimal.NewFromInt(int64(intervals)))
		}

		openTime := place.OpenTime
		if openTime != "" && openTime != "全天" {
			parts := strings.Split(openTime, "-")
			if len(parts) >= 2 {
				closeTimeStr := order.BeginTime.Format("2006-01-02") + " " + parts[1] + ":00"
				if closeT, err := time.ParseInLocation("2006-01-02 15:04:05", closeTimeStr, time.Local); err == nil {
					okTime := closeT.Sub(order.BeginTime)
					if useTime > okTime {
						availableMinutes := int(okTime.Minutes())
						useMinutes := minutes
						rate, _ := decimal.NewFromString(place.Rate)
						if availableMinutes >= 30 {
							availableMinutes -= 30
							useMinutes -= 30
							totalPrice = decimal.NewFromFloat(float64(useMinutes-availableMinutes) * 0.5).Add(rate)
						} else {
							useMinutes -= availableMinutes
							totalPrice = decimal.NewFromFloat(float64(useMinutes) * 0.5).Add(rate)
						}
					}
				}
				if totalPrice.GreaterThan(decimal.NewFromInt(200)) {
					totalPrice = decimal.NewFromInt(200)
				}
			}
		}
	}

	// Java orderService.stopLock(orderid) 下发关锁指令，省略。
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", place.Placeid).Update("state", "可使用")

	chargingTime := ""
	if !order.BeginTime.IsZero() {
		d := now.Sub(order.BeginTime)
		chargingTime = time.Time{}.Add(d).Format("15:04:05")
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", req.Orderid).Updates(map[string]interface{}{
		"over_time":         now,
		"close_time":        now,
		"total_price":       totalPrice,
		"basic_consumption": totalPrice,
		"state":             orderStateWaitPay,
		"charging_time":     chargingTime,
		"stop_mode":         "扫码自停",
	})
	// 微信订阅消息推送省略（Java subscribeMessageService.ParkingMessage）。
	c.JSON(http.StatusOK, "ok")
}

// ChargeOrderPay POST /wx/wode/chargeOrderPay 对齐 Java wxWodeSerImpl.chargeOrderPay：
// charge_order_tbl 置为已完成（Java CaiCaiPayUtil.isPaySuccess() 恒为 true，外呼省略）。
func (w *WodeController) ChargeOrderPay(c *gin.Context) {
	var req struct {
		Orderid string `json:"orderid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, "fail")
		return
	}
	var o model.ChargeOrderTbl
	if err := conf.Db.Where("orderid = ?", req.Orderid).First(&o).Error; err != nil {
		c.JSON(http.StatusOK, "fail")
		return
	}
	// CaiCaiPayUtil.isPaySuccess() 恒为 true；Java updataByOrderid 会回写其它同值字段（等价于只改 state）。
	conf.Db.Model(&model.ChargeOrderTbl{}).Where("orderid = ?", o.Orderid).Update("state", "已完成")
	c.JSON(http.StatusOK, "ok")
}

// ============================ OwnerController（/wx/owner） ============================

// GetByIdPlace GET /wx/owner/getByIdPlace 对齐 Java indexSerImpl.getByIdPlace：按 placeid 查单个车位（返回列表）。
func (c *OwnerController) GetByIdPlace(ctx *gin.Context) {
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", ctx.Query("placeid")).First(&place).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess([]PlaceDtoDTO{}))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess([]PlaceDtoDTO{placeToPlaceDtoDTO(place)}))
}

// Save POST /wx/owner/save 对齐 Java indexCon.add + indexSerImpl.save：新增车位（证书图片可选，COS 上传省略）。
func (c *OwnerController) Save(ctx *gin.Context) {
	code := ctx.PostForm("code")
	openid := ctx.PostForm("openid")
	place := ctx.PostForm("place") // relatedBuilding
	province := ctx.PostForm("province")
	city := ctx.PostForm("city")
	district := ctx.PostForm("district")
	street := ctx.PostForm("street") // streer
	latitude := ctx.PostForm("latitude")
	longitude := ctx.PostForm("longitude")

	var imageID int32
	// Java：certificate == null ? null : imageService.add(certificate)；COS/磁盘上传省略，图片字节落 tab_image。
	if _, err := ctx.FormFile("file"); err == nil {
		imageID, _ = saveImageFromCtx(ctx)
	}

	spacesCode := nextPlaceSpacesCode(code)
	if spacesCode == "" {
		ctx.JSON(http.StatusOK, false)
		return
	}
	rec := model.PlaceTbl{
		Placeid:         spacesCode,
		Ownerid:         openid,
		Province:        province,
		City:            city,
		District:        district,
		Streer:          street,
		Longitude:       longitude,
		Latitude:        latitude,
		RelatedBuilding: place,
		State:           "待审核",
		ImageID:         imageID,
	}
	res := conf.Db.Select("placeid", "ownerid", "province", "city", "district", "streer", "longitude", "latitude", "relatedBuilding", "state", "image_id").
		Create(&rec)
	ctx.JSON(http.StatusOK, res.Error == nil)
}

// PlaceAddIdCardR POST /wx/owner/placeAdd_idCard_r 对齐 Java mySerImpl.placeAdd_r：回写身份证反面照。
// Java 用 saveFileUtil.save 落盘/COS，此处省略，图片字节存入 tab_image，以图片 id 字符串回写。
func (c *OwnerController) PlaceAddIdCardR(ctx *gin.Context) {
	id := ctx.PostForm("id")
	url := ""
	if _, err := ctx.FormFile("idCard_r"); err == nil {
		if imageID, ok := saveNamedImage(ctx, "idCard_r"); ok {
			url = strconv.Itoa(int(imageID))
		}
	}
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", id).Update("idCard_r", url)
	ctx.JSON(http.StatusOK, "ok")
}

// PlaceAddProve POST /wx/owner/placeAdd_prove 对齐 Java mySerImpl.placeAdd_p：回写产权证明。
// 同 placeAdd_idCard_r，文件存储省略，图片 id 字符串回写。
func (c *OwnerController) PlaceAddProve(ctx *gin.Context) {
	id := ctx.PostForm("id")
	url := ""
	if _, err := ctx.FormFile("prove"); err == nil {
		if imageID, ok := saveNamedImage(ctx, "prove"); ok {
			url = strconv.Itoa(int(imageID))
		}
	}
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", id).Update("prove", url)
	ctx.JSON(http.StatusOK, "ok")
}

// ownerMessageDTO 对应 Java domain.MessageBean。
type ownerMessageDTO struct {
	ID      int32  `json:"id,omitempty"`
	Ownerid string `json:"ownerid,omitempty"`
	Message string `json:"message,omitempty"`
	Msgtime string `json:"msgtime,omitempty"`
	Status  string `json:"status,omitempty"`
}

// GetMsgs GET /wx/owner/getmsgs 对齐 Java mySerImpl.getmsgs：查询业主消息并倒序。
func (c *OwnerController) GetMsgs(ctx *gin.Context) {
	var msgs []model.MessageTbl
	conf.Db.Where("ownerid = ?", ctx.Query("ownerid")).Find(&msgs)
	// Java Collections.reverse 倒序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	list := make([]ownerMessageDTO, 0, len(msgs))
	for _, m := range msgs {
		list = append(list, ownerMessageDTO{ID: m.ID, Ownerid: m.Ownerid, Message: m.Message, Msgtime: m.Msgtime, Status: m.Status})
	}
	ctx.JSON(http.StatusOK, list)
}
