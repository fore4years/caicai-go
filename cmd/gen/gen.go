package main

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

const dsn = "root:lanlan1313!@#@tcp(81.69.46.160:3306)/parking?charset=utf8mb4&parseTime=True&loc=Local"

func connection() *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("链接失败 %w", err))
	}
	return db
}

func main() {
	cfg := gen.Config{
		OutPath:          "objects",
		Mode:             gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldWithTypeTag: true,
	}
	// 金额列 DECIMAL 映射为 decimal.Decimal，避免生成 float64 导致精度丢失。
	// 注意：gorm 拿到的实际类型名是大写 "DECIMAL"（小写 "decimal" 匹配不上，会回落成 string），
	// 故同时登记两种大小写，确保任何情况下都映射为 decimal.Decimal。
	cfg.WithDataTypeMap(map[string]func(columnType gorm.ColumnType) (dataType string){
		"decimal": func(columnType gorm.ColumnType) (dataType string) {
			return "decimal.Decimal"
		},
		"DECIMAL": func(columnType gorm.ColumnType) (dataType string) {
			return "decimal.Decimal"
		},
		// double 列（余额/冻结金额/发票金额/电表电量等计费相关）也映射为 decimal.Decimal，保证精度。
		"double": func(columnType gorm.ColumnType) (dataType string) {
			return "decimal.Decimal"
		},
		"DOUBLE": func(columnType gorm.ColumnType) (dataType string) {
			return "decimal.Decimal"
		},
		// point 列（MySQL 空间类型）映射为 model.Point（基于 orb.Point，见 model/point.go）。
		"point": func(columnType gorm.ColumnType) (dataType string) {
			return "Point"
		},
		"POINT": func(columnType gorm.ColumnType) (dataType string) {
			return "Point"
		},
		// 日期/时间类列统一映射为 *time.Time（可空指针，保留 NULL 语义，避免插入 0000-00-00）。
		"datetime": func(columnType gorm.ColumnType) (dataType string) {
			return "*time.Time"
		},
		"timestamp": func(columnType gorm.ColumnType) (dataType string) {
			return "*time.Time"
		},
		"date": func(columnType gorm.ColumnType) (dataType string) {
			return "*time.Time"
		},
	})
	cfg.WithImportPkgPath("github.com/shopspring/decimal")

	g := gen.NewGenerator(cfg)

	g.UseDB(connection())

	g.ApplyBasic(g.GenerateAllTable()...)

	g.Execute()
}
