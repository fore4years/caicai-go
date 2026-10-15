package repository

import (
	"caicai-go/conf"
)

// SystemUser 对应 system_tbl 行。
// 注意 is_admin 用 int 而非 bool：前端按 ===1/===2 判断权限，DB 值可能为 0/1/2。
type SystemUser struct {
	ID       string `json:"id"`
	Account  string `json:"account"`
	Password string `json:"password"`
	IsAdmin  int    `json:"isAdmin"`
}

// SystemUserByAccount 按账号查询系统用户。
func SystemUserByAccount(account string) (*SystemUser, error) {
	var s SystemUser
	if err := conf.Db.Raw(
		"SELECT id, account, password, is_admin FROM system_tbl WHERE account = ?", account,
	).Scan(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// AllSystemUsers 查询所有系统用户（按 id 升序）。
func AllSystemUsers() ([]SystemUser, error) {
	var users []SystemUser
	if err := conf.Db.Raw(
		"SELECT id, account, password, is_admin FROM system_tbl ORDER BY id",
	).Scan(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateSystemPassword 更新系统用户密码。
func UpdateSystemPassword(account, password string) {
	conf.Db.Exec("UPDATE system_tbl SET password = ? WHERE account = ?", password, account)
}

// UpdateSystemUserAdminStatus 更新系统用户权限，返回受影响行数与错误。
func UpdateSystemUserAdminStatus(userID string, adminStatus int) (int64, error) {
	res := conf.Db.Exec("UPDATE system_tbl SET is_admin = ? WHERE id = ?", adminStatus, userID)
	return res.RowsAffected, res.Error
}
