package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// gatewayDTO 对应 Java model.GatewayBean（tab_gateway，驼峰）。
type gatewayDTO struct {
	ID           int32          `json:"id"`
	Mac          string         `json:"mac,omitempty"`
	Iccid        string         `json:"iccid,omitempty"`
	Name         string         `json:"name,omitempty"`
	InstallPlace string         `json:"installPlace,omitempty"`
	InstallTime  *LocalDateTime `json:"installTime,omitempty"`
}

func gatewayToDTO(g model.TabGateway) gatewayDTO {
	return gatewayDTO{
		ID:           g.ID,
		Mac:          g.Mac,
		Iccid:        g.Iccid,
		Name:         g.Name,
		InstallPlace: g.InstallPlace,
		InstallTime:  timeToLocal(g.InstallTime),
	}
}

// faultDTO 对应 Java domain.fault（fault_tbl，驼峰）。
type faultDTO struct {
	Faultid         string `json:"faultid"`
	Openid          string `json:"openid,omitempty"`
	Lockmac         string `json:"lockmac,omitempty"`
	Error           string `json:"error,omitempty"`
	Image           string `json:"image,omitempty"`
	PutTime         string `json:"putTime,omitempty"`
	State           string `json:"state,omitempty"`
	MaintenanceTime string `json:"maintenanceTime,omitempty"`
	Name            string `json:"name,omitempty"`
}

func faultToDTO(f model.FaultTbl) faultDTO {
	return faultDTO{
		Faultid:         f.Faultid,
		Openid:          f.Openid,
		Lockmac:         f.Lockmac,
		Error:           f.Error,
		Image:           f.Image,
		PutTime:         timeToString(f.PutTime),
		State:           f.State,
		MaintenanceTime: timeToString(f.MaintenanceTime),
		Name:            f.Name,
	}
}

// AdminLoRaController 对齐 Java System.controller.LoRacon（网关管理，legacy raw 返回）。
type AdminLoRaController struct{}

// GetLoRaMax /system/getLoRaMax → 网关总数
func (a *AdminLoRaController) GetLoRaMax(c *gin.Context) {
	var n int64
	conf.Db.Model(&model.TabGateway{}).Count(&n)
	c.JSON(http.StatusOK, n)
}

// GetLoRapage /system/getLoRapage?page=&length= → List<GatewayBean>
func (a *AdminLoRaController) GetLoRapage(c *gin.Context) {
	current, size := parsePage(c)
	var gateways []model.TabGateway
	conf.Db.Model(&model.TabGateway{}).Order("install_time desc").
		Offset(pageOffset(current, size)).Limit(size).Find(&gateways)
	list := make([]gatewayDTO, 0, len(gateways))
	for _, g := range gateways {
		list = append(list, gatewayToDTO(g))
	}
	c.JSON(http.StatusOK, list)
}

// GetGatewayById /system/getGatewayById?id=
func (a *AdminLoRaController) GetGatewayById(c *gin.Context) {
	var g model.TabGateway
	if err := conf.Db.Where("id = ?", c.Query("id")).First(&g).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, gatewayToDTO(g))
}

