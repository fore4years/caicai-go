package repository

import (
	"fmt"
	"strconv"
	"time"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 本文件对应 handler/private_charging.go（PrivateChargingController）的内联 SQL，
// 统一抽离为命名函数。
// ============================================================================

// PrivateChargingPwmRow 私桩信息 + pwm 的 JOIN 扫描结构。
type PrivateChargingPwmRow struct {
	model.PrivateChargingTbl
	Pwm int32 `gorm:"column:pwm"`
}

// PrivateShareChargingRow 共享私桩 + 距离（km）的 JOIN 扫描结构。
type PrivateShareChargingRow struct {
	model.PrivateChargingTbl
	DistanceKm float64 `gorm:"column:distance_km"`
}

// InsertImage 新增图片，返回图片 id。
func InsertImage(data []byte) (int32, error) {
	img := model.TabImage{Image: data}
	if err := conf.Db.Create(&img).Error; err != nil {
		return 0, err
	}
	return img.ID, nil
}

// PrivateChargingByOpenidSpacesNum 按 openid+spaces_num 查询私桩信息。
func PrivateChargingByOpenidSpacesNum(openid, spacesNum string) (*model.PrivateChargingTbl, error) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("openid = ? AND spaces_num = ?", openid, spacesNum).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// DeleteImageByID 按 id 删除图片，返回受影响行数。
func DeleteImageByID(id int32) int64 {
	res := conf.Db.Where("id = ?", id).Delete(&model.TabImage{})
	return res.RowsAffected
}

// UpdatePrivateChargingByOpenidSpacesNum 按 openid+spaces_num 更新私桩信息，返回受影响行数。
func UpdatePrivateChargingByOpenidSpacesNum(openid, spacesNum string, bean model.PrivateChargingTbl) int64 {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("openid = ? AND spaces_num = ?", openid, spacesNum).Updates(bean)
	return res.RowsAffected
}

// InsertPrivateCharging 新增私桩信息。
func InsertPrivateCharging(bean *model.PrivateChargingTbl) error {
	return conf.Db.Create(bean).Error
}

// CreateGateway 新增网关。
func CreateGateway(gw *model.TabGateway) error {
	return conf.Db.Create(gw).Error
}

// CreateProduct 新增产品。
func CreateProduct(p *model.TabProduct) error {
	return conf.Db.Create(p).Error
}

// CreateChargingGun 新增充电枪。
func CreateChargingGun(gun *model.TabChargingGun) error {
	return conf.Db.Create(gun).Error
}

// PrivatePlaceByChargingGunID 按充电枪 id 查询私桩车位。
func PrivatePlaceByChargingGunID(gunID int32) (*model.PrivatePlaceTbl, error) {
	var p model.PrivatePlaceTbl
	if err := conf.Db.Where("charging_gun_id = ?", gunID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// NextPrivateSpacesCode 取私桩车位编号：code + 5 位递增编号。
func NextPrivateSpacesCode(code string) string {
	if code == "" {
		return ""
	}
	var maxCode string
	conf.Db.Raw("SELECT MAX(spaces_code) FROM private_place_tbl WHERE spaces_code LIKE ?", code+"%").Scan(&maxCode)
	if maxCode == "" {
		maxCode = "0"
	}
	remainder := maxCode
	if len(maxCode) >= len(code) && maxCode[:len(code)] == code {
		remainder = maxCode[len(code):]
	}
	n, err := strconv.Atoi(remainder)
	if err != nil {
		return ""
	}
	return code + fmt.Sprintf("%05d", n+1)
}

// DeletePrivateChargingByID 按 id 删除私桩信息，返回受影响行数。
func DeletePrivateChargingByID(id string) int64 {
	res := conf.Db.Where("id = ?", id).Delete(&model.PrivateChargingTbl{})
	return res.RowsAffected
}

// PrivateChargingByID 按 id 查询私桩信息。
func PrivateChargingByID(id string) (*model.PrivateChargingTbl, error) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// PrivatePlacePointBySpacesCode 按 spaces_code 查询私桩车位的坐标文本/地址/充电枪。
func PrivatePlacePointBySpacesCode(spacesCode string) (point, place string, gunID int32, err error) {
	var pp struct {
		Point         string `gorm:"column:point"`
		Place         string `gorm:"column:place"`
		ChargingGunID int32  `gorm:"column:charging_gun_id"`
	}
	if err := conf.Db.Raw(
		"SELECT ST_AsText(point) AS point, place, charging_gun_id FROM private_place_tbl WHERE spaces_code = ?", spacesCode,
	).Scan(&pp).Error; err != nil {
		return "", "", 0, err
	}
	return pp.Point, pp.Place, pp.ChargingGunID, nil
}

// PrivateChargingByOpenid 按 openid 查询私桩信息（首条）。
func PrivateChargingByOpenid(openid string) (*model.PrivateChargingTbl, error) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("openid = ?", openid).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// PrivateChargingByOpenidID 按 openid+id 查询私桩信息。
func PrivateChargingByOpenidID(openid, id string) (*model.PrivateChargingTbl, error) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("openid = ? AND id = ?", openid, id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// PrivateChargingsByOpenid 按 openid 查询私桩列表。
func PrivateChargingsByOpenid(openid string) []model.PrivateChargingTbl {
	var rows []model.PrivateChargingTbl
	conf.Db.Where("openid = ?", openid).Find(&rows)
	return rows
}

// PrivateChargingByGunProductIDOpenid 经充电枪 product_id + openid 关联查询私桩信息。
func PrivateChargingByGunProductIDOpenid(pid, openid string) (*model.PrivateChargingTbl, error) {
	var p model.PrivateChargingTbl
	err := conf.Db.Raw(
		`SELECT private_charging_tbl.* FROM private_charging_tbl
		 LEFT JOIN private_place_tbl pp ON private_charging_tbl.spaces_num = pp.spaces_code
		 LEFT JOIN tab_charging_gun tcg ON pp.charging_gun_id = tcg.id
		 WHERE tcg.product_id = ? AND private_charging_tbl.openid = ?`,
		pid, openid,
	).Scan(&p).Error
	if err != nil || p.ID == 0 {
		return nil, err
	}
	return &p, nil
}

// PrivateChargingByGunProductID 经充电枪 product_id 关联查询私桩信息。
func PrivateChargingByGunProductID(pid string) (*model.PrivateChargingTbl, error) {
	var p model.PrivateChargingTbl
	err := conf.Db.Raw(
		`SELECT private_charging_tbl.* FROM private_charging_tbl
		 LEFT JOIN private_place_tbl pp ON private_charging_tbl.spaces_num = pp.spaces_code
		 LEFT JOIN tab_charging_gun tcg ON pp.charging_gun_id = tcg.id
		 WHERE tcg.product_id = ?`,
		pid,
	).Scan(&p).Error
	if err != nil || p.ID == 0 {
		return nil, err
	}
	return &p, nil
}

// PrivateChargingsWithPwm 查询私桩信息 + pwm。
func PrivateChargingsWithPwm() []PrivateChargingPwmRow {
	var rows []PrivateChargingPwmRow
	conf.Db.Raw(
		`SELECT private_charging_tbl.*, tab_charging_gun.pwm
		 FROM private_charging_tbl
		 LEFT JOIN tab_charging_gun ON private_charging_tbl.product_id = tab_charging_gun.product_id`,
	).Scan(&rows)
	return rows
}

// UpdatePrivateChargingOneClickOpening 更新私桩一键开放状态，返回受影响行数。
func UpdatePrivateChargingOneClickOpening(id string, opening int32) int64 {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", id).Update("one_click_opening", opening)
	return res.RowsAffected
}

// UpdatePrivateChargingFields 按 id 更新私桩字段，返回受影响行数。
func UpdatePrivateChargingFields(id string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", id).Updates(updates)
	return res.RowsAffected
}

// PrivateChargingsByOneClickOpening 查询一键开放的私桩列表。
func PrivateChargingsByOneClickOpening() []model.PrivateChargingTbl {
	var rows []model.PrivateChargingTbl
	conf.Db.Where("one_click_opening = 1").Find(&rows)
	return rows
}

// PrivateShareChargingsByLocation 查询 3km 内共享私桩（按距离升序）。
func PrivateShareChargingsByLocation(latitude, longitude float64) []PrivateShareChargingRow {
	var rows []PrivateShareChargingRow
	conf.Db.Raw(
		`SELECT private_charging_tbl.id, private_charging_tbl.community_spaces_num, private_charging_tbl.share_time,
		        private_charging_tbl.fee, private_charging_tbl.product_type, private_charging_tbl.image_id,
		        private_charging_tbl.electricity_type, private_charging_tbl.product_id, private_charging_tbl.community_name,
		        private_charging_tbl.exchange_place,
		        ROUND(6371 * ACOS(COS(RADIANS(?)) * COS(RADIANS(ST_Y(private_place_tbl.point)))
		              * COS(RADIANS(ST_X(private_place_tbl.point)) - RADIANS(?)) + SIN(RADIANS(?))
		              * SIN(RADIANS(ST_Y(private_place_tbl.point)))),2) AS distance_km
		 FROM private_charging_tbl
		 LEFT JOIN private_place_tbl ON private_place_tbl.spaces_code = private_charging_tbl.spaces_num
		 WHERE private_place_tbl.point IS NOT NULL
		   AND private_charging_tbl.one_click_opening = 1
		   AND private_charging_tbl.status = 1
		 HAVING distance_km < 3
		 ORDER BY distance_km ASC`,
		latitude, longitude, latitude,
	).Scan(&rows)
	return rows
}

// PrivateShareChargingsByName 按小区名等值查询共享私桩。
func PrivateShareChargingsByName(name string) []model.PrivateChargingTbl {
	var rows []model.PrivateChargingTbl
	conf.Db.Where("community_name = ? AND one_click_opening = 1 AND status = 1", name).Find(&rows)
	return rows
}

// CountPrivateShareChargingsByName 按小区名统计共享私桩数量。
func CountPrivateShareChargingsByName(name string) int64 {
	var n int64
	conf.Db.Model(&model.PrivateChargingTbl{}).Where("community_name = ? AND one_click_opening = 1 AND status = 1", name).Count(&n)
	return n
}

// UpdateOrderNeighborRelocateTime 更新订单挪车通知时间，返回受影响行数。
func UpdateOrderNeighborRelocateTime(orderid string) int64 {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderid).Update("neighbor_relocate_time", time.Now())
	return res.RowsAffected
}

// UpdatePrivateChargingByProductID 按 product_id 更新私桩字段，返回受影响行数。
func UpdatePrivateChargingByProductID(productID string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("product_id = ?", productID).Updates(updates)
	return res.RowsAffected
}
