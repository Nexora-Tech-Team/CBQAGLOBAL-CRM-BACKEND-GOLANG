package response

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"erp-cbqa-global/lib/errorlog"
)

type Response struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type ResponsePagination struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}
type Meta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func Error(c *gin.Context, code int, message string) {
	if code == http.StatusInternalServerError {
		err := errorlog.SendErrorLogGChat(&errorlog.ErrorData{
			Error:  message,
			Params: c.Params,
			Method: c.Request.Method,
			Path:   c.Request.RequestURI,
			Host:   c.Request.Host,
			Header: c.Request.Header,
		})
		if err != nil {
			log.Println(err)
		}
	}

	c.JSON(code, Response{Data: nil, Message: message})
}

func Json(c *gin.Context, code int, data interface{}) {
	c.JSON(code, Response{Data: data, Message: "OK"})
}

func JsonPagination(c *gin.Context, code int, data interface{}, page, limit int, total int64) {
	c.JSON(code, Response{
		Data: ResponsePagination{
			Data: data,
			Meta: Meta{
				Page:  page,
				Limit: limit,
				Total: total,
			},
		},
		Message: "OK",
	})
}
