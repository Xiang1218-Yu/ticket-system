package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ok 返回 200 + 成功体。
func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

// created 返回 201 + 成功体。
func created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "created", "data": data})
}

// fail 把业务错误映射为合适的 HTTP 状态码并返回统一错误体。
// 单一职责：负责 service error → HTTP 的映射与响应格式。
func fail(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "资源不存在"})
	case strings.Contains(msg, "无权"):
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": msg})
	case strings.Contains(msg, "不能为空") || strings.Contains(msg, "不存在") ||
		strings.Contains(msg, "已存在") || strings.Contains(msg, "不允许") ||
		strings.Contains(msg, "长度") || strings.Contains(msg, "已关闭") ||
		strings.Contains(msg, "已评价") || strings.Contains(msg, "未变更") ||
		strings.Contains(msg, "需在") || strings.Contains(msg, "缺少"):
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": msg})
	case strings.Contains(msg, "未登录") || strings.Contains(msg, "失效"):
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": msg})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
	}
}

// bindJSON 解析 JSON 并在失败时返回 400。
func bindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误: " + err.Error()})
		return false
	}
	return true
}

// pageQuery 读取分页参数并归一化。
func pageQuery(c *gin.Context) (page, pageSize int) {
	page = atoi(c.Query("page"), 1)
	pageSize = atoi(c.Query("page_size"), 10)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return
}

// atoi 解析整型查询参数，失败回退默认值。
func atoi(s string, def int) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	if s == "" {
		return def
	}
	return n
}
