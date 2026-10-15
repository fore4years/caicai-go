package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// UpdateUserFieldsByPhone 按手机号更新用户字段，返回受影响行数。
func UpdateUserFieldsByPhone(phone string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.UserTbl{}).Where("phone = ?", phone).Updates(updates)
	return res.RowsAffected
}

// UpdateUserPhoneField 按 openid 更新手机号，返回受影响行数。
func UpdateUserPhoneField(openid, phone string) int64 {
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("phone", phone)
	return res.RowsAffected
}

// UpdateUserIdEntityField 按 openid 更新身份标识，返回受影响行数。
func UpdateUserIdEntityField(openid, idEntity string) int64 {
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("id_entity", idEntity)
	return res.RowsAffected
}

// UpdateUserFieldsByOpenid 按 openid 更新用户字段，返回受影响行数。
func UpdateUserFieldsByOpenid(openid string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Updates(updates)
	return res.RowsAffected
}

// UpdateUserAvatar 按 openid 更新头像，返回受影响行数。
func UpdateUserAvatar(openid string, data []byte) int64 {
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("avatar", data)
	return res.RowsAffected
}

// UpdateUserIdCardEmblem 按 openid 更新身份证国徽面，返回受影响行数。
func UpdateUserIdCardEmblem(openid string, imageID int32) int64 {
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("id_card_emblem", imageID)
	return res.RowsAffected
}

// UpdateUserIdCardAvatar 按 openid 更新身份证头像面，返回受影响行数。
func UpdateUserIdCardAvatar(openid string, imageID int32) int64 {
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Update("id_card_avatar", imageID)
	return res.RowsAffected
}
