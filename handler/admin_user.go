package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// AdminUserController 对齐 Java System.controller.OmUserCon（运维用户审核管理）。
type AdminUserController struct{}

// GetOmUserByPhone GET /system/user/getOmUserByPhone?phone=&current=&size=
func (a *AdminUserController) GetOmUserByPhone(c *gin.Context) {
	current, size := parsePage(c)
	phone := c.Query("phone")
	ctx := c.Request.Context()

	total, err := repository.CountOmUsers(ctx, phone)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}
	users, err := repository.ListOmUsers(ctx, phone, pageOffset(current, size), size)
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

	total, err := repository.CountOmUsers(ctx, "")
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}
	users, err := repository.ListOmUsers(ctx, "", pageOffset(current, size), size)
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

	rowsAffected, err := repository.UpdateUserOmEnable(c.Request.Context(), openID, enable)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}
