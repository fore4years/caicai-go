package repository

import (
	"time"

	"caicai-go/conf"
	"caicai-go/model"
)

// NeighborShareRow 邻享 JOIN 查询的中间扫描结构。
type NeighborShareRow struct {
	ID             int32     `gorm:"column:id"`
	OwnerUserID    int32     `gorm:"column:owner_user_id"`
	Openid         string    `gorm:"column:openid"`
	ChargingPileID int32     `gorm:"column:charging_pile_id"`
	Status         string    `gorm:"column:status"`
	Phone          string    `gorm:"column:phone"`
	CreateTime     time.Time `gorm:"column:create_time"`
	UpdateTime     time.Time `gorm:"column:update_time"`
	Name           string    `gorm:"column:name"`
	NickName       string    `gorm:"column:nick_name"`
}

// PlacesByOwnerid 按 ownerid 查询车位列表。
func PlacesByOwnerid(ownerid string) []model.PlaceTbl {
	var places []model.PlaceTbl
	conf.Db.Where("ownerid = ?", ownerid).Find(&places)
	return places
}

// PrivatePlaceBySpacesCode 按 spaces_code 查询私桩车位。
func PrivatePlaceBySpacesCode(spacesCode string) (*model.PrivatePlaceTbl, error) {
	var p model.PrivatePlaceTbl
	if err := conf.Db.Where("spaces_code = ?", spacesCode).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ChargingGunByProductID 按 product_id 查询充电枪。
func ChargingGunByProductID(productID string) (*model.TabChargingGun, error) {
	var gun model.TabChargingGun
	if err := conf.Db.Where("product_id = ?", productID).First(&gun).Error; err != nil {
		return nil, err
	}
	return &gun, nil
}

// PrivatePlaceExistsByChargingGunID 判断充电枪是否已绑定私桩车位。
func PrivatePlaceExistsByChargingGunID(gunID int32) bool {
	var existing model.PrivatePlaceTbl
	err := conf.Db.Where("charging_gun_id = ?", gunID).First(&existing).Error
	return err == nil
}

// InsertPrivatePlace 新增私桩车位。
func InsertPrivatePlace(wkt, place, openid string, gunID int32, spacesCode string) error {
	return conf.Db.Exec(
		"INSERT INTO private_place_tbl (point, place, openid, charging_gun_id, spaces_code) VALUES (ST_GeomFromText(?),?,?,?,?)",
		wkt, place, openid, gunID, spacesCode,
	).Error
}

// PrivateChargingByProductID 按 product_id 查询私桩信息。
func PrivateChargingByProductID(productID string) (*model.PrivateChargingTbl, error) {
	var charging model.PrivateChargingTbl
	if err := conf.Db.Where("product_id = ?", productID).First(&charging).Error; err != nil {
		return nil, err
	}
	return &charging, nil
}

// PrivateChargingOpenidsByOrderID 查询订单关联的私桩桩主 openid 列表。
func PrivateChargingOpenidsByOrderID(orderID string) []string {
	var openids []string
	conf.Db.Raw(
		"SELECT private_charging_tbl.openid FROM order_tbl "+
			"LEFT JOIN private_place_tbl ON private_place_tbl.id = order_tbl.spaces_id "+
			"LEFT JOIN private_charging_tbl ON private_charging_tbl.spaces_num = private_place_tbl.spaces_code "+
			"WHERE order_tbl.orderid = ?", orderID,
	).Scan(&openids)
	return openids
}

// FreeChargingUsersByStationID 按充电站 id 查询免服务费用户列表。
func FreeChargingUsersByStationID(stationID string) []model.FreeChargingUsersTbl {
	var rows []model.FreeChargingUsersTbl
	conf.Db.Where("charging_station_id = ?", stationID).Find(&rows)
	return rows
}

// FreeChargingUserByOpenid 按 openid 查询免服务费用户。
func FreeChargingUserByOpenid(openid string) (*model.FreeChargingUsersTbl, error) {
	var u model.FreeChargingUsersTbl
	if err := conf.Db.Where("openid = ?", openid).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FreeChargingUserByOpenidStationID 按 openid + 充电站 id 查询免服务费用户。
func FreeChargingUserByOpenidStationID(openid string, stationID int32) (*model.FreeChargingUsersTbl, error) {
	var existing model.FreeChargingUsersTbl
	if err := conf.Db.Where("openid = ? AND charging_station_id = ?", openid, stationID).First(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

// UpdateFreeChargingUserStatus 更新免服务费用户状态，返回受影响行数。
func UpdateFreeChargingUserStatus(openid string, stationID interface{}, status int) int64 {
	res := conf.Db.Model(&model.FreeChargingUsersTbl{}).
		Where("openid = ? AND charging_station_id = ?", openid, stationID).
		Update("status", status)
	return res.RowsAffected
}

// DeleteFreeChargingUserByOpenid 按 openid 删除免服务费用户，返回受影响行数。
func DeleteFreeChargingUserByOpenid(openid string) int64 {
	res := conf.Db.Where("openid = ?", openid).Delete(&model.FreeChargingUsersTbl{})
	return res.RowsAffected
}

// NeighborShareUsersByOpenid 按 openid 查询邻享共享用户。
func NeighborShareUsersByOpenid(openid string) []model.NeighborShareUserTbl {
	var existing []model.NeighborShareUserTbl
	conf.Db.Where("openid = ?", openid).Find(&existing)
	return existing
}

// UpdateNeighborShareUserStatus 更新邻享共享用户状态，返回受影响行数。
func UpdateNeighborShareUserStatus(id interface{}, status int) int64 {
	res := conf.Db.Model(&model.NeighborShareUserTbl{}).Where("id = ?", id).Update("status", status)
	return res.RowsAffected
}

// NeighborShareByPidOpenid 按充电桩编号与 openid 查询邻享共享用户。
func NeighborShareByPidOpenid(pid, openid string) (*model.NeighborShareUserTbl, error) {
	var u model.NeighborShareUserTbl
	if err := conf.Db.Raw(
		"SELECT nsut.* FROM private_charging_tbl "+
			"LEFT JOIN private_place_tbl pp ON private_charging_tbl.spaces_num = pp.spaces_code "+
			"LEFT JOIN tab_charging_gun tcg ON pp.charging_gun_id = tcg.id "+
			"LEFT JOIN neighbor_share_user_tbl nsut ON private_charging_tbl.id = nsut.owner_user_id "+
			"WHERE tcg.product_id = ? AND nsut.openid = ?",
		pid, openid,
	).Scan(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// NeighborShareRows 邻享共享用户 JOIN 查询。
func NeighborShareRows(ownerID, status string) []NeighborShareRow {
	var rows []NeighborShareRow
	db := conf.Db.Table("neighbor_share_user_tbl AS neighbor").
		Select("neighbor.id, neighbor.owner_user_id, neighbor.openid, neighbor.charging_pile_id, neighbor.status, neighbor.phone, neighbor.create_time, neighbor.update_time, user_tbl.name, user_tbl.nick_name").
		Joins("LEFT JOIN user_tbl ON neighbor.openid = user_tbl.openid").
		Where("neighbor.owner_user_id = ?", ownerID)
	if status != "" {
		db = db.Where("neighbor.status = ?", status)
	}
	db.Scan(&rows)
	return rows
}
