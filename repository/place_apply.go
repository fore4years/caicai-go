package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// CountPlaceApplies 统计车位申请总数。
func CountPlaceApplies() int64 {
	var count int64
	conf.Db.Model(&model.PlaceapplyTbl{}).Count(&count)
	return count
}

// PagePlaceApplies 分页查询车位申请（applyTime 倒序）。
func PagePlaceApplies(offset, limit int) []model.PlaceapplyTbl {
	var rows []model.PlaceapplyTbl
	conf.Db.Model(&model.PlaceapplyTbl{}).Order("applyTime desc").Offset(offset).Limit(limit).Find(&rows)
	return rows
}

// PlaceApplyByID 按 id 查询车位申请。
func PlaceApplyByID(id string) (*model.PlaceapplyTbl, error) {
	var pa model.PlaceapplyTbl
	if err := conf.Db.Where("id = ?", id).First(&pa).Error; err != nil {
		return nil, err
	}
	return &pa, nil
}

// UpdatePlaceApplyState 更新车位申请状态，返回受影响行数。
func UpdatePlaceApplyState(id, state string) int64 {
	res := conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", id).Update("state", state)
	return res.RowsAffected
}

// InsertMessage 插入消息记录。
func InsertMessage(ownerid, msgtime, message string, status interface{}) error {
	return conf.Db.Exec("INSERT INTO message_tbl (ownerid, msgtime, message, status) VALUES (?,?,?,?)", ownerid, msgtime, message, status).Error
}

// MaxPlaceByCode 查询某编号前缀下的最大车位。
func MaxPlaceByCode(code string) (*model.PlaceTbl, error) {
	var p model.PlaceTbl
	if err := conf.Db.Where("placeid LIKE ?", code+"%").Order("placeid desc").Limit(1).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePlaceFields 按指定字段插入车位。
func CreatePlaceFields(place *model.PlaceTbl, fields []string) error {
	return conf.Db.Select(fields).Create(place).Error
}

// CountPlaceAppliesByState 按状态统计车位申请数。
func CountPlaceAppliesByState(state string) int64 {
	var max int64
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("state = ?", state).Count(&max)
	return max
}

// PagePlaceAppliesByState 按状态分页查询车位申请（applyTime 倒序）。
func PagePlaceAppliesByState(state string, offset, limit int) []model.PlaceapplyTbl {
	var rows []model.PlaceapplyTbl
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("state = ?", state).Order("applyTime desc").Offset(offset).Limit(limit).Find(&rows)
	return rows
}

// DeletePlaceApplyByID 删除车位申请。
func DeletePlaceApplyByID(id string) {
	conf.Db.Where("id = ?", id).Delete(&model.PlaceapplyTbl{})
}

// CountPlaceAppliesByFieldLike 按字段模糊统计车位申请数。
func CountPlaceAppliesByFieldLike(field, keyword string) int64 {
	var max int64
	conf.Db.Model(&model.PlaceapplyTbl{}).Where(field+" LIKE ?", "%"+keyword+"%").Count(&max)
	return max
}

// PagePlaceAppliesByFieldLike 按字段模糊分页查询车位申请（applyTime 倒序）。
func PagePlaceAppliesByFieldLike(field, keyword string, offset, limit int) []model.PlaceapplyTbl {
	var rows []model.PlaceapplyTbl
	conf.Db.Model(&model.PlaceapplyTbl{}).Where(field+" LIKE ?", "%"+keyword+"%").Order("applyTime desc").Offset(offset).Limit(limit).Find(&rows)
	return rows
}
