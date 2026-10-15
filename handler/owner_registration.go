package handler

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/repository"
)

// YezhuController 对应 Java YezhuCon（/chezhu）。
type YezhuController struct{}

// zhuceReq 业主注册车位请求参数（对齐 Java domain.Place）。
type zhuceReq struct {
	Ownerid         string `form:"ownerid" json:"ownerid"`
	Province        string `form:"province" json:"province"`
	City            string `form:"city" json:"city"`
	District        string `form:"district" json:"district"`
	Streer          string `form:"streer" json:"streer"`
	RelatedBuilding string `form:"relatedBuilding" json:"relatedBuilding"`
	Longitude       string `form:"longitude" json:"longitude"`
	Latitude        string `form:"latitude" json:"latitude"`
	Rate            string `form:"rate" json:"rate"`
	State           string `form:"state" json:"state"`
	FixType         string `form:"fixType" json:"fixType"`
	LedID           string `form:"ledId" json:"ledId"`
}

// Zhuce 业主注册车位（对应 Java YeZhuSer.zhuce）。
func (y *YezhuController) Zhuce(c *gin.Context) {
	ctx := c.Request.Context()
	var req zhuceReq
	_ = c.ShouldBind(&req)

	cityName := req.Province + req.City + req.District
	no := getCityNo(cityName)
	if no == "" {
		no = "888888"
	}

	// selectIdById: 取该城市编号下最大的 placeid
	placeids, err := repository.PlaceidsLike(no + "%")
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.String(http.StatusOK, "注册失败，ser返回可能为null")
		return
	}
	if len(placeids) == 0 {
		no = no + "00001"
	} else {
		xianzaiID := placeids[0]
		if len(xianzaiID) > 6 {
			n, _ := strconv.Atoi(xianzaiID[6:])
			no = no + fmt.Sprintf("%05d", n+1)
		} else {
			no = no + "00001"
		}
	}

	place := model.PlaceTbl{
		Placeid:         no,
		Ownerid:         req.Ownerid,
		Province:        req.Province,
		City:            req.City,
		District:        req.District,
		Streer:          req.Streer,
		RelatedBuilding: req.RelatedBuilding,
		Longitude:       req.Longitude,
		Latitude:        req.Latitude,
		Rate:            req.Rate,
		State:           req.State,
		FixType:         req.FixType,
		LedID:           req.LedID,
	}
	if err := repository.CreatePlace(ctx, &place); err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.String(http.StatusOK, "注册失败，ser返回可能为null")
		return
	}
	c.String(http.StatusOK, "注册成功")
}

// GetLockList 获取所有车位锁（对应 Java YeZhuSer.getLockList）。
func (y *YezhuController) GetLockList(c *gin.Context) {
	list, err := repository.AllLocks()
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, []map[string]string{})
		return
	}
	c.JSON(http.StatusOK, list)
}

// 城市编号映射（diqubianhao.txt）懒加载。
var (
	cityMapOnce sync.Once
	cityMap     map[string]string
)

// getCityNo 根据「省+市+区」拼接的城市名查询行政区划编号（对应 Java diquToBianhao.getCityNo）。
func getCityNo(cityName string) string {
	cityMapOnce.Do(func() {
		cityMap = make(map[string]string)
		data, err := os.ReadFile("conf/diqubianhao.txt")
		if err != nil {
			logger.Mylog.Err(err).Caller().Send()
			return
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				cityMap[parts[1]] = parts[0]
			}
		}
	})
	return cityMap[cityName]
}
