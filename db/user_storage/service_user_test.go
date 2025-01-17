package userstorage_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"gitlab.dev.ict/golang/go-ai/helpers/tools"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	"gitlab.dev.ict/golang/go-ai/db/user_storage/repos"
)

func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&us.User{}, &us.UserPhoto{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initForTests(t *testing.T) (context.Context, *us.StorageService, *gorm.DB) {
	var log = gologgers.New(gologgers.WithChannel("DB"), gologgers.WithOC(), gologgers.WithLevel(gologgers.LevelTrace))
	var ctx = utils.CtxWithRid("3438953898348593985")
	service, err := repos.NewUserStorageSQLite("db.sqlite?_foreign_keys=on", log, repos.WithDebug(false))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	db := service.Db()
	return ctx, service, db
}

func TestCreateUserWithPhoto(t *testing.T) {
	t.Log(os.Getwd())
	ctx, service, db := initForTests(t)

	// read img from file
	imgData, err := tools.ReadImageFromFile("../../_testdata/Dimon4ik.png")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Run("create_user_with_photo", func(t *testing.T) {
		user := &us.User{
			Username: "newwwuser",
			Password: "password",
		}

		err := service.CreateUserWithPhoto(ctx, user, imgData)
		assert.NoError(t, err)

		var createdUser *us.User
		err = db.First(&createdUser, "username = ?", "newwwuser").Error
		assert.NoError(t, err)
		t.Logf("Created user: %s", createdUser)

		// var userPhoto us.UserPhoto
		// err = db.First(&userPhoto, "user_id = ?", createdUser.ID).Error

		userPhoto, err := service.GetUserPhoto(ctx, createdUser.ID)
		assert.NoError(t, err)
		t.Logf("User photo: id=%d mimeType=%s => %v", userPhoto.UserID, userPhoto.MimeType, userPhoto.Data[:10])

	})

	t.Run("successful user creation with photo", func(t *testing.T) {
		user := &us.User{
			Username: "testuser2",
			Password: "password",
		}
		// photoData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
		photoData := imgData

		err := service.CreateUserWithPhoto(ctx, user, photoData)
		assert.NoError(t, err)

		var createdUser us.User
		err = db.First(&createdUser, "username = ?", "testuser2").Error
		assert.NoError(t, err)
		assert.Equal(t, "testuser2", createdUser.Username)

		var userPhoto us.UserPhoto
		err = db.First(&userPhoto, "user_id = ?", createdUser.ID).Error
		assert.NoError(t, err)

		pathToSave := "./test.png"
		t.Logf("Save image to %s", pathToSave)
		assert.NoError(t, userPhoto.SaveToFile(pathToSave))

		// assert.Equal(t, photoData, userPhoto.Data)
		assert.Equal(t, "png", userPhoto.MimeType)
	})

	t.Run("get user with photo", func(t *testing.T) {
		u, err := service.GetUserFull(ctx, 7)
		assert.NoError(t, err)
		assert.NotNil(t, u.Photo)
		ToJSON(t, u, "USER+")
	})

	t.Run("create and get user with photo", func(t *testing.T) {
		user := &us.User{Username: "NewUser7", Password: "LNNJKNBHKBKHB"}
		err := service.CreateUserWithPhoto(ctx, user, imgData)
		assert.NoError(t, err)
		ToJSON(t, user, "CREATED_USER")

		u, err := service.GetUserFull(ctx, user.ID)
		assert.NoError(t, err)
		assert.NotNil(t, u.Photo)
		ToJSON(t, u, "USER+")

		t.Log("Save photo to file")
		assert.NoError(t, u.Photo.SaveToFile("test.png"))

		t.Logf("Delete user %d\n", u.ID)
		// assert.NoError(t, service.DeleteUser(ctx, u.ID))

		// Delete user
		assert.NoError(t, service.DeleteUserWithOption(ctx, u.ID, false))
	})

	t.Run("create user, group, assign role, and check permissions", func(t *testing.T) {
		// Step 1: Create user
		user := &us.User{
			Username: faker.FirstName(),
			Email:    lo.ToPtr(faker.Email()),
			Password: faker.Password(),
		}
		err := service.CreateUser(ctx, user)
		assert.NoError(t, err)

		// Step 2: Create new group and link to user
		group, err := service.GroupService().CreateOrUpdateGroup(ctx, "TestGroup", "A test group")
		assert.NoError(t, err)
		ToJSON(t, lo.Must(service.GetUserFull(ctx, user.ID)), "USER_BEFORE_GROUP_APPEND")
		err = db.Model(&user).Association("Groups").Append(group)
		service.RoleService().AddRoleToUser(ctx, user.ID, "VOIP_USER")
		ToJSON(t, lo.Must(service.GetUserFull(ctx, user.ID)), "USER_AFTER_GROUP_APPEND")
		assert.NoError(t, err)

		// Step 3: Assign role to group
		role := &us.Role{
			Code:        "TEST_ROLE",
			Name:        "Test Role",
			Description: "A role for testing",
		}
		err = db.Create(role).Error
		assert.NoError(t, err)
		ToJSON(t, role, "ROLE")

		permission := &us.Permission{
			Code:        "TEST_PERMISSION",
			Name:        "Test Permission",
			Description: "A permission for testing",
		}
		err = db.Create(permission).Error
		assert.NoError(t, err)
		ToJSON(t, permission, "PERMISSION")

		err = db.Model(role).Association("Permissions").Append(permission)
		assert.NoError(t, err)
		ToJSON(t, role, "ROLE+")
		ToJSON(t, lo.Must(service.RoleService().GetRole(ctx, role.Code)), "ROLE++")

		err = service.GroupService().AssignRoleToGroup(ctx, group.ID, role.Code)
		assert.NoError(t, err)

		// Step 4: Check what permissions user have
		userPermissions, err := service.RoleService().GetUserPermissions(ctx, user.ID)
		assert.NoError(t, err)
		ToJSON(t, userPermissions, "USER_PERMISSIONS")

		permissionCodes := lo.Map(userPermissions, func(p us.Permission, _ int) string {
			return p.Code
		})

		assert.Contains(t, permissionCodes, "TEST_PERMISSION")
	})

	t.Run("create user with related data", func(t *testing.T) {
		user := &us.User{
			Username:      faker.FirstName(),
			Email:         lo.ToPtr(faker.Email()),
			Password:      faker.Password(),
			LastLoginTime: lo.ToPtr(time.Unix(faker.RandomUnixTime(), 0)),
			Chats: []us.Chat{{
				Chat:      datatypes.JSON(populateChatFakeData(goopenai.NewChat("You are expert...")).Json()),
				ChainName: "Teeest",
			}},
			Groups: []us.Group{
				{
					Name:  "ICT-VAS-WEB",
					Roles: []us.Role{{ID: 4}},
				},
			},
		}

		tx := db.Create(&user)
		t.Log(tx.Statement.Vars...)
		t.Log(tx.Statement.Preloads)
		t.Log(tx.Statement.Error)
		t.Log(user)
	})
}

func TestCreateUserWithID5(t *testing.T) {
	ctx, service, db := initForTests(t)

	user := &us.User{
		// Model:    gorm.Model{ID: 5},
		Username: faker.FirstName(),
		Email:    lo.ToPtr(faker.Email()),
		Password: faker.Password(),
	}
	err := service.CreateUser(ctx, user)
	assert.NoError(t, err)

	var createdUser us.User
	err = db.First(&createdUser, user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, user.Username, createdUser.Username)
	ToJSON(t, lo.Must(service.GetUserFull(ctx, user.ID)), "USER_AFTER_CREATE")
}

func TestDeleteUserWithAllRelatedData(t *testing.T) {
	t.Log("Sart...")
	ctx, service, db := initForTests(t)

	user, err := service.GetUserFull(ctx, 1)
	assert.NoError(t, err)

	// err = service.DeleteUserWithOption(ctx, user.ID, true)
	tx := db.Unscoped().Delete(user)
	assert.NoError(t, tx.Error)

	// Check that the user is deleted
	var deletedUser us.User
	err = db.Unscoped().First(&deletedUser, user.ID).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Check that the related data is deleted
	var chat us.Chat
	err = db.Unscoped().Where("user_id = ?", user.ID).First(&chat).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	var group us.Group
	err = db.Unscoped().First(&group, 1).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	var role us.Role
	err = db.Unscoped().First(&role, 5).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	var permission us.Permission
	err = db.Unscoped().First(&permission, 8).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Check that the linking tables are cleared
	var userGroups []us.UserGroup
	err = db.Unscoped().Where("user_id = ?", user.ID).Find(&userGroups).Error
	assert.NoError(t, err)
	assert.Empty(t, userGroups)

	var userRoles []us.UserRole
	err = db.Unscoped().Where("user_id = ?", user.ID).Find(&userRoles).Error
	assert.NoError(t, err)
	assert.Empty(t, userRoles)
}

func TestChats(t *testing.T) {
	t.Run("test - 1", func(t *testing.T) {
		usChat := help_create_fake_chat(t, 1)
		ToJSON(t, usChat, "create_fake_chat___usChat")
		t.Log(usChat.Data().Json(true))
	})

}

func TestJsonUpdate(t *testing.T) {
	s := `{"role":"system","content":"Hello, I'm a system message"}`
	tools.AddFieldToJson(&s, "time", time.Now().Format("2006-01-02T15:04:05.000"))
	tools.AddFieldToJson(&s, "time", time.Now().Format("2006-01-02T15:04:05.000"))
	t.Log(s)
}

func help_create_fake_user(t *testing.T) *us.User {
	user := &us.User{
		Username:      faker.FirstName(),
		Email:         lo.ToPtr(faker.Email()),
		Password:      faker.Password(),
		LastLoginTime: lo.ToPtr(time.Unix(faker.RandomUnixTime(), 0)),
	}
	t.Logf("FakeUser: %s", user)
	return user
}

func help_create_fake_chat(t *testing.T, uid uint) *us.Chat {
	chat := &us.Chat{
		UserID: uid,
		Chat:   datatypes.JSON(populateChatFakeData(goopenai.NewChat("Syyyyyys prompt test")).Json()),
	}
	t.Logf("FakeChat: %s", utils.Json(chat))
	return chat
}

func populateChatFakeData(ch *goopenai.Chat) *goopenai.Chat {
	for _, v := range lo.Range(3) {
		_ = v
		ch.AddUserMessage(faker.Sentence()).AddAssistantMessage(faker.Sentence())
	}

	// for _, v := range []int{0, 1, 2, 3, 45, 6, 7, 8, 9} {
	// 	_ = v
	// 	ch.AddUserMessage(faker.Sentence()).AddAssistantMessage(faker.Sentence())
	// 	ch.AddUserMessage(faker.Sentence()).AddAssistantMessage(faker.Sentence())
	// }
	return ch
}
