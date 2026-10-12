package model

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/encoding/wkt"
)

// Point 表示 MySQL POINT 空间类型（经度在前，纬度在后），基于 orb.Point。
// 用于 gorm 与数据库之间的 WKB 编解码（Scan/Value）。
type Point struct {
	orb.Point
}

// Scan 实现 sql.Scanner：从数据库读取 WKB 字节并解析为点。
func (p *Point) Scan(value interface{}) error {
	if value == nil {
		p.Point = orb.Point{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		if len(v) >= 2 && v[:2] == "0x" {
			var err error
			if data, err = hex.DecodeString(v[2:]); err != nil {
				return err
			}
		} else {
			data = []byte(v)
		}
	default:
		return fmt.Errorf("unsupported POINT value type: %T", value)
	}
	if len(data) == 0 {
		p.Point = orb.Point{}
		return nil
	}
	geom, err := wkb.Unmarshal(data)
	if err != nil {
		return err
	}
	pt, ok := geom.(orb.Point)
	if !ok {
		return fmt.Errorf("expected POINT, got %T", geom)
	}
	p.Point = pt
	return nil
}

// Value 实现 driver.Valuer：将点序列化为 WKB。
func (p Point) Value() (driver.Value, error) {
	return wkb.Marshal(p.Point)
}

// Lng 返回经度（POINT 的 X 坐标）。
func (p Point) Lng() float64 {
	return p.Point.Lon()
}

// Lat 返回纬度（POINT 的 Y 坐标）。
func (p Point) Lat() float64 {
	return p.Point.Lat()
}

// NewPoint 用经纬度（lng=经度, lat=纬度）构造一个 Point。
func NewPoint(lng, lat float64) Point {
	return Point{Point: orb.Point{lng, lat}}
}

// MarshalJSON 将 Point 序列化为 [经度, 纬度] 数组。
func (p Point) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{p.Lng(), p.Lat()})
}

// String 返回 WKT 文本（如 "POINT(lng lat)"）。
func (p Point) String() string {
	return wkt.MarshalString(p.Point)
}
