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
	g := gen.NewGenerator(gen.Config{
		OutPath: "./objects",
		Mode:    gen.WithDefaultQuery | gen.WithQueryInterface,
	})

	g.UseDB(connection())

	g.ApplyBasic(g.GenerateAllTable()...)

	g.Execute()
}
