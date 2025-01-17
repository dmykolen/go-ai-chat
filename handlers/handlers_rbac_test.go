package handlers

import (
	"context"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	"gitlab.dev.ict/golang/go-ai/db/user_storage/repos"
	"gitlab.dev.ict/golang/go-ai/models/dto"
	"gitlab.dev.ict/golang/libs/gologgers"
	u "gitlab.dev.ict/golang/libs/utils"
	"gitlab.dev.ict/golang/util/httpclient"
)

// MockUserStorage is a mock implementation of the UserStorage interface
type MockUserStorage struct {
	mock.Mock
}

func (m *MockUserStorage) GetUsers(ctx context.Context) ([]us.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]us.User), args.Error(1)
}

var log = gologgers.New(gologgers.WithChannel("HANDLER-TEST"), gologgers.WithOC(), gologgers.WithLevel(gologgers.LevelTrace))
var ctx = u.CtxWithRid("3438953898348593985")
var storage = lo.Must(repos.NewUserStorageSQLite("db.sqlite", log))

const (
	reqUser = `{
    "Username": "Selmer",
    "Email": "wiYjgUI@iDoLyUk.info",
    "Password": "$2a$12$2ZqPsesp/p5avAq/VDoCb.Qns9iUjjFS5INQ1Q75TelHUi/1JFeOC",
    "Roles": [
        "SUPERADMIN",
        "ADMIN_VOIP"
    ]
}`
)

func prepareTest() (*fiber.App, *RBACHandler) {
	app := fiber.New(fiber.Config{})
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	handler := NewRBACHandler(WithLog(log), WithStore(storage))

	app.Get("/users", handler.GetUsers)
	app.Get("/roles", handler.GetRoles)
	app.Get("/groups", handler.GetGroups)
	app.Get("/permissions", handler.GetPermissions)

	// Posts
	app.Post("/users", handler.CreateUser)
	app.Post("/roles", handler.CreateRole)
	app.Post("/groups", handler.CreateGroup)
	app.Post("/permissions", handler.CreatePermission)

	// Puts
	app.Put("/users/:id", handler.UpdateUser)
	app.Put("/roles/:id", handler.UpdateRole)

	return app, handler
}

