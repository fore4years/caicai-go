package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
)

// systemRow 对应 Java domain.system（system_tbl）。
// 注意 is_admin 用 int 而非 bool：前端按 ===1/===2 判断权限，DB 值可能为 0/1/2。
type systemRow struct {
	ID       string `json:"id"`
	Account  string `json:"account"`
	Password string `json:"password"`
	IsAdmin  int    `json:"isAdmin"`
}

// AdminAuthController 对齐 Java System.controller.loginCon / SystemUserCon。
type AdminAuthController struct{}

// Login POST /system/login，返回原始字符串（"" 成功，否则错误信息）。
func (a *AdminAuthController) Login(c *gin.Context) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusOK, "账号不存在")
		return
	}

	var s systemRow
	if err := conf.Db.Raw(
		"SELECT id, account, password, is_admin FROM system_tbl WHERE account = ?", req.Account,
	).Scan(&s).Error; err != nil || s.Account == "" {
		c.String(http.StatusOK, "账号不存在")
		return
	}
	if s.Password != req.Password {
		c.String(http.StatusOK, "密码错误")
		return
	}

	// 对齐 Java：成功后设置 username cookie（7 天）。
	c.SetCookie("username", req.Account, 7*24*60*60, "/", "", false, false)
	c.String(http.StatusOK, "")
}

// PwdChange POST /system/pwdChange，body {account, pwd_o, pwd_n}。
func (a *AdminAuthController) PwdChange(c *gin.Context) {
	var req struct {
		Account string `json:"account"`
		PwdO    string `json:"pwd_o"`
		PwdN    string `json:"pwd_n"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusOK, "no")
		return
	}

	var s systemRow
	if err := conf.Db.Raw(
		"SELECT id, account, password, is_admin FROM system_tbl WHERE account = ?", req.Account,
	).Scan(&s).Error; err != nil || s.Account == "" {
		c.String(http.StatusOK, "no")
		return
	}
	if s.Password != req.PwdO {
		c.String(http.StatusOK, "no")
		return
	}

	conf.Db.Exec("UPDATE system_tbl SET password = ? WHERE account = ?", req.PwdN, req.Account)
	c.String(http.StatusOK, "更改密码成功！")
}

// CheckAdmin GET /system/checkAdmin?account=xxx，返回 {code, msg, data:{isAdmin}}。
func (a *AdminAuthController) CheckAdmin(c *gin.Context) {
	account := strings.TrimSpace(c.Query("account"))
	if account == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "账号不能为空"})
		return
	}

	var s systemRow
	if err := conf.Db.Raw(
		"SELECT id, account, password, is_admin FROM system_tbl WHERE account = ?", account,
	).Scan(&s).Error; err != nil || s.Account == "" {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "账号不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "查询成功", "data": gin.H{"isAdmin": s.IsAdmin}})
}

// GetAllUsers GET /system/getAllUsers，返回 Result<Map>（对齐 Java SystemUserSerImpl.getAllUsers：
// data 为含 code/msg/data/total 的 Map，外层再包 Result.success）。
func (a *AdminAuthController) GetAllUsers(c *gin.Context) {
	var users []systemRow
	if err := conf.Db.Raw(
		"SELECT id, account, password, is_admin FROM system_tbl ORDER BY id",
	).Scan(&users).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{
			"code": 500, "msg": "获取用户数据失败：" + err.Error(), "data": gin.H{},
		}))
		return
	}
	if len(users) == 0 {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{
			"code": 404, "msg": "未找到用户数据", "data": gin.H{},
		}))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"code": 200, "msg": "获取用户数据成功", "data": users, "total": len(users),
	}))
}

// UpdateUserAdminStatus POST /system/updateUserAdminStatus?userId=xxx&isAdmin=true。
func (a *AdminAuthController) UpdateUserAdminStatus(c *gin.Context) {
	userId := c.Query("userId")
	isAdmin := c.Query("isAdmin") == "true"

	if strings.TrimSpace(userId) == "" {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{"code": 400, "msg": "用户ID不能为空"}))
		return
	}

	adminStatus := 0
	if isAdmin {
		adminStatus = 1
	}

	res := conf.Db.Exec("UPDATE system_tbl SET is_admin = ? WHERE id = ?", adminStatus, userId)
	if res.Error != nil {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{"code": 500, "msg": "更新用户权限失败：" + res.Error.Error()}))
		return
	}
	if res.RowsAffected > 0 {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{
			"code": 200, "msg": "用户权限更新成功",
			"data": gin.H{"userId": userId, "newAdminStatus": adminStatus},
		}))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{"code": 404, "msg": "用户不存在或更新失败"}))
}
