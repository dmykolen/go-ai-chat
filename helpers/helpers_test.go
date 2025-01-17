package helpers

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dev.ict/golang/go-ai/helpers/tools"
	"golang.org/x/crypto/bcrypt"
)

func Test_Random(t *testing.T) {
	r1 := tools.RandInt(10, 500)
	r2 := tools.RandInt(10, 500)
	assert.NotEqual(t, r1, r2, "Expected random numbers to be different")
	assert.GreaterOrEqual(t, r1, 10, "Expected random number to be greater or equal to 10")
	assert.LessOrEqual(t, r1, 500, "Expected random number to be less or equal to 500")
	t.Logf("r1=%d r2=%d", r1, r2)
}

func Test_ValidateMSISDN(t *testing.T) {
	tests := []struct {
		msisdn string
		valid  bool
	}{
		{"1234567890", false},
		{"380632107489", true},
	}

	for _, test := range tests {
		err := ValidateMSISDN(test.msisdn)
		if test.valid {
			assert.NoError(t, err, "Expected no error for valid MSISDN: %s", test.msisdn)
		} else {
			assert.Error(t, err, "Expected error for invalid MSISDN: %s", test.msisdn)
			if err != nil {
				t.Log(err)
			}
		}
	}
}

func TestHashPassword(t *testing.T) {
	password := "password123"
	hashedPassword, err := HashPassword(password)
	assert.NoError(t, err, "Expected no error when hashing password")
	t.Logf("password=%s hashed_password=%s", password, hashedPassword)

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	assert.NoError(t, err, "Expected no error when comparing hashed password with original password")
}

func Test_RegexPatterns(t *testing.T) {
	r := regexp.MustCompile(`.*\.(js|css|png)$`)
	assert.Regexp(t, r, "/js/main.js")
	assert.Regexp(t, r, "/main.js")
	assert.Regexp(t, r, "js/main.js")
	assert.NotRegexp(t, r, "css/index")
	assert.NotRegexp(t, r, "index")
	assert.NotRegexp(t, r, "main.html")
}

func Test_Fmt(t *testing.T) {
	t.Logf("%-6s %s", "GET", "/api/v1/users")
	t.Logf("%-6s %s", "POST", "/api/v1/users")
	t.Logf("%-6s %s", "DELETE", "/api/v1/users")
}
