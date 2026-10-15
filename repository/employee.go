package repository

import (
	"caicai-go/conf"
)

// Employee 对应 Java domain.OmEmployee 的序列化字段（驼峰）。
// audit_status 用 int（DB 值 0/1/-1），不能用 model 里的 bool。
type Employee struct {
	EmployeeID  string `gorm:"column:employee_id" json:"employeeId"`
	Name        string `gorm:"column:name" json:"name"`
	Password    string `gorm:"column:password" json:"password"`
	Phone       string `gorm:"column:phone" json:"phone"`
	IDCard      string `gorm:"column:id_card" json:"idCard"`
	EntryDate   string `gorm:"column:entry_date" json:"entryDate"`
	Department  string `gorm:"column:department" json:"department"`
	Position    string `gorm:"column:position" json:"position"`
	AuditStatus int    `gorm:"column:audit_status" json:"auditStatus"`
}

const employeeSelect = "SELECT employee_id, name, password, phone, id_card, DATE_FORMAT(entry_date, '%Y-%m-%d') AS entry_date, department, position, audit_status FROM om_employee"

// EmployeeIDExists 判断工号是否存在。
func EmployeeIDExists(id string) bool {
	var count int64
	conf.Db.Table("om_employee").Where("employee_id = ?", id).Count(&count)
	return count > 0
}

// InsertEmployee 新增运维职工。
func InsertEmployee(employeeID, name, password, phone, idCard, entryDate, department, position string) error {
	return conf.Db.Exec(
		"INSERT INTO om_employee (employee_id, name, password, phone, id_card, entry_date, department, position, audit_status) VALUES (?,?,?,?,?,?,?,?,?)",
		employeeID, name, password, phone, idCard, entryDate, department, position, 0,
	).Error
}

// EmployeeByName 按姓名查询运维职工。
func EmployeeByName(name string) (*Employee, error) {
	var e Employee
	err := conf.Db.Raw(employeeSelect+" WHERE name = ?", name).Scan(&e).Error
	return &e, err
}

// AllEmployees 查询所有运维职工（按入职日期倒序）。
func AllEmployees() ([]Employee, error) {
	var employees []Employee
	err := conf.Db.Raw(employeeSelect + " ORDER BY entry_date DESC").Scan(&employees).Error
	return employees, err
}

// EmployeeExists 判断员工是否存在。
func EmployeeExists(employeeID string) bool {
	var count int64
	conf.Db.Table("om_employee").Where("employee_id = ?", employeeID).Count(&count)
	return count > 0
}

// DeleteEmployee 删除运维职工。
func DeleteEmployee(employeeID string) {
	conf.Db.Table("om_employee").Where("employee_id = ?", employeeID).Delete(nil)
}

// UpdateEmployeeStatus 更新运维职工审核状态，返回受影响行数。
func UpdateEmployeeStatus(employeeID string, status int) int64 {
	res := conf.Db.Table("om_employee").Where("employee_id = ?", employeeID).Update("audit_status", status)
	return res.RowsAffected
}
