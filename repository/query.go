package repository

import (
	"context"

	"gorm.io/gen"

	"caicai-go/model"
	"caicai-go/objects"
)

// ============================ user_tbl ============================

// UserByOpenid 按 openid 查询用户。
func UserByOpenid(ctx context.Context, openid string) (*model.UserTbl, error) {
	return objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).First()
}

// UserByPhone 按手机号查询用户。
func UserByPhone(ctx context.Context, phone string) (*model.UserTbl, error) {
	return objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Phone.Eq(phone)).First()
}

// UserByOmid 按运维 openid（omid）查询用户。
func UserByOmid(ctx context.Context, omid string) (*model.UserTbl, error) {
	return objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Omid.Eq(omid)).First()
}

// UserByYiparlOpenid 按驿帕尔 openid 查询用户。
func UserByYiparlOpenid(ctx context.Context, openid string) (*model.UserTbl, error) {
	return objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.YiparlOpenid.Eq(openid)).First()
}

// CreateUser 新增用户。
func CreateUser(ctx context.Context, u *model.UserTbl) error {
	return objects.UserTbl.WithContext(ctx).Create(u)
}

// UpdateUserByOpenid 按 openid 批量更新用户字段。
func UpdateUserByOpenid(ctx context.Context, openid string, updates map[string]interface{}) error {
	_, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).Updates(updates)
	return err
}

// UpdateUserByPhone 按手机号批量更新用户字段。
func UpdateUserByPhone(ctx context.Context, phone string, updates map[string]interface{}) error {
	_, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Phone.Eq(phone)).Updates(updates)
	return err
}

// UpdateUserPhoneByOpenid 按 openid 更新手机号。
func UpdateUserPhoneByOpenid(ctx context.Context, openid, phone string) error {
	_, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).Update(objects.UserTbl.Phone, phone)
	return err
}

// UpdateUserOmidByPhone 按手机号更新 omid。
func UpdateUserOmidByPhone(ctx context.Context, phone, omid string) error {
	_, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Phone.Eq(phone)).Update(objects.UserTbl.Omid, omid)
	return err
}

// UpdateUserYiparlByPhone 按手机号更新驿帕尔 openid。
func UpdateUserYiparlByPhone(ctx context.Context, phone, openid string) error {
	_, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Phone.Eq(phone)).Update(objects.UserTbl.YiparlOpenid, openid)
	return err
}

// UpdateUserPlateNum 按 openid 更新车牌号，返回受影响行数。
func UpdateUserPlateNum(ctx context.Context, openid, plateNum string) (int64, error) {
	info, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).Update(objects.UserTbl.PlateNum, plateNum)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}

// UpdateUserOmEnable 按 omid 更新运维审核状态，返回受影响行数。
func UpdateUserOmEnable(ctx context.Context, omid, enable string) (int64, error) {
	info, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Omid.Eq(omid)).Update(objects.UserTbl.OmEnable, enable)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}

// ============================ owner_tbl ============================

// OwnerByOwnerid 按 ownerid 查询业主。
func OwnerByOwnerid(ctx context.Context, ownerid string) (*model.OwnerTbl, error) {
	return objects.OwnerTbl.WithContext(ctx).Where(objects.OwnerTbl.Ownerid.Eq(ownerid)).First()
}

// CreateOwner 新增业主。
func CreateOwner(ctx context.Context, o *model.OwnerTbl) error {
	return objects.OwnerTbl.WithContext(ctx).Create(o)
}

// UpdateOwnerByOwnerid 按 ownerid 批量更新业主字段。
func UpdateOwnerByOwnerid(ctx context.Context, ownerid string, updates map[string]interface{}) error {
	_, err := objects.OwnerTbl.WithContext(ctx).Where(objects.OwnerTbl.Ownerid.Eq(ownerid)).Updates(updates)
	return err
}

// UpdateOwnerPhone 按 ownerid 更新手机号。
func UpdateOwnerPhone(ctx context.Context, ownerid, phone string) error {
	_, err := objects.OwnerTbl.WithContext(ctx).Where(objects.OwnerTbl.Ownerid.Eq(ownerid)).Update(objects.OwnerTbl.Phone, phone)
	return err
}

// ============================ driveruse_tbl ============================

// CreateDriverUse 新增预约次数记录。
func CreateDriverUse(ctx context.Context, d *model.DriveruseTbl) error {
	return objects.DriveruseTbl.WithContext(ctx).Create(d)
}

// ============================ order_tbl / charge_order_tbl ============================

