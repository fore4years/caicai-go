package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
)

const (
	appid     = "wx4fc1474ea10f5d5e"
	appSecret = "23109c5078014fece6dd81659659f0ce"

	// 运维小程序 appID / appSecret
	appidOperation     = "wxfb8c0a1d508363e3"
	appSecretOperation = "d4a948407507dc2ac5da079b57dc2b48"

	// 微信接口地址
	jscode2sessionURL = "https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code"
	accessTokenURL    = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s"
	phoneNumberURL    = "https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s"
)

// uniAppToken 微信小程序 access_token 缓存（对齐 WeChatServiceImpl 的 uniAppToken 字段）。
// Java 侧用 Timer 每 110 分钟定时刷新，这里简化为「为空则拉取」。
var (
	uniAppTokenMu sync.RWMutex
	uniAppToken   string
)

type UserController struct{}

// userLoginReq 用户登录请求参数（对齐 Java domain user）
type userLoginReq struct {
	NickName  string `form:"nickName" json:"nickName"`
	Gender    string `form:"gender" json:"gender"`
	AvatarURL string `form:"avatarUrl" json:"avatarUrl"`
	Province  string `form:"province" json:"province"`
	City      string `form:"city" json:"city"`
	Phone     string `form:"phone" json:"phone"`
	Integral  string `form:"integral" json:"integral"`
	FreeTime  string `form:"freeTime" json:"freeTime"`
	IDNumber  string `form:"id_number" json:"id_number"`
	PlateNum  string `form:"plate_num" json:"plate_num"`
	Name      string `form:"name" json:"name"`
}

// ownerLoginReq 业主登录请求参数（对齐 Java domain owner）
type ownerLoginReq struct {
	NickName  string `form:"nickName" json:"nickName"`
	Gender    string `form:"gender" json:"gender"`
	AvatarURL string `form:"avatarUrl" json:"avatarUrl"`
	Province  string `form:"province" json:"province"`
	City      string `form:"city" json:"city"`
	Phone     string `form:"phone" json:"phone"`
	WxNumber  string `form:"wxNumber" json:"wxNumber"`
	Email     string `form:"email" json:"email"`
	Account   string `form:"account" json:"account"`
}

// WxLogin 用户登录（对应 Java loginWxSerImpl.wxDenglu）
func (u *UserController) WxLogin(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	var req userLoginReq
	_ = c.ShouldBind(&req)

	// 1. 向微信服务器请求 openid
	fanhui, err := wxGet(fmt.Sprintf(jscode2sessionURL, appid, appSecret, code))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, gin.H{"jieguo": false})
		return
	}

	openid, ok := getOpenid(fanhui)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"jieguo": false})
		return
	}

	// 2. 判断是否已注册
	zhuce := 0
	user, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 未注册：写入 user_tbl + driveruse_tbl
		user = &model.UserTbl{
			Openid:   openid,
			NickName: req.NickName,
			Province: req.Province,
			City:     req.City,
			Phone:    req.Phone,
			Integral: req.Integral,
			FreeTime: req.FreeTime,
			IDNumber: req.IDNumber,
			PlateNum: req.PlateNum,
			Name:     req.Name,
		}
		if req.AvatarURL != "" {
			user.Avatar = []byte(req.AvatarURL)
		}
		if err := objects.UserTbl.WithContext(ctx).Create(user); err != nil {
			logger.Mylog.Err(err).Caller().Send()
			c.JSON(http.StatusOK, gin.H{"jieguo": false})
			return
		}
		saveDriverUse(ctx, openid)
	} else if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, gin.H{"jieguo": false})
		return
	} else {
		// 已注册：更新微信资料，再重新查询
		zhuce = 1
		_, _ = objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).Updates(map[string]interface{}{
			"nick_name": req.NickName,
			"province":  req.Province,
			"city":      req.City,
			"integral":  req.Integral,
			"free_time": req.FreeTime,
			"id_number": req.IDNumber,
			"plate_num": req.PlateNum,
			"name":      req.Name,
		})
		user, _ = objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).First()
	}

	c.JSON(http.StatusOK, gin.H{
		"jieguo": true,
		"openid": openid,
		"zhuce":  zhuce,
		"user":   user,
	})
}

// LoginOwner 业主登录（对应 Java loginWxSerImpl.login_owner）
func (u *UserController) LoginOwner(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	var req ownerLoginReq
	_ = c.ShouldBind(&req)

	fanhui, err := wxGet(fmt.Sprintf(jscode2sessionURL, appid, appSecret, code))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, gin.H{"jieguo": false})
		return
	}

	ownerid, ok := getOpenid(fanhui)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"jieguo": false})
		return
	}

	zhuce := 0
	owner, err := objects.OwnerTbl.WithContext(ctx).Where(objects.OwnerTbl.Ownerid.Eq(ownerid)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		owner = &model.OwnerTbl{
			Ownerid:   ownerid,
			NickName:  req.NickName,
			Gender:    req.Gender,
			AvatarURL: req.AvatarURL,
			Province:  req.Province,
			City:      req.City,
			Phone:     req.Phone,
			WxNumber:  req.WxNumber,
			Email:     req.Email,
			Account:   req.Account,
		}
		if err := objects.OwnerTbl.WithContext(ctx).Create(owner); err != nil {
			logger.Mylog.Err(err).Caller().Send()
			c.JSON(http.StatusOK, gin.H{"jieguo": false})
			return
		}
	} else if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, gin.H{"jieguo": false})
		return
	} else {
		zhuce = 1
		_, _ = objects.OwnerTbl.WithContext(ctx).Where(objects.OwnerTbl.Ownerid.Eq(ownerid)).Updates(map[string]interface{}{
			"nickName":  req.NickName,
			"gender":    req.Gender,
			"avatarUrl": req.AvatarURL,
			"province":  req.Province,
			"city":      req.City,
			"wxNumber":  req.WxNumber,
			"email":     req.Email,
			"account":   req.Account,
		})
		owner, _ = objects.OwnerTbl.WithContext(ctx).Where(objects.OwnerTbl.Ownerid.Eq(ownerid)).First()
	}

	c.JSON(http.StatusOK, gin.H{
		"jieguo":  true,
		"ownerid": ownerid,
		"owner":   owner,
		"zhuce":   zhuce,
	})
}

