package root

import (
	"erp-cbqa-global/lib/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Index(context *gin.Context) {
	response.Json(context, http.StatusOK, nil)
}
