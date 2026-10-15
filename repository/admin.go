package repository

import (
	"strconv"

	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// ListHotels 分页查询酒店（id 等值、openid 等值、hotelName 模糊、creat_time 倒序）。
func ListHotels(id, openid, hotelName string, offset, limit int) (int64, []model.HotelTbl) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.HotelTbl{})
		if id != "" {
			if v, err := strconv.Atoi(id); err == nil {
				db = db.Where("id = ?", v)
			}
		}
		if openid != "" {
			db = db.Where("openid = ?", openid)
		}
		if hotelName != "" {
			db = db.Where("hotel_name LIKE ?", "%"+hotelName+"%")
		}
		return db
	}
	var total int64
	query().Count(&total)
	var rows []model.HotelTbl
	query().Order("creat_time desc").Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}

// UpdateHotelReview 更新酒店审核状态，返回受影响行数。
func UpdateHotelReview(id, review int) int64 {
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", id).Update("review", review)
	return res.RowsAffected
}

// DeleteHotelsByIDs 批量删除酒店，返回受影响行数。
func DeleteHotelsByIDs(ids []int32) int64 {
	res := conf.Db.Where("id IN ?", ids).Delete(&model.HotelTbl{})
	return res.RowsAffected
}

// ListPurchasePoles 分页查询购桩申请（apply_time 倒序）。
func ListPurchasePoles(orderNumber, openid, orderStatus string, offset, limit int) (int64, []model.PurchasePoleApplicationTbl) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.PurchasePoleApplicationTbl{})
		if orderNumber != "" {
			db = db.Where("order_number = ?", orderNumber)
		}
		if openid != "" {
			db = db.Where("openid = ?", openid)
		}
		if orderStatus != "" {
			// 处理"退款中"状态识别为"已退款"
			if orderStatus == "退款中" {
				db = db.Where("order_status = ?", "已退款")
			} else {
				db = db.Where("order_status = ?", orderStatus)
			}
		}
		return db
	}
	var total int64
	query().Count(&total)
	var rows []model.PurchasePoleApplicationTbl
	query().Order("apply_time desc").Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}

// PurchasePoleByOrderNumber 按订单号查询购桩申请。
func PurchasePoleByOrderNumber(orderNumber string) (*model.PurchasePoleApplicationTbl, error) {
	var r model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", orderNumber).First(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdatePurchasePoleStatus 更新购桩订单状态，返回受影响行数。
func UpdatePurchasePoleStatus(orderNumber, orderStatus string) int64 {
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", orderNumber).
		Update("order_status", orderStatus)
	return res.RowsAffected
}

// DeletePurchasePolesByOrderNumbers 批量删除购桩申请，返回受影响行数。
func DeletePurchasePolesByOrderNumbers(orderNumbers []string) int64 {
	res := conf.Db.Where("order_number IN ?", orderNumbers).Delete(&model.PurchasePoleApplicationTbl{})
	return res.RowsAffected
}

// ListProfitSharingConfigs 分页查询分账配置（spacesId 等值、receiverMchId 模糊、create_time 升序）。
func ListProfitSharingConfigs(spacesID, receiverMchID string, offset, limit int) (int64, []model.TabProfitSharingConfig) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.TabProfitSharingConfig{})
		if spacesID != "" {
			if id, err := strconv.Atoi(spacesID); err == nil {
				db = db.Where("spaces_id = ?", id)
			}
		}
		if receiverMchID != "" {
			db = db.Where("receiver_mch_id LIKE ?", "%"+receiverMchID+"%")
		}
		return db
	}
	var total int64
	query().Count(&total)
	var rows []model.TabProfitSharingConfig
	query().Order("create_time asc").Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}

// DeleteProfitSharingConfigsByIDs 批量删除分账配置。
func DeleteProfitSharingConfigsByIDs(ids []int32) {
	conf.Db.Where("id IN ?", ids).Delete(&model.TabProfitSharingConfig{})
}
