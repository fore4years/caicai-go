package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
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
	repository.Count(&model.TabGateway{}, &n)
	c.JSON(http.StatusOK, n)
}

// GetLoRapage /system/getLoRapage?page=&length= → List<GatewayBean>
func (a *AdminLoRaController) GetLoRapage(c *gin.Context) {
	current, size := parsePage(c)
	var gateways []model.TabGateway
	repository.Page(&model.TabGateway{}, "install_time desc", pageOffset(current, size), size, &gateways)
	list := make([]gatewayDTO, 0, len(gateways))
	for _, g := range gateways {
		list = append(list, gatewayToDTO(g))
	}
	c.JSON(http.StatusOK, list)
}

// GetGatewayById /system/getGatewayById?id=
func (a *AdminLoRaController) GetGatewayById(c *gin.Context) {
	var g model.TabGateway
	if err := repository.FindByID(&g, c.Query("id")).Error; err != nil {
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
	repository.UpdateByID(&model.WangguanTbl{}, req.ID, map[string]interface{}{
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
	repository.DeleteByID(&model.WangguanTbl{}, c.Param("id"))
	c.JSON(http.StatusOK, nil)
}

// LoRaDel /system/LoRaDel body {xuanzhong:[id...]}
func (a *AdminLoRaController) LoRaDel(c *gin.Context) {
	var req struct {
		Xuanzhong []string `json:"xuanzhong"`
	}
	if err := c.ShouldBindJSON(&req); err == nil {
		for _, id := range req.Xuanzhong {
			repository.DeleteByID(&model.WangguanTbl{}, id)
		}
	}
	c.String(http.StatusOK, "删除成功！")
}

func loraSearch(c *gin.Context, fields []string) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	for _, field := range fields {
		max, gateways := repository.SearchGateways(field, tiaojian, pageOffset(current, size), size)
		if max != 0 {
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
	if strings.TrimSpace(mac) == "" {
		var total int64
		repository.Count(&model.TabGateway{}, &total)
		if total == 0 {
			c.JSON(http.StatusOK, nil)
			return
		}
		var gateways []model.TabGateway
		repository.Page(&model.TabGateway{}, "install_time desc", pageOffset(current, size), size, &gateways)
		list := make([]gatewayDTO, 0, len(gateways))
		for _, g := range gateways {
			list = append(list, gatewayToDTO(g))
		}
		c.JSON(http.StatusOK, gin.H{"max": total, "list": list})
		return
	}
	max, gateways := repository.PageGatewaysByMac(mac, pageOffset(current, size), size)
	if max == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}
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
	max, rows := repository.SearchWangguanByAzadd(tiaojian, pageOffset(current, size), size)
	c.JSON(http.StatusOK, gin.H{"max": max, "list": rows})
}

// AdminLockController 对齐 Java System.controller.SystemLockCon（车位锁，legacy raw 返回）。
type AdminLockController struct{}

// GetlockPage /system/getlockPage?page=&length=
func (a *AdminLockController) GetlockPage(c *gin.Context) {
	current, size := parsePage(c)
	var locks []model.LockTbl
	repository.Page(&model.LockTbl{}, "", pageOffset(current, size), size, &locks)
	list := make([]lockDTO, 0, len(locks))
	for _, l := range locks {
		list = append(list, lockToDTO(l))
	}
	c.JSON(http.StatusOK, list)
}

// GetChargePage /system/getChargePage?page=&length=
func (a *AdminLockController) GetChargePage(c *gin.Context) {
	current, size := parsePage(c)
	charges := repository.AllCharges(pageOffset(current, size), size)
	rows := make([]chargeDTO, 0, len(charges))
	for _, ch := range charges {
		rows = append(rows, chargeDTO{Placeid: ch.Placeid, Charge: ch.Charge})
	}
	c.JSON(http.StatusOK, rows)
}

// GetlockMax /system/getlockMax
func (a *AdminLockController) GetlockMax(c *gin.Context) {
	var n int64
	repository.Count(&model.LockTbl{}, &n)
	c.JSON(http.StatusOK, n)
}

// GetPlace /system/getPlace body {placeid}
func (a *AdminLockController) GetPlace(c *gin.Context) {
	var req struct {
		Placeid string `json:"placeid"`
	}
	_ = c.ShouldBindJSON(&req)
	p, err := repository.PlaceByPlaceidDB(req.Placeid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, placeToDTO(*p))
}

// disablePlace 切换车位状态，返回提示语。
func disablePlace(placeid string) string {
	p, err := repository.PlaceByPlaceidDB(placeid)
	if err != nil {
		return "禁用失败，车位状态非可使用"
	}
	switch p.State {
	case "可使用":
		repository.UpdatePlaceStateByPlaceid(placeid, "已禁用")
		return "禁用成功！"
	case "已禁用":
		repository.UpdatePlaceStateByPlaceid(placeid, "可使用")
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
// 对齐 Java：左连 place_tbl 按车位状态筛选，忽略分页返回全部（max 为返回条数）。
func (a *AdminLockController) LockChange(c *gin.Context) {
	var req struct {
		Page     string `json:"page"`
		Length   string `json:"length"`
		Tiaojian string `json:"tiaojian"`
	}
	_ = c.ShouldBindJSON(&req)

	locks := repository.LocksByState(req.Tiaojian)
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
		max, locks := repository.SearchLocks(field, tiaojian, pageOffset(current, size), size)
		if max != 0 {
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
	repository.InsertMessage(req.Ownerid, time.Now().Format("2006-01-02 15:04:05"), "充电桩申请已通过~", "未读")
	repository.UpdatePlaceIscharge(req.Placeid)
	rowsAffected := repository.UpdateChargeStateByPlaceid(req.Placeid, "已通过")

	if rowsAffected != 0 {
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
	repository.Count(&model.FaultTbl{}, &max)
	var faults []model.FaultTbl
	repository.Page(&model.FaultTbl{}, "putTime desc", pageOffset(current, size), size, &faults)
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
			repository.DeleteFaultByFaultid(id)
		}
	}
	c.String(http.StatusOK, "批量删除成功！")
}

// GetFaultById /system/getFaultById?faultid=
func (a *AdminFaultController) GetFaultById(c *gin.Context) {
	f, err := repository.FaultByFaultid(c.Query("faultid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, faultToDTO(*f))
}

// UpdateState /system/updateState?faultid=&state=
func (a *AdminFaultController) UpdateState(c *gin.Context) {
	repository.UpdateFaultState(c.Query("faultid"), c.Query("state"))
	c.String(http.StatusOK, "状态更新成功！")
}

// Sousuo /system/fault/sousuo?tiaojian=&page=&length=
func (a *AdminFaultController) Sousuo(c *gin.Context) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	for _, field := range []string{"lockmac", "putTime", "error", "state"} {
		max, faults := repository.SearchFaults(field, tiaojian, pageOffset(current, size), size)
		if max != 0 {
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
