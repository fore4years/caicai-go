package repository

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// ListOrders 分页查询订单（订单管理列表，reserve_time 倒序）。
func ListOrders(orderid, openid, startDate, endDate, state, consumptionType, address string, offset, limit int) (int64, []model.OrderTbl) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.OrderTbl{})
		if orderid != "" {
			db = db.Where("orderid = ?", orderid)
		}
		if openid != "" {
			db = db.Where("openid = ?", openid)
		}
		if startDate != "" {
			db = db.Where("begin_time >= ?", startDate)
		}
		if endDate != "" {
			db = db.Where("over_time <= ?", endDate+" 23:59:59")
		}
		if state != "" {
			db = db.Where("state = ?", state)
		}
		if consumptionType != "" {
			db = db.Where("consumption_type = ?", consumptionType)
		}
		if address != "" {
			db = db.Where("consumption_location LIKE ?", "%"+address+"%")
		}
		return db
	}
	var total int64
	query().Count(&total)
	var orders []model.OrderTbl
	query().Order("reserve_time desc").Offset(offset).Limit(limit).Find(&orders)
	return total, orders
}

// ListOrdersByPid 分页查询某 pid 下已完成订单（充电记录，begin_time 倒序）。
func ListOrdersByPid(pid, startDate, endDate string, offset, limit int) (int64, []model.OrderTbl) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.OrderTbl{}).Where("pid = ?", pid).Where("state = ?", "已完成")
		if startDate != "" {
			db = db.Where("over_time >= ?", startDate)
		}
		if endDate != "" {
			db = db.Where("over_time <= ?", endDate)
		}
		return db
	}
	var total int64
	query().Count(&total)
	var orders []model.OrderTbl
	query().Order("begin_time desc").Offset(offset).Limit(limit).Find(&orders)
	return total, orders
}

// OrderStatistics 汇总已完成订单的订单数/充电度数/费用（全部 + 今日）。
func OrderStatistics(pid, startDate, endDate string) (totalOrders, todayOrders int64, totalDegree float64, totalFee decimal.Decimal, todayDegree float64, todayFee decimal.Decimal) {
	statsQuery := func(today bool) *gorm.DB {
		db := conf.Db.Model(&model.OrderTbl{}).Where("state = ?", "已完成")
		if pid != "" {
			db = db.Where("pid = ?", pid)
		}
		if today {
			t := time.Now().Format("2006-01-02")
			db = db.Where("begin_time >= ?", t+" 00:00:00").Where("over_time <= ?", t+" 23:59:59")
		} else {
			if startDate != "" {
				db = db.Where("begin_time >= ?", startDate)
			}
			if endDate != "" {
				db = db.Where("over_time <= ?", endDate+" 23:59:59")
			}
		}
		return db
	}

	type agg struct {
		Degree float64         `gorm:"column:d"`
		Fee    decimal.Decimal `gorm:"column:f"`
	}

	statsQuery(false).Count(&totalOrders)
	statsQuery(true).Count(&todayOrders)

	var totalAgg, todayAgg agg
	statsQuery(false).Select("COALESCE(SUM(charging_degree),0) AS d, COALESCE(SUM(total_price),0) AS f").Scan(&totalAgg)
	statsQuery(true).Select("COALESCE(SUM(charging_degree),0) AS d, COALESCE(SUM(total_price),0) AS f").Scan(&todayAgg)

	return totalOrders, todayOrders, totalAgg.Degree, totalAgg.Fee, todayAgg.Degree, todayAgg.Fee
}

// CountOrdersByFilters 按 pid/时间/状态统计订单数（导出用）。
func CountOrdersByFilters(pid, startDate, endDate, state string) int64 {
	db := conf.Db.Model(&model.OrderTbl{})
	if pid != "" {
		db = db.Where("pid = ?", pid)
	}
	if startDate != "" {
		db = db.Where("begin_time >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("over_time <= ?", endDate+" 23:59:59")
	}
	if state != "" {
		db = db.Where("state = ?", state)
	}
	var count int64
	db.Count(&count)
	return count
}

// DeleteOrdersByIDs 批量删除订单。
func DeleteOrdersByIDs(orderIds []string) error {
	return conf.Db.Where("orderid IN ?", orderIds).Delete(&model.OrderTbl{}).Error
}

// OrderSum 汇总某列（charging_degree / basic_consumption），返回字符串；无数据返回 nil。
func OrderSum(pid, startDate, endDate, column string, today bool) *string {
	q := "SELECT SUM(" + column + ") FROM order_tbl WHERE pid = ? AND state = '已完成'"
	args := []interface{}{pid}

	if today {
		t := time.Now()
		day := t.Format("2006-01-02")
		next := t.AddDate(0, 0, 1).Format("2006-01-02")
		q += " AND over_time >= ? AND over_time <= ?"
		args = append(args, day+" 00:00:00", next+" 00:00:00")
	} else {
		if startDate != "" {
			q += " AND over_time >= ?"
			args = append(args, startDate)
		}
		if endDate != "" {
			q += " AND over_time <= ?"
			args = append(args, endDate)
		}
	}

	var ns sql.NullString
	if err := conf.Db.Raw(q, args...).Row().Scan(&ns); err != nil {
		return nil
	}
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}
