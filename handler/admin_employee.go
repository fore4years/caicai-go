package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// AdminEmployeeController 对齐 Java controller.OmEmployeeController（运维职工管理）。
type AdminEmployeeController struct{}

// generateEmployeeID 生成 QCYC + 4 位随机数字的工号，并保证唯一。
func generateEmployeeID() string {
	for {
		id := "QCYC" + fmt.Sprintf("%04d", rand.Intn(10000))
		if !repository.EmployeeIDExists(id) {
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
	err := repository.InsertEmployee(employeeID, req.Name, req.Password, req.Phone, req.IDCard, req.EntryDate, req.Department, req.Position)
	if err != nil {
		// 对齐 Java OmEmployeeController：唯一键冲突按异常信息区分是手机号重复还是其他字段重复。
		msg := err.Error()
		if strings.Contains(msg, "uk_phone") {
			c.JSON(http.StatusOK, ResultError(500, "手机号已存在，注册失败"))
		} else if strings.Contains(msg, "Duplicate") {
			c.JSON(http.StatusOK, ResultError(500, "数据重复，注册失败"))
		} else {
			c.JSON(http.StatusOK, ResultError(500, "运维职工信息新增异常："+msg))
		}
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "运维职工信息新增成功", Data: true})
}

// GetByName GET /om/employee/getByName?name=
func (a *AdminEmployeeController) GetByName(c *gin.Context) {
	e, err := repository.EmployeeByName(c.Query("name"))
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, "查询异常："+err.Error()))
		return
	}
	if e.EmployeeID == "" {
		c.JSON(http.StatusOK, ResultError(404, "用户不存在"))
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "查询成功", Data: e})
}

// GetAll GET /om/employee/getAll（对齐 Java OmEmployeeMapper.selectAll：按入职日期倒序）。
func (a *AdminEmployeeController) GetAll(c *gin.Context) {
	employees, err := repository.AllEmployees()
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, "查询异常："+err.Error()))
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "查询成功", Data: employees})
}

// DeleteById DELETE /om/employee/deleteById?employeeId=
func (a *AdminEmployeeController) DeleteById(c *gin.Context) {
	employeeID := c.Query("employeeId")
	if !repository.EmployeeExists(employeeID) {
		c.JSON(http.StatusOK, ResultError(500, "运维职工信息删除失败，可能是员工不存在"))
		return
	}
	repository.DeleteEmployee(employeeID)
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
	rowsAffected := repository.UpdateEmployeeStatus(employeeID, status)
	if rowsAffected == 0 {
		c.JSON(http.StatusOK, ResultError(500, "审核状态更新失败，可能是员工不存在或状态值不合法"))
		return
	}
	c.JSON(http.StatusOK, Result{Success: true, Code: 200, Msg: "审核状态更新成功", Data: true})
}
