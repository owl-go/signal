package logsvr

import (
	"encoding/json"

	"mgkj/pkg/log"

	"github.com/gin-gonic/gin"
)

func getJsonMap(c *gin.Context) map[string]interface{} {
	data, err := c.GetRawData()
	if err != nil {
		log.Errorf("read no data from raw")
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	var params map[string]interface{}
	err = json.Unmarshal(data, &params)
	if err != nil {
		return nil
	}
	return params
}

func response(c *gin.Context, statusCode int, resultCode int, resultMsg string) {
	c.JSON(statusCode, gin.H{
		"code": resultCode,
		"msg":  resultMsg,
	})
}

func reporting(c *gin.Context) {
	data := getJsonMap(c)
	if data == nil {
		log.Errorf("read no data")
		response(c, 200, 1, "verify data error!")
		return
	}
	insertKeys := []string{"reportId", "name", "rid", "uid", "rvalue", "timestamp"}
	var insertValues [][]interface{}
	var tempValue []interface{}
	for _, v := range insertKeys {
		var k string
		if v == "reportId" {
			k = "id"
		} else {
			k = v
		}
		if _, ok := data[k]; ok {
			tempValue = append(tempValue, data[k])
		}
	}
	if len(tempValue) == len(insertKeys) {
		insertValues = append(insertValues, tempValue)
	}
	if len(insertValues) > 0 {
		mysql.Insert("reporting", insertKeys, insertValues)
	}

	response(c, 200, 0, "success")
	return
}

func tracingReport(c *gin.Context) {
	data := getJsonMap(c)
	if data == nil {
		log.Errorf("read no data")
		response(c, 200, 1, "verify data error!")
		return
	}
	logger.Infof("msg", data)
	response(c, 200, 0, "success")
	return
}

//健康监测
func probeHandler(c *gin.Context) {
	response(c, 200, 0, "success")
	return
}
