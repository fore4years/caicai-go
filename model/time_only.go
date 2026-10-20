package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TimeOnly 表示纯时段（如 00:00-08:00），强制 GORM 映射为 MySQL TIME 类型（而非 datetime）。
// 包装 time.Time，字符串格式统一为 "15:04:05"。
type TimeOnly struct {
	time.Time
}

// GormDBDataType 强制数据库列类型为 TIME（AutoMigrate 时生效）。
func (TimeOnly) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return "TIME"
}

// Value 实现 driver.Valuer：以 "15:04:05" 写入数据库。
func (t TimeOnly) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Format("15:04:05"), nil
}

// Scan 实现 sql.Scanner：从 "15:04:05"（或含日期的字符串）解析。
func (t *TimeOnly) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case []byte:
		return t.parse(string(v))
	case string:
		return t.parse(v)
	default:
		return fmt.Errorf("unsupported TIME value type: %T", value)
	}
}

func (t *TimeOnly) parse(s string) error {
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	for _, layout := range []string{"15:04:05", "2006-01-02 15:04:05", time.RFC3339} {
		if tt, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			t.Time = tt
			return nil
		}
	}
	return fmt.Errorf("cannot parse TIME value: %q", s)
}

// MarshalJSON 以 "HH:MM:SS" 序列化。
func (t TimeOnly) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + t.Format("15:04:05") + `"`), nil
}

// UnmarshalJSON 解析 "HH:MM:SS"。
func (t *TimeOnly) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	return t.parse(s)
}
