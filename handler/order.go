package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// OrderController 对应 Java OrderController（/order）。
type OrderController struct{}

// GetNotFinish 根据当前用户 openid 获取未完成订单（对应 Java OrderController.getNotFinish）。
func (o *OrderController) GetNotFinish(c *gin.Context) {
	openid := c.GetString("openid")
	order, err := repository.FirstNotFinishedOrderByOpenid(c.Request.Context(), openid)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(*order)))
}

// GetUserInfo 获取当前用户信息（对应 Java UserController.getUserInfo）。
// 用 phone 查询：JWT 的 aud 恒为手机号，跨小程序（主小程序/运维/驿帕尔）唯一，
// 避免运维登录时 JWT 的 openid 实际是 omid、按 openid 查不到用户的问题。
func (o *OrderController) GetUserInfo(c *gin.Context) {
	phone := c.GetString("phone")
	user, err := repository.UserByPhone(c.Request.Context(), phone)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(userToDTO(*user)))
}

// userToDTO 将表模型映射为与 Java UserBean 序列化一致的 DTO。
func userToDTO(u model.UserTbl) UserDTO {
	return UserDTO{
		Openid:       u.Openid,
		Omid:         u.Omid,
		YiparlOpenid: u.YiparlOpenid,
		NickName:     u.NickName,
		Province:     u.Province,
		City:         u.City,
		Phone:        u.Phone,
		Integral:     u.Integral,
		FreeTime:     u.FreeTime,
		PlateNum:     u.PlateNum,
		IDNumber:     u.IDNumber,
		Name:         u.Name,
		Avatar:       u.Avatar,
		IDCardEmblem: u.IDCardEmblem,
		IDCardAvatar: u.IDCardAvatar,
		Balans:       u.Balans.Round(2),
		FreezeBalans: u.FreezeBalans,
		IDEntity:     u.IDEntity,
		OmEnable:     u.OmEnable,
		IsSteer:      u.IsSteer,
		IsProcedure:  u.IsProcedure,
		IsLogin:      u.IsLogin,
	}
}
