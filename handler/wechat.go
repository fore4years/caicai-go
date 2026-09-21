package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
)

const (
	// 运维小程序（对齐 AppConstant.OM_APP_ID/OM_SECRET）
	omAppID     = "wxfb8c0a1d508363e3"
	omAppSecret = "19e79b4dbc0699ef85eb9e037995d60e"

	// 逸泊停车小程序（对齐 AppConstant.YIPARL_APP_ID/YIPARL_SECRET）
	yiparlAppID     = "wxb0c20064f6756257"
	yiparlAppSecret = "b368e88a1c984fd0d0000afafe09c997"

	// JWT 密钥（对齐 JwtUtil 的 HMAC256 密钥）
	jwtSecret = "www.caicaiparking.com"
)

// TokenBean 对应 Java model.sys.TokenBean。
type TokenBean struct {
	Token     string `json:"token"`
	LongToken string `json:"longToken"`
}

// registerReq 对应 Java model.sys.RegisterDTO。
// 注意：phone 字段实际承载「手机号授权 code」，Java 里命名有误导。
type registerReq struct {
	Phone    string `json:"phone"`
	Code     string `json:"code"`
	IDEntity string `json:"idEntity"`
}

// WeChatController 对应 Java WeChatController（/wechat）。
type WeChatController struct{}

// openidByApp 通过登录 code 获取 openid（按 appid/secret）。
func openidByApp(code, appid, secret string) (string, bool) {
	fanhui, err := wxGet(fmt.Sprintf(jscode2sessionURL, appid, secret, code))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		return "", false
	}
	return getOpenid(fanhui)
}

// getPhoneNumberByCode 通过手机号授权 code 获取手机号（getuserphonenumber）。
func getPhoneNumberByCode(code, token string) (string, bool) {
	msg, err := wxPostJSON(fmt.Sprintf(phoneNumberURL, token), fmt.Sprintf(`{"code":"%s"}`, code))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		return "", false
	}
	if errcode, ok := msg["errcode"].(float64); ok && errcode == 0 {
		if phoneInfo, ok := msg["phone_info"].(map[string]interface{}); ok {
			if phone, ok := phoneInfo["phoneNumber"].(string); ok {
				return phone, true
			}
		}
	}
	return "", false
}

// access_token 缓存（appid -> token）
var accessTokenCache sync.Map

// getAccessToken 获取小程序 access_token（带缓存，对齐 WeChatServiceImpl 的 uniAppToken）。
func getAccessToken(appid, secret string) string {
	if v, ok := accessTokenCache.Load(appid); ok {
		return v.(string)
	}
	body, err := wxGet(fmt.Sprintf(accessTokenURL, appid, secret))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		return ""
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		logger.Mylog.Err(err).Caller().Send()
		return ""
	}
	if resp.AccessToken != "" {
		accessTokenCache.Store(appid, resp.AccessToken)
	}
	return resp.AccessToken
}

// jwtSign 生成 JWT（HS256），对齐 Java JwtUtil.sign。
func jwtSign(phone, openid, name string, expire time.Duration) string {
	claims := jwt.MapClaims{
		"aud":      []string{phone},
		"openid":   openid,
		"username": name,
		"exp":      time.Now().Add(expire).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		return ""
	}
	return signed
}

// getToken 生成短 token（60 分钟）与长 token（7 天）。
func getToken(phone, openid, name string) TokenBean {
	return TokenBean{
		Token:     jwtSign(phone, openid, name, 60*time.Minute),
		LongToken: jwtSign(phone, openid, name, 7*24*time.Hour),
	}
}

// checkAndSaveDriverUse 预约次数表不存在则写入（对应 DriverUseServiceImpl.save）。
func checkAndSaveDriverUse(ctx *gin.Context, openid string) {
	if _, err := objects.DriveruseTbl.WithContext(ctx.Request.Context()).Where(objects.DriveruseTbl.Openid.Eq(openid)).First(); err != nil {
		saveDriverUse(ctx.Request.Context(), openid)
	}
}

// addUser 对应 Java UserServiceImpl.add。
func addUser(ctx *gin.Context, openid, phone, idEntity string) (TokenBean, error) {
	user, err := objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Phone.Eq(phone)).First()
	if err == nil {
		_, _ = objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Phone.Eq(phone)).Updates(map[string]interface{}{
			"openid":    openid,
			"id_entity": idEntity,
		})
		return getToken(user.Phone, openid, user.Name), nil
	}

	newUser := &model.UserTbl{
		NickName:     "微信用户",
		Openid:       openid,
		Phone:        phone,
		IDEntity:     idEntity,
		Balans:       0,
		FreezeBalans: 0,
	}
	if err := objects.UserTbl.WithContext(ctx.Request.Context()).Create(newUser); err != nil {
		return TokenBean{}, fmt.Errorf("用户注册失败")
	}
	return getToken(phone, openid, "微信用户"), nil
}

