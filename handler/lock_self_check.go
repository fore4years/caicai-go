package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// ============ ZijianController（base /zijian，锁自检） ============

// ZijianController 对齐 Java yingjianbufen.controller.zijianCon。
type ZijianController struct{}

// Shibai /zijian/shibai?id= —— 对齐 zijianSerImpl.shibai：自检失败，电量置 n。
func (c *ZijianController) Shibai(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.shibai(ctx.Query("id"))))
}

func (c *ZijianController) shibai(id string) int {
	if _, err := repository.OnlineLockByLockID(id); err != nil {
		// 锁未上线，业务处理完毕
		return 1
	}
	if repository.UpdateOnlineLockBattery(id, "n") > 0 {
		return 1
	}
	return 0
}

// Chenggong /zijian/chenggong?id= —— 对齐 zijianSerImpl.chenggong：自检成功，注册上线/更新电量。
func (c *ZijianController) Chenggong(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.chenggong(ctx.Query("id"))))
}

func (c *ZijianController) chenggong(id string) int {
	if _, err := repository.OnlineLockByLockID(id); err != nil {
		// 锁未上线：生成 9 位编号并注册上线（battery=y, online=n）
		number := 0
		maxNumStr := repository.MaxOnlineLockNumber()
		if maxNumStr != "" {
			if n, e2 := strconv.Atoi(maxNumStr); e2 == nil {
				number = n
			}
		}
		inNum := fmt.Sprintf("%09d", number+1)
		if repository.InsertOnlineLock(inNum, id) > 0 {
			return 1
		}
		return 0
	}
	if repository.UpdateOnlineLockBattery(id, "y") > 0 {
		return 1
	}
	return 0
}
