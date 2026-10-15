package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// ChargeByPlaceid 按 placeid 查询充电桩。
func ChargeByPlaceid(placeid string) (*model.ChargeTbl, error) {
	var c model.ChargeTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// PlaceByPlaceidAndOwnerid 按 placeid+ownerid 查询车位。
func PlaceByPlaceidAndOwnerid(placeid, ownerid string) (*model.PlaceTbl, error) {
	var p model.PlaceTbl
	if err := conf.Db.Where("placeid = ? AND ownerid = ?", placeid, ownerid).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ChargeMessageByNidDirectionNotNull 按 nid+direction 查询已分配编号的充电桩。
func ChargeMessageByNidDirectionNotNull(nid, direction string) (*model.ChargeMessageTbl, error) {
	var cm model.ChargeMessageTbl
	if err := conf.Db.Where("nid = ? AND direction = ? AND chargeid IS NOT NULL", nid, direction).First(&cm).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

// ChargeMessageByPlaceid 按 placeid 查询充电桩消息。
func ChargeMessageByPlaceid(placeid string) (*model.ChargeMessageTbl, error) {
	var cm model.ChargeMessageTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&cm).Error; err != nil {
		return nil, err
	}
	return &cm, nil
}

// MaxChargeid 查询最大充电桩编号。
func MaxChargeid() string {
	var maxChargeID string
	conf.Db.Raw("SELECT MAX(chargeid) FROM charge_message_tbl WHERE LENGTH(chargeid) = 10").Row().Scan(&maxChargeID)
	return maxChargeID
}

// InsertChargeMessage 插入充电桩消息，返回受影响行数。
func InsertChargeMessage(chargeid, placeid, nid, direction, ammeter string) int64 {
	res := conf.Db.Exec(
		"INSERT INTO charge_message_tbl (chargeid, placeid, nid, direction, ammeter) VALUES (?, ?, ?, ?, ?)",
		chargeid, placeid, nid, direction, ammeter,
	)
	return res.RowsAffected
}