// Upbyid /system/upbyid/:id → 更新 wangguan_tbl（body 为 LoRas）
func (a *AdminLoRaController) Upbyid(c *gin.Context) {
	var req struct {
		ID           int32  `json:"id"`
		Name         string `json:"name"`
		Sn           string `json:"sn"`
		Wgtype       string `json:"wgtype"`
		Organization string `json:"organization"`
		Azadd        string `json:"azadd"`
		Cjshijian    string `json:"cjshijian"`
		Xgshijian    string `json:"xgshijian"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, 0)
		return
	}
	conf.Db.Model(&model.WangguanTbl{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
		"name":         req.Name,
		"sn":           req.Sn,
		"wgtype":       req.Wgtype,
		"organization": req.Organization,
		"azadd":        req.Azadd,
		"xgshijian":    time.Now(),
	})
	c.JSON(http.StatusOK, req.ID)
}

// Debyid /system/debyid/:id → 删除 wangguan_tbl
func (a *AdminLoRaController) Debyid(c *gin.Context) {
	conf.Db.Where("id = ?", c.Param("id")).Delete(&model.WangguanTbl{})
	c.JSON(http.StatusOK, nil)
}

// LoRaDel /system/LoRaDel body {xuanzhong:[id...]}
func (a *AdminLoRaController) LoRaDel(c *gin.Context) {
	var req struct {
		Xuanzhong []string `json:"xuanzhong"`
	}
	if err := c.ShouldBindJSON(&req); err == nil {
		for _, id := range req.Xuanzhong {
			conf.Db.Where("id = ?", id).Delete(&model.WangguanTbl{})
		}
	}
	c.String(http.StatusOK, "删除成功！")
}

func loraSearch(c *gin.Context, fields []string) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	for _, field := range fields {
		var max int64
		conf.Db.Model(&model.TabGateway{}).Where(field+" LIKE ?", "%"+tiaojian+"%").Count(&max)
		if max != 0 {
			var gateways []model.TabGateway
			conf.Db.Model(&model.TabGateway{}).Where(field+" LIKE ?", "%"+tiaojian+"%").
				Order("install_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&gateways)
			list := make([]gatewayDTO, 0, len(gateways))
			for _, g := range gateways {
				list = append(list, gatewayToDTO(g))
			}
			c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
			return
		}
	}
	c.JSON(http.StatusOK, nil)
}

// LoRasousuo /system/LoRasousuo
func (a *AdminLoRaController) LoRasousuo(c *gin.Context) {
	loraSearch(c, []string{"mac", "iccid", "name", "install_place", "install_time"})
}

// SearchByMac /system/searchByMac?mac=
func (a *AdminLoRaController) SearchByMac(c *gin.Context) {
	current, size := parsePage(c)
	mac := c.Query("mac")
	if mac == "" {
		a.GetLoRapage(c)
		return
	}
	var max int64
	conf.Db.Model(&model.TabGateway{}).Where("mac LIKE ?", "%"+mac+"%").Count(&max)
	var gateways []model.TabGateway
	conf.Db.Model(&model.TabGateway{}).Where("mac LIKE ?", "%"+mac+"%").
		Order("install_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&gateways)
	list := make([]gatewayDTO, 0, len(gateways))
	for _, g := range gateways {
		list = append(list, gatewayToDTO(g))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
}

// RaChange /system/raChange → wangguan_tbl 按安装地址模糊搜索
func (a *AdminLoRaController) RaChange(c *gin.Context) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	var max int64
	conf.Db.Model(&model.WangguanTbl{}).Where("azadd LIKE ?", "%"+tiaojian+"%").Count(&max)
	var rows []model.WangguanTbl
	conf.Db.Model(&model.WangguanTbl{}).Where("azadd LIKE ?", "%"+tiaojian+"%").
		Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	c.JSON(http.StatusOK, gin.H{"max": max, "list": rows})
}

// AdminLockController 对齐 Java System.controller.SystemLockCon（车位锁，legacy raw 返回）。
type AdminLockController struct{}

// GetlockPage /system/getlockPage?page=&length=
func (a *AdminLockController) GetlockPage(c *gin.Context) {
	current, size := parsePage(c)
	var locks []model.LockTbl
	conf.Db.Model(&model.LockTbl{}).Order("install_time desc").
		Offset(pageOffset(current, size)).Limit(size).Find(&locks)
	list := make([]lockDTO, 0, len(locks))
	for _, l := range locks {
		list = append(list, lockToDTO(l))
	}
	c.JSON(http.StatusOK, list)
}

// GetChargePage /system/getChargePage?page=&length=
func (a *AdminLockController) GetChargePage(c *gin.Context) {
	current, size := parsePage(c)
	var rows []chargeDTO
	conf.Db.Raw("SELECT placeid, charge FROM charge_tbl LIMIT ?,?", pageOffset(current, size), size).Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// GetlockMax /system/getlockMax
func (a *AdminLockController) GetlockMax(c *gin.Context) {
	var n int64
	conf.Db.Model(&model.LockTbl{}).Count(&n)
	c.JSON(http.StatusOK, n)
}

// GetPlace /system/getPlace body {placeid}
func (a *AdminLockController) GetPlace(c *gin.Context) {
	var req struct {
		Placeid string `json:"placeid"`
	}
	_ = c.ShouldBindJSON(&req)
	var p model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", req.Placeid).First(&p).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, placeToDTO(p))
}

// disablePlace 切换车位状态，返回提示语。
func disablePlace(placeid string) string {
	var p model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&p).Error; err != nil {
		return "禁用失败，车位状态非可使用"
	}
	switch p.State {
	case "可使用":
		conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("state", "已禁用")
		return "禁用成功！"
	case "已禁用":
		conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("state", "可使用")
		return "解除禁用成功！"
	default:
		return "禁用失败，车位状态非可使用"
	}
}

// Disable /system/disable body {placeid}
func (a *AdminLockController) Disable(c *gin.Context) {
	var req struct {
		Placeid string `json:"placeid"`
	}
	_ = c.ShouldBindJSON(&req)
	c.String(http.StatusOK, disablePlace(req.Placeid))
}

// LockChange /system/lockChange body {page,length,tiaojian}
func (a *AdminLockController) LockChange(c *gin.Context) {
	var req struct {
		Page     string `json:"page"`
		Length   string `json:"length"`
		Tiaojian string `json:"tiaojian"`
	}
	_ = c.ShouldBindJSON(&req)
	current := 1
	size := 10
	if req.Page != "" {
		current, _ = strconv.Atoi(req.Page)
	}
	if req.Length != "" {
		size, _ = strconv.Atoi(req.Length)
	}

	db := conf.Db.Model(&model.LockTbl{})
	if req.Tiaojian == "已禁用" {
		db = db.Where("state = ?", "已禁用")
	} else {
		db = db.Where("state <> ?", "已禁用")
	}
	var locks []model.LockTbl
	db.Offset(pageOffset(current, size)).Limit(size).Find(&locks)
	list := make([]lockDTO, 0, len(locks))
	for _, l := range locks {
		list = append(list, lockToDTO(l))
	}
	c.JSON(http.StatusOK, gin.H{"max": len(list), "list": list})
}

// MushDis /system/mushDis body {xuanzhong:[placeid...]}
func (a *AdminLockController) MushDis(c *gin.Context) {
	var req struct {
		Xuanzhong []string `json:"xuanzhong"`
	}
	if err := c.ShouldBindJSON(&req); err == nil {
		for _, id := range req.Xuanzhong {
			disablePlace(id)
		}
	}
	c.String(http.StatusOK, "批量禁用/解禁成功!")
}

// Sousuo /system/lock/sousuo?tiaojian=&page=&length=
func (a *AdminLockController) Sousuo(c *gin.Context) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	for _, field := range []string{"fix_date", "placeid", "lockid", "lockmac", "state"} {
		var max int64
		conf.Db.Model(&model.LockTbl{}).Where(field+" LIKE ?", "%"+tiaojian+"%").Count(&max)
		if max != 0 {
			var locks []model.LockTbl
			conf.Db.Model(&model.LockTbl{}).Where(field+" LIKE ?", "%"+tiaojian+"%").
				Offset(pageOffset(current, size)).Limit(size).Find(&locks)
			list := make([]lockDTO, 0, len(locks))
			for _, l := range locks {
				list = append(list, lockToDTO(l))
			}
			c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
			return
		}
	}
	c.JSON(http.StatusOK, nil)
}

// Accessupdatestate /system/accessupdatestate body {placeid,ownerid}
func (a *AdminLockController) Accessupdatestate(c *gin.Context) {
	var req struct {
		Placeid string `json:"placeid"`
		Ownerid string `json:"ownerid"`
	}
	_ = c.ShouldBindJSON(&req)

	// 插入消息 + 标记充电桩 + 通过审核（对齐 Java access()）
	conf.Db.Exec("INSERT INTO message_tbl (ownerid, msgTime, message, status) VALUES (?,?,?,?)",
		req.Ownerid, time.Now().Format("2006-01-02 15:04:05"), "充电桩申请已通过~", "未读")
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", req.Placeid).Update("ischarge", 1)
	res := conf.Db.Exec("UPDATE charge_tbl SET charge = '已通过' WHERE placeid = ?", req.Placeid)

	if res.RowsAffected != 0 {
		c.String(http.StatusOK, "已通过")
		return
	}
	c.String(http.StatusOK, "重新请求")
}

// AdminFaultController 对齐 Java System.controller.faultCon（故障，legacy raw 返回）。
type AdminFaultController struct{}

// GetFault /system/getFault?page=&length=
func (a *AdminFaultController) GetFault(c *gin.Context) {
	current, size := parsePage(c)
	var max int64
	conf.Db.Model(&model.FaultTbl{}).Count(&max)
	var faults []model.FaultTbl
	conf.Db.Model(&model.FaultTbl{}).Order("putTime desc").
		Offset(pageOffset(current, size)).Limit(size).Find(&faults)
	list := make([]faultDTO, 0, len(faults))
	for _, f := range faults {
		list = append(list, faultToDTO(f))
	}
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
}

// FaultMushDel /system/faultMushDel body {xuanzhong:[faultid...]}
func (a *AdminFaultController) FaultMushDel(c *gin.Context) {
	var req struct {
		Xuanzhong []string `json:"xuanzhong"`
	}
	if err := c.ShouldBindJSON(&req); err == nil {
		for _, id := range req.Xuanzhong {
			conf.Db.Where("faultid = ?", id).Delete(&model.FaultTbl{})
		}
	}
	c.String(http.StatusOK, "批量删除成功！")
}

// GetFaultById /system/getFaultById?faultid=
func (a *AdminFaultController) GetFaultById(c *gin.Context) {
	var f model.FaultTbl
	if err := conf.Db.Where("faultid = ?", c.Query("faultid")).First(&f).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, faultToDTO(f))
}

// UpdateState /system/updateState?faultid=&state=
func (a *AdminFaultController) UpdateState(c *gin.Context) {
	conf.Db.Model(&model.FaultTbl{}).Where("faultid = ?", c.Query("faultid")).Update("state", c.Query("state"))
	c.String(http.StatusOK, "状态更新成功！")
}

// Sousuo /system/fault/sousuo?tiaojian=&page=&length=
func (a *AdminFaultController) Sousuo(c *gin.Context) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	for _, field := range []string{"lockmac", "putTime", "error", "state"} {
		var max int64
		conf.Db.Model(&model.FaultTbl{}).Where(field+" LIKE ?", "%"+tiaojian+"%").Count(&max)
		if max != 0 {
			var faults []model.FaultTbl
			conf.Db.Model(&model.FaultTbl{}).Where(field+" LIKE ?", "%"+tiaojian+"%").
				Offset(pageOffset(current, size)).Limit(size).Find(&faults)
			list := make([]faultDTO, 0, len(faults))
			for _, f := range faults {
				list = append(list, faultToDTO(f))
			}
			c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
			return
		}
	}
	c.JSON(http.StatusOK, nil)
}
