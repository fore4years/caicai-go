package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret 与 Java JwtUtil 的 HMAC256 密钥一致。
const JWTSecret = "www.caicaiparking.com"

// ParseToken 校验并解析 JWT，返回 phone（aud）、openid、name（username）。
// 对齐 Java JwtUtil.getUser / checkSign。
func ParseToken(tokenStr string) (phone, openid, name string, err error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", "", errors.New("token 无效或已过期")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", "", errors.New("invalid claims")
	}
	if aud, ok := claims["aud"].([]interface{}); ok && len(aud) > 0 {
		if p, ok := aud[0].(string); ok {
			phone = p
		}
	}
	openid, _ = claims["openid"].(string)
	name, _ = claims["username"].(string)
	return phone, openid, name, nil
}

// excludedPrefixes 白名单前缀，对齐 Java WebAppConfigurer 中 LoginInterceptor 的 excludePathPatterns。
// Java 使用 Ant 风格匹配（/** 匹配任意层），这里以「前缀匹配」近似。
var excludedPrefixes = []string{
	// swagger / 静态资源
	"/swagger-resources/", "/webjars/", "/v2/", "/swagger-ui/", "/v3/",
	"/static/", "/error", "/page/", "/js/", "/css/", "/img/", "/fonts/", "/agreement/",
	// 微信登录/注册
	"/wechat/", "/doc",
	// 系统后台 / 运维
	"/system/", "/om/employee/",
	// 小程序首页（/wx 前缀，含 /wx 本身）
	"/wx",
	// 微信扫码进入放行
	"/receptacle/getReceptacleAll",
	"/privateCharging/getById/", "/privateCharging/getAll", "/privateCharging/updateByProductId",
	"/ChargeStation/getChargeStation",
	"/spaces/getByPidAndDirection", "/price/getByTime", "/priceTime/getAll",
	// 后台管理系统放行
	"/orderElectronic/getElectronicByPrivateOrderidAndPrivateUser",
	"/privatePlace/getByOrder/", "/orderElectronic/getElectronicByOrderid", "/orderTwice/getByOpenid/",
	// 小程序放行
	"/image/getById", "/spaces/getByPoint", "/spaces/getAll", "/spaces/getById", "/spaces/setEnableONid",
	"/product/getNidAndGatewayByPid",
	// 支付回调
	"/pay/payCallback",
	// 根据 openid 查用户
	"/user/getUserInfoByOpenId/",
	// 账户详情
	"/account/private/getByOrderId/", "/account/public/getByOrderId/",
	// 设备数据 / 外部 API（走其它鉴权）
	"/api/device/", "/external/",
}

// isExcluded 判断路径是否在白名单内。
func isExcluded(path string) bool {
	for _, p := range excludedPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// Auth 全局鉴权中间件，对齐 Java WebAppConfigurer 中 LoginInterceptor：
// 拦截 /**，白名单（excludedPrefixes）放行，其余要求携带有效 JWT。
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isExcluded(c.Request.URL.Path) {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		phone, openid, name, err := ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{
				"success": false,
				"code":    401,
				"msg":     "未登录",
			})
			return
		}
		c.Set("phone", phone)
		c.Set("openid", openid)
		c.Set("name", name)
		c.Next()
	}
}