func TestGets(t *testing.T) {
	app, _ := prepareTest()

	t.Run("create-test", func(t *testing.T) {
		user := us.User{
			Username: faker.FirstName(),
			Email:    lo.ToPtr(faker.Email()),
			Password: faker.Password(),
			Roles: []us.Role{
				{ID: 1},
				{ID: 2},
			},
		}

		req := httpclient.HttpReqUniversal("/users", u.Json(&user), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		t.Log(u.DumpRequest(req))
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		t.Log("resp:", resp)
		t.Log("resp:", u.ReadBodyRespToString(resp))
	})

	t.Run("update-test-2", func(t *testing.T) {
		t.Log("dfsdfsdfsd")

		req := httpclient.HttpReqUniversal("/users/2", []byte(reqUser), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		req.Method = "PUT"
		t.Log(u.DumpRequest(req))
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		t.Log("resp:", resp)
		t.Log("resp:", u.ReadBodyRespToString(resp))
	})

	t.Run("getRoles", func(t *testing.T) {
		req := httpclient.HttpReq("/roles", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		t.Log("resp:", resp)
		t.Log("resp:", u.ReadBodyRespToString(resp))
	})

	t.Run("getUsers", func(t *testing.T) {
		req := httpclient.HttpReq("/users", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		t.Log("resp:", resp)
		t.Log("resp:", u.ReadBodyRespToString(resp))
	})

	t.Run("update-user-1", func(t *testing.T) {
		t.Log("dfsdfsdfsd")

		storage.GroupService().CreateOrUpdate(ctx, &us.Group{Name: "VAS_NOC", Description: "VAS_NOC", Roles: []us.Role{{Code: "ADMIN_VOIP"}}})

		reqU := &dto.UserDTO{
			ID:       1,
			Username: faker.FirstName(),
			Email:    lo.ToPtr(faker.Email()),
			Status:   "deactive",
			Roles:    []string{"ADMIN_VOIP"},
			Groups:   []string{"VAS_NOC"},
		}

		req := httpclient.HttpReqUniversal("/users/1", u.Json(reqU), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		req.Method = "PUT"
		t.Log(u.DumpRequest(req))
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		t.Log("resp:", resp)
		t.Log("resp:", u.ReadBodyRespToString(resp))

		usr, err := storage.GetUserFull(ctx, 1)
		assert.NoError(t, err)
		t.Logf(">>>user:\n%s", u.JsonPretty(usr))
	})

	t.Run("GetGroups", func(t *testing.T) {
		req := httpclient.HttpReq("/groups", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		t.Log("resp:", u.ReadBodyRespToString(resp))
	})

	t.Run("GetPermissions", func(t *testing.T) {
		req := httpclient.HttpReq("/permissions", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestPosts(t *testing.T) {
	app, _ := prepareTest()

	t.Run("CreatePermission", func(t *testing.T) {
		permissionPayload := `{
			"Name": "TEST_PERMISSION_2031",
			"Code": "TEST_PERMISSION_2031",
			"Description": "A test permission"
		}`
		req := httpclient.HttpReqUniversal("/permissions", []byte(permissionPayload), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		body := u.ReadBodyRespToString(resp)
		t.Log("body ===>", body)
	})

	t.Run("CreateRole", func(t *testing.T) {
		rolePayload := `{
			"name": "TEST_ROLE_2031",
			"code": "TEST_ROLE_2031",
			"description": "A test role",
			"permissions": ["TEST_PERMISSION_2031"]
		}`
		req := httpclient.HttpReqUniversal("/roles", []byte(rolePayload), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		body := u.ReadBodyRespToString(resp)
		t.Log("body ===>", body)
	})

	t.Run("CreateGroup", func(t *testing.T) {
		groupPayload := `{
			"Name": "TEST_GROUP_2031",
			"Description": "A test group",
			"Roles": ["TEST_ROLE_2031"]
		}`
		req := httpclient.HttpReqUniversal("/groups", []byte(groupPayload), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		body := u.ReadBodyRespToString(resp)
		t.Log("body ===>", body)
	})

	t.Run("CreateUser", func(t *testing.T) {
		userPayload := `{
			"Username": "cccvvvvvvvvvvvv",
			"Email": "testuser@example.com",
			"Password": "securepassword",
			"Roles": ["TEST_ROLE_2031"],
			"Groups": ["TEST_GROUP_2031"]
		}`
		req := httpclient.HttpReqUniversal("/users", []byte(userPayload), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := u.ReadBodyRespToString(resp)
		t.Log("body ===>", body)
		assert.Contains(t, body, "user success created")
	})

}

func Test_sync(t *testing.T) {
	err := storage.GroupService().SyncUserADGroups(ctx, 1, []string{"XXXXXXXXDEV", "XXXXXXXVAS_NOC"})
	assert.NoError(t, err)
	t.Log(u.JsonPrettyStr(lo.Must(storage.GroupService().GetGroups(ctx))))
}

func TestPuts(t *testing.T) {
	app, _ := prepareTest()
	t.Run("UpdateUser", func(t *testing.T) {
		userPayload := `{
			"Username": "TestUserUpdated",
			"Email": "Xyi@aaa.com",
			"Roles": ["TEST_ROLE_2", "55555555"],
			"Groups": ["DEV"]
		}`
		req := httpclient.HttpReqUniversal("/users/1", []byte(userPayload), ctx, fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		req.Method = "PUT"
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Optionally, verify the response body
		body := u.ReadBodyRespToString(resp)
		assert.Contains(t, body, "user updated successfully")
	})
}
