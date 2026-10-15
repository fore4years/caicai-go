package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// HotelByID 按 id 查询酒店。
func HotelByID(id interface{}) (*model.HotelTbl, error) {
	var h model.HotelTbl
	if err := conf.Db.Where("id = ?", id).First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

// HotelsByOpenid 按 openid 查询酒店列表。
func HotelsByOpenid(openid string) []model.HotelTbl {
	var rows []model.HotelTbl
	conf.Db.Where("openid = ?", openid).Find(&rows)
	return rows
}

// UpdateHotelByID 按 id 更新酒店，返回受影响行数。
func UpdateHotelByID(id interface{}, updates interface{}) int64 {
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", id).Updates(updates)
	return res.RowsAffected
}

// UpdateHotelParkingNumber 更新酒店车位数量，返回受影响行数。
func UpdateHotelParkingNumber(id, parkingNumber int32) int64 {
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", id).Update("parking_number", parkingNumber)
	return res.RowsAffected
}

// HotelProductByProductID 按 productId 查询酒店产品。
func HotelProductByProductID(productID string) (*model.HotelProductTbl, error) {
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ?", productID).First(&hp).Error; err != nil {
		return nil, err
	}
	return &hp, nil
}

// UpdateHotelProductByProductID 按 productId 更新酒店产品，返回受影响行数。
func UpdateHotelProductByProductID(productID string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.HotelProductTbl{}).Where("productId = ?", productID).Updates(updates)
	return res.RowsAffected
}

// HotelProductByProductIDHotelID 按 productId + hotelId 查询酒店产品。
func HotelProductByProductIDHotelID(productID, hotelID string) (*model.HotelProductTbl, error) {
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ? AND hotelId = ?", productID, hotelID).First(&hp).Error; err != nil {
		return nil, err
	}
	return &hp, nil
}

// UpdateHotelProductByProductIDHotelID 更新酒店产品，返回受影响行数。
func UpdateHotelProductByProductIDHotelID(productID, hotelID string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.HotelProductTbl{}).Where("productId = ? AND hotelId = ?", productID, hotelID).Updates(updates)
	return res.RowsAffected
}

// HotelProductsByHotelID 按 hotelId 查询酒店产品列表。
func HotelProductsByHotelID(hotelID string) []model.HotelProductTbl {
	var rows []model.HotelProductTbl
	conf.Db.Where("hotelId = ?", hotelID).Find(&rows)
	return rows
}

// HotelProductExistsByHotelIDProductID 判断酒店产品是否存在。
func HotelProductExistsByHotelIDProductID(hotelID interface{}, productID string) bool {
	var existing model.HotelProductTbl
	err := conf.Db.Where("hotelId = ? AND productId = ?", hotelID, productID).First(&existing).Error
	return err == nil
}

// DeleteHotelProduct 删除酒店产品。
func DeleteHotelProduct(productID, hotelID string) {
	conf.Db.Where("productId = ? AND hotelId = ?", productID, hotelID).Delete(&model.HotelProductTbl{})
}

// DecrementHotelParkingNumber 酒店车位数量减一。
func DecrementHotelParkingNumber(hotelID string) {
	conf.Db.Exec("UPDATE hotel_tbl SET parking_number = parking_number - 1 WHERE id = ?", hotelID)
}

// HotelStaffByHotelIDOpenID 按 hotelId + openId 查询酒店员工。
func HotelStaffByHotelIDOpenID(hotelID, openID string) (*model.HotelStaffTbl, error) {
	var s model.HotelStaffTbl
	if err := conf.Db.Where("hotelId = ? AND openId = ?", hotelID, openID).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// HotelStaffsByHotelID 按 hotelId 查询酒店员工列表。
func HotelStaffsByHotelID(hotelID string) []model.HotelStaffTbl {
	var staffs []model.HotelStaffTbl
	conf.Db.Where("hotelId = ?", hotelID).Find(&staffs)
	return staffs
}

// HotelStaffsByOpenIDLockPermissions 按 openId + 权限查询酒店员工。
func HotelStaffsByOpenIDLockPermissions(openID, lockPermissions string) []model.HotelStaffTbl {
	var staffs []model.HotelStaffTbl
	conf.Db.Where("openId = ? AND lockPermissions = ?", openID, lockPermissions).Find(&staffs)
	return staffs
}

// UpdateHotelStaffLockPermissions 更新酒店员工权限，返回受影响行数。
func UpdateHotelStaffLockPermissions(hotelID, openID, lockPermissions string) int64 {
	res := conf.Db.Model(&model.HotelStaffTbl{}).Where("hotelId = ? AND openId = ?", hotelID, openID).Update("lockPermissions", lockPermissions)
	return res.RowsAffected
}

// DeleteHotelStaff 删除酒店员工，返回受影响行数。
func DeleteHotelStaff(hotelID, openID string) int64 {
	res := conf.Db.Where("hotelId = ? AND openId = ?", hotelID, openID).Delete(&model.HotelStaffTbl{})
	return res.RowsAffected
}
