package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gen"

	"caicai-go/objects"
)

// AdminUserController 对齐 Java System.controller.OmUserCon（运维用户审核管理）。
type AdminUserController struct{}

// omUserConds 构建运维用户查询条件：omid 与 om_enable 均非空，phone 可选模糊匹配。
func omUserConds(phone string) []gen.Condition {
	conds := []gen.Condition{
		objects.UserTbl.Omid.IsNotNull(),
		objects.UserTbl.OmEnable.IsNotNull(),
	}
	if phone != "" {
		conds = append(conds, objects.UserTbl.Phone.Like("%"+phone+"%"))
	}
	return conds
}

// GetOmUserByPhone GET /system/user/getOmUserByPhone?phone=&current=&size=
func (a *AdminUserController) GetOmUserByPhone(c *gin.Context) {
	current, size := parsePage(c)
	phone := c.Query("phone")
	ctx := c.Request.Context()
	conds := omUserConds(phone)

	total, err := objects.UserTbl.WithContext(ctx).Where(conds...).Count()
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}
	users, err := objects.UserTbl.WithContext(ctx).Where(conds...).
		Offset(pageOffset(current, size)).Limit(size).Find()
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}

	list := make([]UserDTO, 0, len(users))
	for _, u := range users {
		list = append(list, userToDTO(*u))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetOmUserInfoAll GET /system/user/getOmUserInfoAll?current=&size=
func (a *AdminUserController) GetOmUserInfoAll(c *gin.Context) {
	current, size := parsePage(c)
	ctx := c.Request.Context()
	conds := omUserConds("")

	total, err := objects.UserTbl.WithContext(ctx).Where(conds...).Count()
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}
	users, err := objects.UserTbl.WithContext(ctx).Where(conds...).
		Offset(pageOffset(current, size)).Limit(size).Find()
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}

	list := make([]UserDTO, 0, len(users))
	for _, u := range users {
		list = append(list, userToDTO(*u))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// UpdateOmEnable GET /system/user/updateOmEnable?enable=&openId=
// 注意 Java 用 openId 参数匹配 omid 列。
func (a *AdminUserController) UpdateOmEnable(c *gin.Context) {
	enable := c.Query("enable")
	openID := c.Query("openId")
	if enable == "" || openID == "" {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}

	info, err := objects.UserTbl.WithContext(c.Request.Context()).
		Where(objects.UserTbl.Omid.Eq(openID)).
		Update(objects.UserTbl.OmEnable, enable)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(info.RowsAffected > 0))
}
