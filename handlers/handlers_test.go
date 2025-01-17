package handlers

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/gookit/goutil"
	"github.com/stretchr/testify/assert"
	help "gitlab.dev.ict/golang/go-ai/helpers"

	// "gitlab.dev.ict/golang/go-web-scc/pkg/ad"
	ad "gitlab.dev.ict/golang/libs/goldap"
)

const (
	sqlRes = `[{"AGENT":"","CHANNEL":"","CODE":181,"CUSTOMER_STATE":"REGISTERED","CUSTOMER_TYPE":"INDIVIDUAL","ENTRY_DATE":"2017-10-27T13:07:45+03:00"},{"AGENT":"","CHANNEL":"","CODE":234,"CUSTOMER_STATE":"REGISTERED","CUSTOMER_TYPE":"INDIVIDUAL","ENTRY_DATE":"2017-10-27T13:07:45+03:00"},{"AGENT":"","CHANNEL":"","CODE":301,"CUSTOMER_STATE":"REGISTERED","CUSTOMER_TYPE":"INDIVIDUAL","ENTRY_DATE":"2017-10-27T13:07:45+03:00"},{"AGENT":"","CHANNEL":"","CODE":302,"CUSTOMER_STATE":"REGISTERED","CUSTOMER_TYPE":"INDIVIDUAL","ENTRY_DATE":"2017-10-27T13:07:45+03:00"},{"AGENT":"","CHANNEL":"","CODE":303,"CUSTOMER_STATE":"REGISTERED","CUSTOMER_TYPE":"INDIVIDUAL","ENTRY_DATE":"2017-10-27T13:07:45+03:00"},{"AGENT":"","CHANNEL":"","CODE":304,"CUSTOMER_STATE":"REGISTERED","CUSTOMER_TYPE":"INDIVIDUAL","ENTRY_DATE":"2017-10-27T13:07:45+03:00"}]`
)

func Test_AD(t *testing.T) {
	adClient := &ad.LDAPConf{InsecSkipVerify: true}
	env.Parse(adClient)
	t.Log(adClient)
	adClient.Init(ad.WithDebug(true), ad.WithLogger(log))
	t.Log(adClient)
	// t.Logf("IS_USER[%s]_VALID => %t", os.Getenv("USER"), adClient.LDAPAuthUser(os.Getenv("USER"), os.Getenv("MY_PASSWORD"), nil))
	sr, e := adClient.LDAPSearchUserByLogin("dmykolen")
	assert.NoError(t, e)
	ad.ParseGroupsFromSearchResult(log.Rec("AD"), sr)
	isOk, entries := adClient.LDAPAuthUser("dmykolen", "d18J6EGBDot!S", nil)
	t.Log("IS_USER_VALID =>", isOk)
	t.Log("ENTRIES =>", entries)
	t.Log(ad.ParseGroupsFromEntry(log.Rec("ADD"), entries))

}
func Test_1(t *testing.T) {
	rp := "../assets/text.txt"
	file := goutil.Must(filepath.Abs(rp))

	t.Log("FILE:", file)
	t.Log("rp FILE:", goutil.Must(filepath.Rel("../", rp)))
}
func Test_addUser(t *testing.T) {
	tests := []struct {
		name string
		user []*User
	}{
		{name: "test-1", user: []*User{NewUser("test-1")}},
		{name: "test-2", user: []*User{NewUser("test-2")}},
		{name: "test-3", user: []*User{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotU, _ := addUser(&fiber.Ctx{}, tt.user...)
			t.Log(gotU)
			appStoreForUsers.Range(func(key, value interface{}) bool {
				t.Logf("key=%v, value=%v\n", key, value)
				return true
			})
		})
	}
}

func Test_Quote(t *testing.T) {
	str := `dfdfsdfs
	sdfsdggsgsdgs
		sdfsddsfsd
		sdfsdfsdf`
	t.Log("STR =>", str)
	t.Log(strings.Trim(strconv.Quote(str), `"`))
	t.Log(strconv.QuoteToASCII(str))

}

func Test_read(t *testing.T) {
	t.Log("Current DIR:", goutil.Must(os.Getwd()))
	buffer := bytes.NewBuffer(goutil.Must(os.ReadFile("text.txt")))
	bufToRead := make([]byte, 128)
	for {
		n, err := buffer.Read(bufToRead)
		if err == io.EOF {
			break
		}
		t.Logf("buffer.Read() => %d; %s", n, bufToRead[:n])
	}
}

func Test_read_chan(t *testing.T) {
	ch := make(chan []byte)
	go help.ReadFileBuffered("text.txt", ch)

	for data := range ch {
		t.Logf("data: %s", data)
	}

	// for {
	// 	select {
	// 	case data, ok := <-ch:
	// 		if !ok {
	// 			t.Log("channel closed")
	// 			return
	// 		}
	// 		t.Logf("data: %s", data)
	// 	}
	// }

}

func Test_Regxp(t *testing.T) {
	arr := []string{
		"https://lifecell.ua/uk/malii-biznes-lifecell/servisi/ip-telefoniia/",
		"https://ai.dev.ict/vectordb-admin",
		"lifecell.ua/tarify",
		"http://www.lifecell.ua/tarify",
		"http:/www.lifecell.ua/tarify",
		"lifecell.com_/tarify",
		"https://www.example.com",
		"http://example.com/path",
		"https://sub.example.com",
		"https://www.example.co.uk",
		"ftp://example.com", // This should be invalid as per our regex
		"https://example",
		"http://localhost:8080",
		"https://192.168.0.1",
		"https://example.com:8080/path?query=123",
	}

	for _, v := range arr {
		domain, isValid := validateURLAndExtractDomain(v)
		t.Logf("validateURL(%s) => HOSTNAME=[%v] IS_VALID=%v", v, domain, isValid)

	}

}

func validateURLAndExtractDomain(url string) (string, bool) {
	var urlRegex = regexp.MustCompile(`^(https?://)?(www\.)?([a-zA-Z0-9.-]+)(:[0-9]+)?(/.*)?$`)
	if urlRegex.MatchString(url) {
		// Extract the domain name from the URL
		matches := urlRegex.FindStringSubmatch(url)
		fmt.Println("matches=====>", matches)
		return matches[3], true
	}
	return "", false
}

func TestLogActiveSSEConnections(t *testing.T) {
	// Initialize appStoreForUsers with test data
	appStoreForUsers = sync.Map{}

	// Create test users
	user1 := &User{
		UUID:     "user1-uuid",
		Login:    "user1",
		ConnTime: time.Now().Add(-10 * time.Minute),
		ActiveConns: map[string]*bufio.Writer{
			"conn1": bufio.NewWriter(os.Stdout),
			"conn2": bufio.NewWriter(os.Stdout),
		},
	}

	user2 := &User{
		UUID:     "user2-uuid",
		Login:    "user2",
		ConnTime: time.Now().Add(-20 * time.Minute),
		ActiveConns: map[string]*bufio.Writer{
			"conn3": bufio.NewWriter(os.Stdout),
			"conn4": bufio.NewWriter(os.Stdout),
		},
	}

	// Add users to appStoreForUsers
	appStoreForUsers.Store(user1.UUID, user1)
	appStoreForUsers.Store(user2.UUID, user2)

	// Create an instance of AppHandler with a mock logger

	appHandler := &AppHandler{log: log}

	// Call the LogActiveSSEConnections method
	appHandler.LogActiveSSEConnections()
}