// OrderByOrderid 按 orderid 查询订单。
func OrderByOrderid(ctx context.Context, orderid string) (*model.OrderTbl, error) {
	return objects.OrderTbl.WithContext(ctx).Where(objects.OrderTbl.Orderid.Eq(orderid)).First()
}

// FirstNotFinishedOrderByOpenid 查询用户第一条未完成订单。
func FirstNotFinishedOrderByOpenid(ctx context.Context, openid string) (*model.OrderTbl, error) {
	return objects.OrderTbl.WithContext(ctx).Where(
		objects.OrderTbl.Openid.Eq(openid),
		objects.OrderTbl.State.Neq("已完成"),
		objects.OrderTbl.State.Neq("已取消"),
	).First()
}

// NotFinishedOrdersByOpenid 查询用户所有未完成订单。
func NotFinishedOrdersByOpenid(ctx context.Context, openid string) ([]*model.OrderTbl, error) {
	return objects.OrderTbl.WithContext(ctx).Where(
		objects.OrderTbl.Openid.Eq(openid),
		objects.OrderTbl.State.Neq("已完成"),
		objects.OrderTbl.State.Neq("已取消"),
	).Find()
}

// NotFinishedChargeOrdersByOpenid 查询用户所有未完成充电订单。
func NotFinishedChargeOrdersByOpenid(ctx context.Context, openid string) ([]*model.ChargeOrderTbl, error) {
	return objects.ChargeOrderTbl.WithContext(ctx).Where(
		objects.ChargeOrderTbl.Openid.Eq(openid),
		objects.ChargeOrderTbl.State.Neq("已完成"),
	).Find()
}

// ============================ 车位 / 充电桩 / 锁 / 场地 ============================

// ParkingSpaceBySpacesCode 按车位编号查询车位。
func ParkingSpaceBySpacesCode(ctx context.Context, code string) (*model.TabParkingSpace, error) {
	return objects.TabParkingSpace.WithContext(ctx).Where(objects.TabParkingSpace.SpacesCode.Eq(code)).First()
}

// ChargeMessageByNidDirection 按 nid + direction 查询已分配充电桩编号的充电桩。
func ChargeMessageByNidDirection(ctx context.Context, nid, direction string) (*model.ChargeMessageTbl, error) {
	return objects.ChargeMessageTbl.WithContext(ctx).Where(
		objects.ChargeMessageTbl.Nid.Eq(nid),
		objects.ChargeMessageTbl.Direction.Eq(direction),
		objects.ChargeMessageTbl.Chargeid.IsNotNull(),
	).First()
}

// LockByPlaceid 按 placeid 查询车位锁。
func LockByPlaceid(ctx context.Context, placeid string) (*model.LockTbl, error) {
	return objects.LockTbl.WithContext(ctx).Where(objects.LockTbl.Placeid.Eq(placeid)).First()
}

// PlaceByPlaceid 按 placeid 查询场地。
func PlaceByPlaceid(ctx context.Context, placeid string) (*model.PlaceTbl, error) {
	return objects.PlaceTbl.WithContext(ctx).Where(objects.PlaceTbl.Placeid.Eq(placeid)).First()
}

// CreatePlace 新增场地。
func CreatePlace(ctx context.Context, p *model.PlaceTbl) error {
	return objects.PlaceTbl.WithContext(ctx).Create(p)
}

// ============================ usermessage_tbl ============================

// MarkMessageRead 将消息标记为已读，返回受影响行数。
func MarkMessageRead(ctx context.Context, id int32) (int64, error) {
	info, err := objects.UsermessageTbl.WithContext(ctx).Where(objects.UsermessageTbl.ID.Eq(id)).Update(objects.UsermessageTbl.Status, "已读")
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}

// ============================ 运维用户（om user） ============================

// omUserConds 构建运维用户查询条件：omid 与 om_enable 均非空，phone 可选模糊匹配。
func omUserConds(phone string) []gen.Condition {
	conds := []gen.Condition{
		objects.UserTbl.Omid.IsNotNull(),
		objects.UserTbl.OmEnable.IsNotNull(),
	}
	if phone != "" {
		conds = append(conds, objects.UserTbl.Phone.Like("%"+phone+"%"))
	}
	return conds
}

// CountOmUsers 统计运维用户数量。
func CountOmUsers(ctx context.Context, phone string) (int64, error) {
	return objects.UserTbl.WithContext(ctx).Where(omUserConds(phone)...).Count()
}

// ListOmUsers 分页查询运维用户。
func ListOmUsers(ctx context.Context, phone string, offset, limit int) ([]*model.UserTbl, error) {
	return objects.UserTbl.WithContext(ctx).Where(omUserConds(phone)...).Offset(offset).Limit(limit).Find()
}
