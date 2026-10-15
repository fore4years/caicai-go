package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/middleware"
	"caicai-go/repository"
)

// ============ UserController 其余端点（挂在 OrderController 上，base /user） ============

// GetUserInfoByOpenId 对应 Java UserController.getUserInfoByOpenId：根据 openid 查用户。
func (o *OrderController) GetUserInfoByOpenId(c *gin.Context) {
	u, err := repository.UserByOpenid(c.Request.Context(), c.Param("openid"))
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(userToDTO(*u)))
}

// UpdateUserInfo 对应 Java UserServiceImpl.updateUserInfo：
//   - 主键是 phone，更新条件为 phone = 当前登录用户手机号（JWT aud）。
//   - phone 取当前登录用户手机号。
//   - omid / openid / free_time / integral / id_card_avatar / id_card_emblem 置空，不参与更新。
func (o *OrderController) UpdateUserInfo(c *gin.Context) {
	var req struct {
		NickName     *string          `json:"nickName"`
		Province     *string          `json:"province"`
		City         *string          `json:"city"`
		PlateNum     *string          `json:"plateNum"`
		IDNumber     *string          `json:"idNumber"`
		Name         *string          `json:"name"`
		Avatar       []byte           `json:"avatar"`
		IDEntity     *string          `json:"idEntity"`
		OmEnable     *string          `json:"omEnable"`
		IsSteer      *string          `json:"isSteer"`
		IsProcedure  *string          `json:"isProcedure"`
		IsLogin      *string          `json:"isLogin"`
		YiparlOpenid *string          `json:"yiparlOpenid"`
		Balans       *decimal.Decimal `json:"balans"`
		FreezeBalans *decimal.Decimal `json:"freezeBalans"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}

	tokenPhone := c.GetString("phone")
	updates := map[string]interface{}{"phone": tokenPhone}
	if req.NickName != nil {
		updates["nick_name"] = *req.NickName
	}
	if req.Province != nil {
		updates["province"] = *req.Province
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.PlateNum != nil {
		updates["plate_num"] = *req.PlateNum
	}
	if req.IDNumber != nil {
		updates["id_number"] = *req.IDNumber
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if len(req.Avatar) > 0 {
		updates["avatar"] = req.Avatar
	}
	if req.IDEntity != nil {
		updates["id_entity"] = *req.IDEntity
	}
	if req.OmEnable != nil {
		updates["om_enable"] = *req.OmEnable
	}
	if req.IsSteer != nil {
		updates["is_steer"] = *req.IsSteer
	}
	if req.IsProcedure != nil {
		updates["is_procedure"] = *req.IsProcedure
	}
	if req.IsLogin != nil {
		updates["is_login"] = *req.IsLogin
	}
	if req.YiparlOpenid != nil {
		updates["yiparl_openid"] = *req.YiparlOpenid
	}
	if req.Balans != nil {
		updates["balans"] = *req.Balans
	}
	if req.FreezeBalans != nil {
		updates["freeze_balans"] = *req.FreezeBalans
	}

	rowsAffected := repository.UpdateUserFieldsByPhone(tokenPhone, updates)
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// UpdatePhone 对应 Java UserServiceImpl.updatePhone：返回字符串消息。
func (o *OrderController) UpdatePhone(c *gin.Context) {
	var req struct {
		Openid string `json:"openid"`
		Phone  string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(""))
		return
	}
	if req.Phone == c.GetString("phone") {
		c.JSON(http.StatusOK, ResultSuccess("修改失败不能使用同一个手机号"))
		return
	}
	rowsAffected := repository.UpdateUserPhoneField(req.Openid, req.Phone)
	if rowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess("修改失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("修改成功"))
}

// RefToken 对应 Java UserServiceImpl.refToken：解析旧 token 后重新签发。
func (o *OrderController) RefToken(c *gin.Context) {
	phone, openid, name, err := middleware.ParseToken(c.Query("token"))
	if err != nil {
		c.JSON(http.StatusOK, ResultError(401, "未登录"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(getToken(phone, openid, name)))
}

// UpdateUserIdEntity 对应 Java UserServiceImpl.updateUserIdEntity：返回 "ok"/"false"。
func (o *OrderController) UpdateUserIdEntity(c *gin.Context) {
	rowsAffected := repository.UpdateUserIdEntityField(c.Query("openid"), c.Query("idEntity"))
	if rowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess("false"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("ok"))
}

// UpdateUserStatus 对应 Java UserServiceImpl.updateUserStatus：仅更新非空参数，返回 "ok"/"false"。
func (o *OrderController) UpdateUserStatus(c *gin.Context) {
	openid := c.GetString("openid")
	updates := map[string]interface{}{}
	if v := c.Query("isSteer"); v != "" {
		updates["is_steer"] = v
	}
	if v := c.Query("isProcedure"); v != "" {
		updates["is_procedure"] = v
	}
	if v := c.Query("isLogin"); v != "" {
		updates["is_login"] = v
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, ResultSuccess("false"))
		return
	}
	rowsAffected := repository.UpdateUserFieldsByOpenid(openid, updates)
	if rowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess("false"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("ok"))
}

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
	rowsAffected := repository.UpdateUserAvatar(openid, data)
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// UpdateEmblem POST /user/updateEmblem 对齐 Java updateEmblem：国徽面图片写入 tab_image，回写 id_card_emblem。
func (o *OrderController) UpdateEmblem(c *gin.Context) {
	imageID, ok := saveImageFromCtx(c)
	if !ok {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	openid := c.GetString("openid")
	rowsAffected := repository.UpdateUserIdCardEmblem(openid, imageID)
	if rowsAffected == 0 {
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
	rowsAffected := repository.UpdateUserIdCardAvatar(openid, imageID)
	if rowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}
