package helpers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gookit/goutil"
	"github.com/gookit/slog"
	"github.com/samber/lo"
	gh "gitlab.dev.ict/golang/libs/gohttp"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gologgers/applogger"
	"gitlab.dev.ict/golang/libs/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	R           = "rid"
	CtxRidFiber = "requestid"
	CtxLogger   = "ctx_logger"
	CtxUser     = "user"
	CtxIsAuth   = "isAuth"
)

var (
	fs = fmt.Sprintf
	v  = validator.New()
)

func init() {
	v.RegisterValidation("msisdn", isMSISDN)
}

func IsValidJSON(js string) bool {
	var i interface{}
	if err := json.Unmarshal([]byte(js), &i); err != nil {
		return false
	}
	return true
}

func CtxValue[T any](c *fiber.Ctx, key string) T {
	return c.Locals(key).(T)
}

func Rid(c *fiber.Ctx) string {
	return c.Locals(CtxRidFiber).(string)
}

func Ridd(c *fiber.Ctx) (string, string) {
	return R, Rid(c)
}

func Log(c *fiber.Ctx) *slog.Record {
	return c.Locals(CtxLogger).(*slog.Record)
}

// GetFileNameWithoutExt - File name without extension
func GetFileNameWithoutExt(fn string) string {
	return fn[:len(fn)-len(filepath.Ext(fn))]
}

// isMSISDN is a custom validator function to check if a string is a valid MSISDN
func isMSISDN(fl validator.FieldLevel) bool {
	msisdn := fl.Field().String()
	pattern := `^380\d{9}$`
	match, _ := regexp.MatchString(pattern, msisdn)
	return match
}

// ValidateMSISDN - validate msisdn
func ValidateMSISDN(msisdn string) error {
	err := v.Var(msisdn, "required,msisdn")
	if err != nil {
		e := err.(validator.ValidationErrors)[0]
		return fmt.Errorf("invalid - %s[%v]", e.ActualTag(), e.Value())
	}
	return nil
}

func ValidateURLAndExtractDomain(url string) (string, bool) {
	var urlRegex = regexp.MustCompile(`^(https?://)?(www\.)?([a-zA-Z0-9.-]+)(:[0-9]+)?(/.*)?$`)
	if urlRegex.MatchString(url) {
		matches := urlRegex.FindStringSubmatch(url)
		return matches[3], true
	}
	return "", false
}

// Hash a password using bcrypt
func HashPassword(password string) (string, error) {
	// Generate a salt with a cost factor of 12
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// Verify a password against its hash
func VerifyPassword(inputPassword, storedHashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(inputPassword))
	return err == nil
}

func ReadFileBuffered(f string, ch chan []byte) {
	buffer := bytes.NewBuffer(goutil.Must(os.ReadFile(f)))
	bufToRead := make([]byte, 128)
	for {
		n, err := buffer.Read(bufToRead)
		if err == io.EOF {
			close(ch)
			break
		}
		ch <- bufToRead[:n]
		time.Sleep(1 * time.Second)
	}
}

func CountLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return CountLines(file)
}

func CountLines(r io.Reader) (int, error) {
	lineCount := 0
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return lineCount, nil
}

func ValidateWithLog(r *slog.Record, obj interface{}) (err error) {
	err = v.Struct(obj)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, vErr := range validationErrors {
			r.Infof("invalid value=[%s] for field=[%s]! ERR=[%s]", vErr.Value(), vErr.Field(), vErr.Error())
		}
	}
	return
}

func Validate(obj interface{}) (err error) {
	err = v.Struct(obj)
	if err != nil {
		return errors.New(err.(validator.ValidationErrors).Error())
	}
	return
}

// ConvertToInt takes an interface as input and converts it to an int
func ConvertToInt(input interface{}) (int, error) {
	switch v := input.(type) {
	case int:
		return v, nil
	case string:
		// Attempt to convert string to int
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("failed to convert string to int: %v", err)
		}
		return i, nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

func PrintEnvVars() {
	evs := os.Environ()
	sort.Strings(evs)
	for _, envVar := range evs {
		fmt.Println(envVar)
	}
}

func HttpClient(log any, to int, certPath string, isv bool, withPrx ...bool) *http.Client {
	cl := gh.New().WithTimeout(lo.Ternary(to == 0, 360, to))
	switch l := log.(type) {
	case *gologgers.Logger:
		cl.WithLogger(l)
	case *applogger.LogCfg:
		cl.WithLogCfg(l)
	default:
		panic("unknown LOGGER")
	}

	if isv {
		cl = cl.WithISV()
	}
	if certPath != "" {
		cl = cl.WithSSLPath(certPath)
	}

	return cl.WithProxy(processProxy(withPrx...)).Build().Client
}

func processProxy(withPrx ...bool) func(*http.Request) (*url.URL, error) {
	if utils.FirstOrDefault(true, withPrx...) {
		return gh.UseProxy(true, "")
	}
	return nil
}
