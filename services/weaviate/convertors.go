package wvservice

import (
	"strconv"
	"time"

	"github.com/gookit/goutil"
	"github.com/weaviate/weaviate/entities/models"
	"gitlab.dev.ict/golang/libs/utils"
)

func GQLRespConvert[T any](gqlResp *models.GraphQLResponse, cname string) (items []*T) {
	v := gqlResp.Data["Get"].(map[string]interface{})[cname].([]interface{})
	utils.JsonToStruct(utils.Json(v), &items)
	return
}

type AdditionalMap map[string]interface{}

func (a AdditionalMap) ID() string {
	if v, ok := a["id"]; ok {
		return v.(string)
	}
	return ""
}

func (a AdditionalMap) CreationTime() *time.Time {
	if v, ok := a["creationTimeUnix"]; ok {
		t := time.UnixMilli(goutil.Int64(v))
		return &t
	}
	return nil
}

func (a AdditionalMap) LastUpdateTime() *time.Time {
	if v, ok := a["lastUpdateTimeUnix"]; ok {
		t := time.UnixMilli(goutil.Int64(v))
		return &t
	}
	return nil
}

func (a AdditionalMap) Score() float64 {
	v, _ := strconv.ParseFloat(a["score"].(string), 64)
	return v
}
