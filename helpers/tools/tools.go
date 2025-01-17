package tools

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"
)

func RandInt(min int, max int) int {
	return rand.New(rand.NewSource(time.Now().UTC().UnixNano())).Intn(max-min) + min
}

func GetIp() string {
	adds, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range adds {
		fmt.Printf("a: %#v\n", a.String())
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			fmt.Printf("ipnet: %+v\n", ipnet.IP)
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// addFieldToJson adds a new field to the JSON string.
func AddFieldToJson(js *string, field string, value interface{}) {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(*js), &jsonData)
	if err != nil {
		return
	}
	jsonData[field] = value

	updatedJson, err := json.Marshal(jsonData)
	if err != nil {
		return
	}
	*js = string(updatedJson)
}
