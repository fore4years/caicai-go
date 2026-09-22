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

// GetAllUsers GET /system/getAllUsers，返回 Result<[]systemRow>。
func (a *AdminAuthController) GetAllUsers(c *gin.Context) {
	var users []systemRow
	if err := conf.Db.Raw(
		"SELECT id, account, password, is_admin FROM system_tbl ORDER BY id",
	).Scan(&users).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(users))
}

// UpdateUserAdminStatus POST /system/updateUserAdminStatus?userId=xxx&isAdmin=true。
func (a *AdminAuthController) UpdateUserAdminStatus(c *gin.Context) {
	userId := c.Query("userId")
	isAdmin := c.Query("isAdmin") == "true"

	var isAdminInt int
	if isAdmin {
		isAdminInt = 1
	}

	res := conf.Db.Exec("UPDATE system_tbl SET is_admin = ? WHERE id = ?", isAdminInt, userId)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
