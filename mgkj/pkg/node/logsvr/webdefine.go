package logsvr

import "github.com/gin-gonic/gin"

func Entry(r *gin.Engine) {
	v1 := r.Group("/v1")
	{
		v1.POST("/api/reporting", reporting)
		v1.POST("/api/tracing_report", tracingReport)
	}
	r.POST("/api/v1/probe", probeHandler)
}
