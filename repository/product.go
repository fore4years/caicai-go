package repository

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// ListProducts 分页查询产品；ordered 为 true 时附加自定义排序。
func ListProducts(gatewayID, tiaojian string, offset, limit int, ordered bool) (int64, []model.TabProduct) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.TabProduct{})
		if gatewayID != "" {
			db = db.Where("gateway_id = ?", gatewayID)
		}
		if tiaojian != "" {
			db = db.Where("pid LIKE ?", "%"+tiaojian+"%")
		}
		return db
	}
	var total int64
	query().Count(&total)
	var products []model.TabProduct
	q := query()
	if ordered {
		q = q.Order("SUBSTRING(pid, 1, 3) ASC, CAST(SUBSTRING(pid, 4) AS UNSIGNED) DESC")
	}
	q.Offset(offset).Limit(limit).Find(&products)
	return total, products
}

// NextProductID 生成下一个产品编号：header + 8 位递增序号。
func NextProductID(header string) string {
	var maxPid sql.NullString
	conf.Db.Raw("SELECT MAX(pid) FROM tab_product WHERE pid LIKE ?", header+"%").Row().Scan(&maxPid)
	if !maxPid.Valid || maxPid.String == "" {
		return header + "00000001"
	}
	pid := strings.TrimPrefix(maxPid.String, header)
	n, err := strconv.Atoi(pid)
	if err != nil {
		return header + "00000001"
	}
	return header + fmt.Sprintf("%08d", n+1)
}

// UpdateProductByPid 按 pid 更新产品，返回受影响行数。
func UpdateProductByPid(pid string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.TabProduct{}).Where("pid = ?", pid).Updates(updates)
	return res.RowsAffected
}

// DeleteProductByPid 按 pid 删除产品。
func DeleteProductByPid(pid string) {
	conf.Db.Where("pid = ?", pid).Delete(&model.TabProduct{})
}

// CreateProductFields 按指定字段插入产品。
func CreateProductFields(prod *model.TabProduct, fields []string) error {
	return conf.Db.Select(fields).Create(prod).Error
}

// ProductsByImeiLike 查询产品：imei 为空返回全部，否则按 Imei 模糊查询。
func ProductsByImeiLike(imei string) []model.TabProduct {
	var rows []model.TabProduct
	if imei == "" {
		conf.Db.Find(&rows)
	} else {
		conf.Db.Where("Imei LIKE ?", "%"+imei+"%").Find(&rows)
	}
	return rows
}

// MaxSCCPid 查询 SCC 前缀产品的最大编号。
func MaxSCCPid() string {
	var latestNumber string
	conf.Db.Raw("SELECT COALESCE(MAX(pid), '') FROM tab_product WHERE pid LIKE ?", "SCC%").Row().Scan(&latestNumber)
	return latestNumber
}

// SCCPids 查询 SCC 前缀产品的编号列表。
func SCCPids() []string {
	var registeredIds []string
	conf.Db.Model(&model.TabProduct{}).Where("pid LIKE ?", "SCC%").Pluck("pid", &registeredIds)
	return registeredIds
}

// NotUseProducts 查询未被充电枪/车位锁使用且非二轮车的产品。
func NotUseProducts() []model.TabProduct {
	var rows []model.TabProduct
	conf.Db.Raw(`SELECT tab_product.* FROM tab_product
		LEFT JOIN tab_charging_gun ON tab_charging_gun.product_id = tab_product.pid
		LEFT JOIN lock_tbl ON lock_tbl.product_id = tab_product.pid
		WHERE tab_charging_gun.product_id IS NULL AND lock_tbl.product_id IS NULL AND tab_product.is_twicecar = 0`).Scan(&rows)
	return rows
}

// TwiceCarProducts 查询二轮车充电站产品。
func TwiceCarProducts() []model.TabProduct {
	var rows []model.TabProduct
	conf.Db.Raw(`SELECT tab_product.* FROM tab_product WHERE tab_product.is_twicecar = 1`).Scan(&rows)
	return rows
}

// ProductBySpacesID 车位 → 充电枪 → 产品。
func ProductBySpacesID(spacesID string) (*model.TabProduct, error) {
	var prod model.TabProduct
	err := conf.Db.Raw(`SELECT tab_product.* FROM tab_parking_spaces
		LEFT JOIN tab_charging_gun ON tab_parking_spaces.charging_gun_id = tab_charging_gun.id
		LEFT JOIN tab_product ON tab_charging_gun.product_id = tab_product.pid
		WHERE tab_parking_spaces.id = ?`, spacesID).Scan(&prod).Error
	if err != nil {
		return nil, err
	}
	return &prod, nil
}
