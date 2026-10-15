package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

func sysTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// sysTimeDiffMs 计算两个 "yyyy-MM-dd HH:mm:ss" 之间的毫秒差（end - start）。
func sysTimeDiffMs(start, end string) int {
	t1, err1 := time.ParseInLocation("2006-01-02 15:04:05", start, time.Local)
	t2, err2 := time.ParseInLocation("2006-01-02 15:04:05", end, time.Local)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(t2.Sub(t1).Milliseconds())
}

// ============ HotelController 自助侧（/hotel） ============

type HotelSelfController struct{}

// Add 对齐 Java HotelServiceImpl.add：更新已有酒店（要求已存在）。
func (c *HotelSelfController) Add(ctx *gin.Context) {
	var h model.HotelTbl
	if err := ctx.ShouldBindJSON(&h); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	if _, err := repository.HotelByID(h.ID); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "酒店封面未上传"))
		return
	}
	h.CreatTime = timePtr(time.Now())
	rowsAffected := repository.UpdateHotelByID(h.ID, h)
	ctx.JSON(http.StatusOK, ResultSuccess(int(rowsAffected)))
}

func (c *HotelSelfController) GetByOpenid(ctx *gin.Context) {
	rows := repository.HotelsByOpenid(ctx.GetString("openid"))
	list := make([]hotelDTO, 0, len(rows))
	for _, h := range rows {
		list = append(list, hotelToDTO(h))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *HotelSelfController) GetById(ctx *gin.Context) {
	h, err := repository.HotelByID(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(hotelToDTO(*h)))
}

// OpenLock 对齐 Java openLock：更新 hotel_product_tbl 使用状态（硬件指令省略）。
func (c *HotelSelfController) OpenLock(ctx *gin.Context) {
	lockID := ctx.Query("lockId")
	if _, err := repository.HotelProductByProductID(lockID); err == nil {
		now := sysTime()
		repository.UpdateHotelProductByProductID(lockID, map[string]interface{}{
			"useState":   "1",
			"startTime":  now,
			"updateTime": now,
		})
	}
	ctx.JSON(http.StatusOK, ResultSuccess(1))
}

// CloseLock 对齐 Java closeLock：更新使用状态并累计使用时长。
func (c *HotelSelfController) CloseLock(ctx *gin.Context) {
	lockID := ctx.Query("lockId")
	if hp, err := repository.HotelProductByProductID(lockID); err == nil {
		endTime := sysTime()
		oldUseTime := 0
		if hp.UseTime != "" {
			oldUseTime, _ = strconv.Atoi(hp.UseTime)
		}
		currentUseTimeMs := sysTimeDiffMs(hp.StartTime, endTime)
		currentUseTimeMinutes := (currentUseTimeMs + 59999) / 60000
		repository.UpdateHotelProductByProductID(lockID, map[string]interface{}{
			"useState":   "0",
			"updateTime": endTime,
			"endTime":    endTime,
			"useTime":    strconv.Itoa(oldUseTime + currentUseTimeMinutes),
		})
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// GetStaffInfo 对齐 Java getStaffInfo：按 hotelId + openId 查询单个员工。
func (c *HotelSelfController) GetStaffInfo(ctx *gin.Context) {
	s, err := repository.HotelStaffByHotelIDOpenID(ctx.Query("hotelId"), ctx.Query("openId"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(s))
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
	if err := repository.Create(&hotel).Error; err != nil {
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
	existing, err := repository.HotelByID(h.ID)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	rowsAffected := repository.UpdateHotelParkingNumber(existing.ID, existing.ParkingNumber+h.ParkingNumber)
	ctx.JSON(http.StatusOK, ResultSuccess(int(rowsAffected)))
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
	if repository.HotelProductExistsByHotelIDProductID(p.HotelID, p.ProductID) {
		ctx.JSON(http.StatusOK, ResultError(400, "该车位锁已存在"))
		return
	}
	p.UseState = "0"
	err := repository.CreateFields(&p, []string{"hotelId", "productId", "useState"})
	success := 0
	if err == nil {
		success = 1
	}
	ctx.JSON(http.StatusOK, ResultSuccess(success))
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
			repository.DecrementHotelParkingNumber(req.HotelID)
		} else {
			repository.DeleteHotelProduct(pid, req.HotelID)
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

	staffs := repository.HotelStaffsByHotelID(hotelID)
	for _, s := range staffs {
		if s.OpenID == openID {
			ctx.JSON(http.StatusOK, ResultSuccess(0))
			return
		}
	}

	staff := c.creatHotelStaff(openID, hotelID, staffs)
	if err := repository.CreateFields(&staff, []string{"openId", "hotelId", "dust", "name", "age", "number", "creatTime", "phone", "icon", "lockPermissions"}); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(-1))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(1))
}

// creatHotelStaff 对齐 Java creatNewStaff：查询 user 封装员工信息（dust=保安，权限 0）。
func (c *HotelSelfController) creatHotelStaff(openID, hotelID string, staffs []model.HotelStaffTbl) model.HotelStaffTbl {
	user := repository.FindUserByOpenid(openID)
	nickName := ""
	phone := ""
	if user != nil {
		nickName = user.NickName
		phone = user.Phone
	}

	lastNumber := "000000"
	for _, s := range staffs {
		lastNumber = s.Number
	}
	lastNumber = nextStaffNumber(lastNumber)

	return model.HotelStaffTbl{
		OpenID:          openID,
		HotelID:         hotelID,
		Dust:            "保安",
		Name:            nickName,
		Age:             "20",
		Number:          lastNumber,
		CreatTime:       sysTime(),
		Phone:           phone,
		Icon:            "", // Java 取 user.avatarUrl，Go 模型无此列（只有 avatar BLOB），置空
		LockPermissions: "0",
	}
}

// GetStaffApply GET /hotel/getStaffApply 对齐 Java getStaffApply：查询酒店开锁员申请表。
func (c *HotelSelfController) GetStaffApply(ctx *gin.Context) {
	staffs := repository.HotelStaffsByHotelID(ctx.Query("hotelId"))
	ctx.JSON(http.StatusOK, ResultSuccess(staffs))
}

// PassApply GET /hotel/PassApply 对齐 Java PassApply：通过申请（lockPermissions=1），
// 微信订阅消息推送省略。
func (c *HotelSelfController) PassApply(ctx *gin.Context) {
	rowsAffected := repository.UpdateHotelStaffLockPermissions(ctx.Query("hotelId"), ctx.Query("openId"), "1")
	ctx.JSON(http.StatusOK, ResultSuccess(int(rowsAffected)))
}

// GetPassApply GET /hotel/getPassApply 对齐 Java getPassApply：查询当前用户已通过申请的开锁权限。
func (c *HotelSelfController) GetPassApply(ctx *gin.Context) {
	openID := ctx.GetString("openid")
	staffs := repository.HotelStaffsByOpenIDLockPermissions(openID, "1")
	ctx.JSON(http.StatusOK, ResultSuccess(staffs))
}

// DeleteStaff GET /hotel/deleteStaff 对齐 Java deleteStaff：删除开锁员，0 行返回 "false"。
func (c *HotelSelfController) DeleteStaff(ctx *gin.Context) {
	if repository.DeleteHotelStaff(ctx.Query("hotelId"), ctx.Query("openId")) == 0 {
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
	hp, err := repository.HotelProductByProductIDHotelID(lockID, hotelID)
	if err != nil {
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
	rowsAffected := repository.UpdateHotelProductByProductIDHotelID(lockID, hotelID, map[string]interface{}{
		"useState":   hp.UseState,
		"useTime":    hp.UseTime,
		"updateTime": hp.UpdateTime,
		"startTime":  hp.StartTime,
		"endTime":    hp.EndTime,
	})
	return rowsAffected > 0
}

// GetLockInfo GET /hotel/getLockInfo 对齐 Java getLockInfo：查询单个锁的使用信息。
func (c *HotelSelfController) GetLockInfo(ctx *gin.Context) {
	hp, err := repository.HotelProductByProductIDHotelID(ctx.Query("lockId"), ctx.Query("hotelId"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(hp))
}

// GetAllLockInfo GET /hotel/getAllLockInfo 对齐 Java getAllLockInfo：查询酒店所有锁的使用信息。
func (c *HotelSelfController) GetAllLockInfo(ctx *gin.Context) {
	rows := repository.HotelProductsByHotelID(ctx.Query("hotelId"))
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}
