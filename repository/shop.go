package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// ShopByShopID 按 shop_id 查询商铺。
func ShopByShopID(shopID string) (*model.ShopTbl, error) {
	var shop model.ShopTbl
	if err := conf.Db.Where("shop_id = ?", shopID).First(&shop).Error; err != nil {
		return nil, err
	}
	return &shop, nil
}

// PlaceByPlaceidDB 按 placeid 查询场地。
func PlaceByPlaceidDB(placeid string) (*model.PlaceTbl, error) {
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&place).Error; err != nil {
		return nil, err
	}
	return &place, nil
}

// LocksByPlaceid 按 placeid 查询车位锁列表。
func LocksByPlaceid(placeid string) []model.LockTbl {
	var locks []model.LockTbl
	conf.Db.Where("placeid = ?", placeid).Find(&locks)
	return locks
}

// LockUseByLockid 按 lockid 查询锁使用信息。
func LockUseByLockid(lockid string) (*model.LockUseTbl, error) {
	var lui model.LockUseTbl
	if err := conf.Db.Where("lockid = ?", lockid).First(&lui).Error; err != nil {
		return nil, err
	}
	return &lui, nil
}

// UserShopsByOpenid 按 openid 查询用户商铺关系。
func UserShopsByOpenid(openid string) []model.UserShopTbl {
	var userShops []model.UserShopTbl
	conf.Db.Where("open_id = ?", openid).Find(&userShops)
	return userShops
}

// ShopsByNameLike 按名称模糊查询商铺。
func ShopsByNameLike(name string) []model.ShopTbl {
	var rows []model.ShopTbl
	conf.Db.Where("name LIKE ?", "%"+name+"%").Find(&rows)
	return rows
}

// ShopStaffByShopid 按 shopId 与权限查询商铺员工。
func ShopStaffByShopid(shopID, lockPermissions string) []model.ShopStaffTbl {
	var staffs []model.ShopStaffTbl
	conf.Db.Where("shopId = ? AND lockPermissions = ?", shopID, lockPermissions).Find(&staffs)
	return staffs
}

// AllShopStaffByShopid 按 shopId 查询商铺员工。
func AllShopStaffByShopid(shopID string) []model.ShopStaffTbl {
	var staffs []model.ShopStaffTbl
	conf.Db.Where("shopId = ?", shopID).Find(&staffs)
	return staffs
}

// UpdateShopStaffLockPermissions 更新商铺员工权限，返回受影响行数。
func UpdateShopStaffLockPermissions(number, lockPermissions string) int64 {
	res := conf.Db.Model(&model.ShopStaffTbl{}).Where("number = ?", number).Update("lockPermissions", lockPermissions)
	return res.RowsAffected
}

// UpdateLockUseByLockid 更新锁使用信息，返回受影响行数。
func UpdateLockUseByLockid(lockid string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.LockUseTbl{}).Where("lockid = ?", lockid).Updates(updates)
	return res.RowsAffected
}
