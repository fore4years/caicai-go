package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

// registerAdminPlaceApply 注册后台车位申请审核（placeApplyCon）路由。
func registerAdminPlaceApply(r *gin.Engine) {
	c := new(handler.AdminPlaceApplyController)
	r.Any("/system/getPlaceApplyMax", c.GetPlaceApplyMax)
	r.Any("/system/getPlaceApplyPaging", c.GetPlaceApplyPaging)
	r.Any("/system/getPlaceApplyById", c.GetPlaceApplyById)
	r.Any("/system/doPlaceApply", c.DoPlaceApply)
	r.Any("/system/paChange", c.PaChange)
	r.Any("/system/mushDel", c.MushDel)
	r.Any("/system/sousuo", c.Sousuo)
}