// GetPhoneNumber 获取用户手机号（对应 Java loginWxSerImpl.getPhoneNumber）
func (u *UserController) GetPhoneNumber(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	openid := c.Query("openid")

	token := getUniAppToken()
	msg, err := wxPostJSON(fmt.Sprintf(phoneNumberURL, token), fmt.Sprintf(`{"code":"%s"}`, code))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, nil)
		return
	}

	if errcode, ok := msg["errcode"].(float64); ok && errcode == 0 {
		if phoneInfo, ok := msg["phone_info"].(map[string]interface{}); ok {
			if phone, ok := phoneInfo["phoneNumber"].(string); ok {
				_, _ = objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).Update(objects.UserTbl.Phone, phone)
				logger.Mylog.Info().Msgf("获取到手机号：%s", phone)
			}
		}
		c.JSON(http.StatusOK, msg)
		return
	}

	logger.Mylog.Info().Msgf("获取手机号失败：%v", msg)
	c.JSON(http.StatusOK, nil)
}

// GetOwnerPhoneNumber 获取业主手机号（对应 Java loginWxSerImpl.getOwnerPhoneNumber）
func (u *UserController) GetOwnerPhoneNumber(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Query("code")
	ownerid := c.Query("ownerid")

	token := getUniAppToken()
	msg, err := wxPostJSON(fmt.Sprintf(phoneNumberURL, token), fmt.Sprintf(`{"code":"%s"}`, code))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, nil)
		return
	}

	if errcode, ok := msg["errcode"].(float64); ok && errcode == 0 {
		if phoneInfo, ok := msg["phone_info"].(map[string]interface{}); ok {
			if phone, ok := phoneInfo["phoneNumber"].(string); ok {
				_, _ = objects.OwnerTbl.WithContext(ctx).Where(objects.OwnerTbl.Ownerid.Eq(ownerid)).Update(objects.OwnerTbl.Phone, phone)
				logger.Mylog.Info().Msgf("获取到手机号：%s", phone)
			}
		}
		c.JSON(http.StatusOK, msg)
		return
	}

	logger.Mylog.Info().Msgf("获取手机号失败：%v", msg)
	c.JSON(http.StatusOK, nil)
}

// Submit 提交车牌号（对应 Java loginWxSerImpl.submit）
func (u *UserController) Submit(c *gin.Context) {
	ctx := c.Request.Context()
	openid := c.Query("openid")
	plateNum := c.Query("plate_num")

	// 写入 driveruse_tbl（预约次数 3）
	if err := objects.DriveruseTbl.WithContext(ctx).Create(&model.DriveruseTbl{
		Openid:        openid,
		ReserveNumber: 3,
		EveryDay:      time.Now(),
	}); err != nil {
		logger.Mylog.Err(err).Caller().Send()
	}

	// 更新 user_tbl 车牌号
	info, err := objects.UserTbl.WithContext(ctx).Where(objects.UserTbl.Openid.Eq(openid)).Update(objects.UserTbl.PlateNum, plateNum)
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, 0)
		return
	}
	if info.RowsAffected == 0 {
		logger.Mylog.Info().Msgf("车牌号提交失败: %s %s", openid, plateNum)
	} else {
		logger.Mylog.Info().Msg("车牌号提交成功")
	}

	c.JSON(http.StatusOK, info.RowsAffected)
}

// saveDriverUse 注册时写入预约次数表（对应 Java DriverUseServiceImpl.save）
func saveDriverUse(ctx context.Context, openid string) {
	if err := objects.DriveruseTbl.WithContext(ctx).Create(&model.DriveruseTbl{
		Openid:        openid,
		ReserveNumber: 3,
		EveryDay:      time.Now(),
	}); err != nil {
		logger.Mylog.Err(err).Caller().Send()
	}
}

// getOpenid 解析微信 jscode2session 返回，取出 openid
func getOpenid(fanhui string) (string, bool) {
	var resp struct {
		Openid  string `json:"openid"`
		Errcode int    `json:"errcode"`
	}
	if err := json.Unmarshal([]byte(fanhui), &resp); err != nil {
		return "", false
	}
	return resp.Openid, resp.Openid != ""
}

// getUniAppToken 获取小程序 access_token（带进程内缓存）
func getUniAppToken() string {
	uniAppTokenMu.RLock()
	if uniAppToken != "" {
		t := uniAppToken
		uniAppTokenMu.RUnlock()
		return t
	}
	uniAppTokenMu.RUnlock()

	body, err := wxGet(fmt.Sprintf(accessTokenURL, appid, appSecret))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		return ""
	}

	var resp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		logger.Mylog.Err(err).Caller().Send()
		return ""
	}

	uniAppTokenMu.Lock()
	uniAppToken = resp.AccessToken
	uniAppTokenMu.Unlock()
	return resp.AccessToken
}

// wxGet 发送 GET 请求并返回响应体字符串
func wxGet(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET 请求失败 %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

// wxPostJSON 发送 POST JSON 请求，返回解析后的 map（对应 Java qingqiu.post）
func wxPostJSON(url, jsonBody string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST 请求失败 %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return m, nil
}
