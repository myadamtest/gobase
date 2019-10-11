package ginext

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/myadamtest/gobase/logkit"
)

type BaseResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func New() *gin.Engine {
	if logkit.IsDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DefaultWriter = logkit.NewLogWriter(logkit.LevelInfo)
	gin.DefaultErrorWriter = logkit.NewLogWriter(logkit.LevelError)
	r := gin.New()
	r.Use(gin.Recovery())
	if logkit.IsDebug() {
		r.Use(gin.Logger())
		r.Use(gin.ErrorLogger())
	}
	r.Use(errorLog)
	return r
}

func errorLog(c *gin.Context) {
	path := c.Request.URL.Path
	requestId := c.GetHeader("X-Request-Id")
	c.Next()
	errors := c.Errors.ByType(gin.ErrorTypeAny)
	method := c.Request.Method
	statusCode := c.Writer.Status()
	if len(errors) > 0 {
		fmt.Fprintf(gin.DefaultErrorWriter, "[GIN] %s %s %3d %s %s", method, path, statusCode, requestId, errors)
	}
}

func JSON(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(200, &BaseResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func QueryInt(c *gin.Context, key string) int {
	str := c.Query(key)
	if str == "" {
		return 0
	}
	re, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return re
}

func PostFormInt(c *gin.Context, key string) int {
	str := c.PostForm(key)
	if str == "" {
		return 0
	}
	re, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return re
}