// addOMUser 对应 Java UserServiceImpl.addOMUser。
func addOMUser(ctx *gin.Context, openid, phone string) (TokenBean, error) {
	user, err := objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Phone.Eq(phone)).First()
	if err == nil {
		switch user.OmEnable {
		case "1":
			_, _ = objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Phone.Eq(phone)).Update(objects.UserTbl.Omid, openid)
			user.Omid = openid
			return getToken(user.Phone, user.Omid, user.Name), nil
		case "0":
			return TokenBean{}, fmt.Errorf("加急审核中,请耐心等待")
		case "-1":
			return TokenBean{}, fmt.Errorf("审核未通过,请咨询客服")
		default:
			_, _ = objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Phone.Eq(phone)).Updates(map[string]interface{}{
				"om_enable": "0",
				"omid":      openid,
			})
			return TokenBean{}, fmt.Errorf("已为您注册运维,请耐心等待审核")
		}
	}

	newUser := &model.UserTbl{
		Phone:    phone,
		Omid:     openid,
		OmEnable: "0",
	}
	if err := objects.UserTbl.WithContext(ctx.Request.Context()).Create(newUser); err != nil {
		return TokenBean{}, fmt.Errorf("用户注册失败")
	}
	return TokenBean{}, fmt.Errorf("注册成功，请耐心等待审核")
}

// addYiparlUser 对应 Java UserServiceImpl.addYiparlUser。
func addYiparlUser(ctx *gin.Context, openid, phone string) (TokenBean, error) {
	user, err := objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Phone.Eq(phone)).First()
	if err == nil {
		_, _ = objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Phone.Eq(phone)).Update(objects.UserTbl.YiparlOpenid, openid)
		user.YiparlOpenid = openid
		return getToken(user.Phone, user.YiparlOpenid, user.Name), nil
	}

	newUser := &model.UserTbl{
		Phone:        phone,
		YiparlOpenid: openid,
	}
	if err := objects.UserTbl.WithContext(ctx.Request.Context()).Create(newUser); err != nil {
		return TokenBean{}, fmt.Errorf("用户注册失败")
	}
	return getToken(phone, openid, ""), nil
}

// GetPhone 获取手机号（对应 Java WeChatController.getPhone）。
func (c *WeChatController) GetPhone(ctx *gin.Context) {
	code := ctx.Query("code")
	phone, ok := getPhoneNumberByCode(code, getAccessToken(appid, appSecret))
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取手机号失败"})
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(phone))
}

// Register 用户注册（对应 Java WeChatController.register）。
func (c *WeChatController) Register(ctx *gin.Context) {
	var req registerReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "参数错误"})
		return
	}

	openid, ok := openidByApp(req.Code, appid, appSecret)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取openid失败"})
		return
	}
	phoneNumber, ok := getPhoneNumberByCode(req.Phone, getAccessToken(appid, appSecret))
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取手机号失败"})
		return
	}

	checkAndSaveDriverUse(ctx, openid)
	token, err := addUser(ctx, openid, phoneNumber, req.IDEntity)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(token))
}

// Login 用户登录（对应 Java WeChatController.login）。
func (c *WeChatController) Login(ctx *gin.Context) {
	code := ctx.Query("code")
	openid, ok := openidByApp(code, appid, appSecret)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取openid失败"})
		return
	}
	user, err := objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Openid.Eq(openid)).First()
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "用户不存在"})
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(getToken(user.Phone, user.Openid, user.Name)))
}

// OmRegister 运维用户注册（对应 Java WeChatController.omRegister）。
func (c *WeChatController) OmRegister(ctx *gin.Context) {
	var req registerReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "参数错误"})
		return
	}

	openid, ok := openidByApp(req.Code, omAppID, omAppSecret)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取openid失败"})
		return
	}
	phoneNumber, ok := getPhoneNumberByCode(req.Phone, getAccessToken(omAppID, omAppSecret))
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取手机号失败"})
		return
	}

	token, err := addOMUser(ctx, openid, phoneNumber)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(token))
}

// OmLogin 运维用户登录（对应 Java WeChatController.omLogin）。
func (c *WeChatController) OmLogin(ctx *gin.Context) {
	code := ctx.Query("code")
	openid, ok := openidByApp(code, omAppID, omAppSecret)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取openid失败"})
		return
	}
	user, err := objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.Omid.Eq(openid)).First()
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "用户不存在"})
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(getToken(user.Phone, user.Omid, user.Name)))
}

// YiParLRegister 逸泊停车用户注册（对应 Java WeChatController.yiParLRegister）。
func (c *WeChatController) YiParLRegister(ctx *gin.Context) {
	var req registerReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "参数错误"})
		return
	}

	openid, ok := openidByApp(req.Code, yiparlAppID, yiparlAppSecret)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取openid失败"})
		return
	}
	phoneNumber, ok := getPhoneNumberByCode(req.Phone, getAccessToken(yiparlAppID, yiparlAppSecret))
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取手机号失败"})
		return
	}

	checkAndSaveDriverUse(ctx, openid)
	token, err := addYiparlUser(ctx, openid, phoneNumber)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(token))
}

// YiParLLogin 逸泊停车用户登录（对应 Java WeChatController.yiParLLogin）。
func (c *WeChatController) YiParLLogin(ctx *gin.Context) {
	code := ctx.Query("code")
	openid, ok := openidByApp(code, yiparlAppID, yiparlAppSecret)
	if !ok {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "获取openid失败"})
		return
	}
	user, err := objects.UserTbl.WithContext(ctx.Request.Context()).Where(objects.UserTbl.YiparlOpenid.Eq(openid)).First()
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Success: false, Code: 1, Msg: "用户不存在"})
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(getToken(user.Phone, user.YiparlOpenid, user.Name)))
}
