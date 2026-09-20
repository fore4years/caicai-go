package user

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Model struct {
	gorm.Model
	OpenId       string          `gorm:"unique;comment:用户唯一标识"`
	OmId         string          `gorm:"unique;comment:运维微信id"`
	YiparlOpenid string          `gorm:"unique;comment:逸泊停车微信id"`
	Nickname     string          `gorm:"unique;comment:微信昵称"`
	Phone        string          `gorm:"unique;comment:电话号码"`
	Province     string          `gorm:"comment:微信所在省"`
	City         string          `gorm:"comment:微信所在市"`
	Integral     string          `gorm:"comment:用户积分"`
	FreeTime     string          `gorm:"comment:免费停车时间"`
	PlateNum     string          `gorm:"comment:用户首选车牌"`
	IdNumber     string          `gorm:"comment:用户身份证号"`
	Name         string          `gorm:"comment:用户姓名"`
	Avatar       string          `gorm:"comment:头像"`
	IdCarEmblem  int             `gorm:"comment:身份证国徽面image_id"`
	IdCardAvatar int             `gorm:"comment:身份证头像面image_id"`
	Balans       decimal.Decimal `gorm:"comment:用户余额"`
	FreezeBalans decimal.Decimal `gorm:"comment:用户冻结金额"`
	IdEntity     string          `gorm:`
}

func (u Model) TableName() string {
	return "user_tbl"
}
