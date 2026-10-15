package repository

import (
	"gorm.io/gorm"

	"caicai-go/conf"
)

// 通用 CRUD 辅助：封装按主键 id 的简单增删改查，返回 *gorm.DB 以便调用方读取 Error/RowsAffected。

// FindAll 查询整表（无过滤），结果写入 dest（切片指针）。
func FindAll(dest interface{}) *gorm.DB {
	return conf.Db.Find(dest)
}

// Count 统计模型表记录数。
func Count(model interface{}, count *int64) *gorm.DB {
	return conf.Db.Model(model).Count(count)
}

// FindByID 按主键 id 查询单条记录，结果写入 dest。
func FindByID(dest interface{}, id interface{}) *gorm.DB {
	return conf.Db.Where("id = ?", id).First(dest)
}

// Create 插入记录。
func Create(value interface{}) *gorm.DB {
	return conf.Db.Create(value)
}

// CreateFields 按指定字段插入记录。
func CreateFields(value interface{}, fields []string) error {
	return conf.Db.Select(fields).Create(value).Error
}

// UpdateByID 按主键 id 更新记录（Updates，非零字段）。
func UpdateByID(model interface{}, id interface{}, updates interface{}) *gorm.DB {
	return conf.Db.Model(model).Where("id = ?", id).Updates(updates)
}

// DeleteByID 按主键 id 删除记录。
func DeleteByID(model interface{}, id interface{}) *gorm.DB {
	return conf.Db.Where("id = ?", id).Delete(model)
}

// Page 分页查询（可选排序），结果写入 dest。
func Page(model interface{}, order string, offset, limit int, dest interface{}) *gorm.DB {
	q := conf.Db.Model(model)
	if order != "" {
		q = q.Order(order)
	}
	return q.Offset(offset).Limit(limit).Find(dest)
}
