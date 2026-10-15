package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ WxGuzhangController（base /wx，故障） ============

// WxGuzhangController 对齐 Java controller.wxGuzhangCon。
type WxGuzhangController struct{}

// GuzhangTijiao /wx/guzhangtijiao（multipart：file + fault 表单字段 + dir）
// 对齐 wxGuzhangSerImpl.submit：保存图片→生成 faultid→置 putTime/state→插入 fault_tbl。
func (c *WxGuzhangController) GuzhangTijiao(ctx *gin.Context) {
	openid := ctx.PostForm("openid")
	lockmac := ctx.PostForm("lockmac")
	errDesc := ctx.PostForm("error")

	// Java saveFileUtil.save(file, dir) 将图片上传到服务器并返回访问 URL。
	// Go 侧省略外部文件存储，仅取原始文件名占位，DB image 字段保留。
	image := ""
	if fh, e := ctx.FormFile("file"); e == nil {
		image = fh.Filename
	}

	rec := model.FaultTbl{
		Faultid: simpleUUID(),
		Openid:  openid,
		Lockmac: lockmac,
		Error:   errDesc,
		Image:   image,
		PutTime: timePtr(time.Now()),
		State:   "待处理",
	}
	// Java submit 无论文件存储是否失败均返回 "ok"（IOException 被吞掉）。
	_ = repository.Create(&rec)
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// GetAllFault /wx/getAllfault —— 对齐 faultDao.getAll：返回全部故障。
func (c *WxGuzhangController) GetAllFault(ctx *gin.Context) {
	var rows []model.FaultTbl
	repository.FindAll(&rows)
	list := make([]faultDTO, 0, len(rows))
	for _, f := range rows {
		list = append(list, faultToDTO(f))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// UpdateState /wx/updateState?faultid=&state= —— 对齐 faultDao.updateState，返回受影响行数。
func (c *WxGuzhangController) UpdateState(ctx *gin.Context) {
	rows := repository.UpdateFaultState(ctx.Query("faultid"), ctx.Query("state"))
	ctx.JSON(http.StatusOK, ResultSuccess(int(rows)))
}

// GetMaintenanceTime /wx/getMaintenance_time?Maintenance_time=&lockmac=&name=
// 对齐 faultDao.getMaintenance_time：update fault_tbl set Maintenance_time,name where lockmac。
func (c *WxGuzhangController) GetMaintenanceTime(ctx *gin.Context) {
	mt := ctx.Query("Maintenance_time")
	lockmac := ctx.Query("lockmac")
	name := ctx.Query("name")

	updates := map[string]interface{}{"name": name}
	if t := parseTimeLoose(mt); !t.IsZero() {
		updates["Maintenance_time"] = t
	}
	if repository.UpdateFaultByLockmac(lockmac, updates) > 0 {
		ctx.JSON(http.StatusOK, ResultSuccess("ok"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess("fail"))
}
