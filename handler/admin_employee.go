package handler

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
)

// employeeDTO 对应 Java domain.OmEmployee 的序列化字段（驼峰）。
// audit_status 用 int（DB 值 0/1/-1），不能用 model 里的 bool。
type employeeDTO struct {
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

// AdminEmployeeController 对齐 Java controller.OmEmployeeController（运维职工管理）。
type AdminEmployeeController struct{}

// generateEmployeeID 生成 QCYC + 4 位随机数字的工号，并保证唯一。
func generateEmployeeID() string {
	for {
		id := "QCYC" + fmt.Sprintf("%04d", rand.Intn(10000))
		var count int64
		conf.Db.Table("om_employee").Where("employee_id = ?", id).Count(&count)
		if count == 0 {
			return id
		}
	}
}

// Add POST /om/employee/add
func (a *AdminEmployeeController) Add(c *gin.Context) {
	var req struct {
		Name       string `json:"name"`
		Password   string `json:"password"`
		Phone      string `json:"phone"`
		IDCard     string `json:"idCard"`
		EntryDate  string `json:"entryDate"`
		Department string `json:"department"`
		Position   string `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "运维职工信息新增失败"))
		return
	}

	employeeID := generateEmployeeID()
	err := conf.Db.Exec(
		"INSERT INTO om_employee (employee_id, name, password, phone, id_card, entry_date, department, position, audit_status) VALUES (?,?,?,?,?,?,?,?,?)",
		employeeID, req.Name, req.Password, req.Phone, req.IDCard, req.EntryDate, req.Department, req.Position, 0,
	).Error
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, "手机号已存在，注册失败"))
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "运维职工信息新增成功", Data: true})
}

// GetByName GET /om/employee/getByName?name=
func (a *AdminEmployeeController) GetByName(c *gin.Context) {
	var e employeeDTO
	if err := conf.Db.Raw(employeeSelect+" WHERE name = ?", c.Query("name")).Scan(&e).Error; err != nil || e.EmployeeID == "" {
		c.JSON(http.StatusOK, ResultError(404, "用户不存在"))
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "查询成功", Data: e})
}

// GetAll GET /om/employee/getAll
func (a *AdminEmployeeController) GetAll(c *gin.Context) {
	var employees []employeeDTO
	if err := conf.Db.Raw(employeeSelect).Scan(&employees).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "查询异常："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "查询成功", Data: employees})
}

// DeleteById DELETE /om/employee/deleteById?employeeId=
func (a *AdminEmployeeController) DeleteById(c *gin.Context) {
	employeeID := c.Query("employeeId")
	var count int64
	conf.Db.Table("om_employee").Where("employee_id = ?", employeeID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusOK, ResultError(500, "运维职工信息删除失败，可能是员工不存在"))
		return
	}
	conf.Db.Table("om_employee").Where("employee_id = ?", employeeID).Delete(nil)
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "运维职工信息删除成功", Data: true})
}

// UpdateStatus GET /om/employee/updateStatus?employeeId=&status=
func (a *AdminEmployeeController) UpdateStatus(c *gin.Context) {
	employeeID := c.Query("employeeId")
	status := queryInt(c, "status", 99)
	if status != 0 && status != 1 && status != -1 {
		c.JSON(http.StatusOK, ResultError(500, "审核状态更新失败，可能是员工不存在或状态值不合法"))
		return
	}
	res := conf.Db.Table("om_employee").Where("employee_id = ?", employeeID).Update("audit_status", status)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultError(500, "审核状态更新失败，可能是员工不存在或状态值不合法"))
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "审核状态更新成功", Data: true})
}
