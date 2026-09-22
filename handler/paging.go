package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// queryInt 读取查询参数，缺失或解析失败时返回默认值。
func queryInt(c *gin.Context, key string, def int) int {
	if v := c.Query(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// queryInt64 读取查询参数并转为 int64，缺失或解析失败时返回默认值。
func queryInt64(c *gin.Context, key string, def int64) int64 {
	if v := c.Query(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

// queryFloat 读取查询参数并转为 float64，缺失或解析失败时返回默认值。
func queryFloat(c *gin.Context, key string) float64 {
	if v := c.Query(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return 0
}

// parsePage 解析分页参数，兼容三种命名约定：
//   - current/size（新控制器，如 OrderManageCon/ProductCon）
//   - page/length（旧控制器，如 LoRacon/faultCon）
//   - pageNum/pageSize（hotel/invoice）
//
// 返回 current（页码，从 1 开始）与 size（每页条数），均保证 >= 1。
func parsePage(c *gin.Context) (current, size int) {
	current = queryInt(c, "current", 0)
	if current == 0 {
		current = queryInt(c, "page", 0)
	}
	if current == 0 {
		current = queryInt(c, "pageNum", 1)
	}
	if current < 1 {
		current = 1
	}

	size = queryInt(c, "size", 0)
	if size == 0 {
		size = queryInt(c, "length", 0)
	}
	if size == 0 {
		size = queryInt(c, "pageSize", 10)
	}
	if size < 1 {
		size = 10
	}
	return current, size
}

// pageOffset 由页码与每页条数计算 SQL offset（供 gorm Offset 使用）。
func pageOffset(current, size int) int {
	return (current - 1) * size
}
