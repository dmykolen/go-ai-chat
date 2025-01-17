package userstorage_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	"gitlab.dev.ict/golang/go-ai/db/user_storage/repos"
)

var log = gologgers.New(gologgers.WithChannel("DB"), gologgers.WithOC(), gologgers.WithLevel(gologgers.LevelTrace))
var ctx = utils.CtxWithRid("3438953898348593985")

func Test_gorm_groups(t *testing.T) {
	repo, err := repos.NewUserStorageSQLite("db.sqlite", log, repos.WithDebug(false))
	if err != nil {
		t.Fatal(err)
	}

	const gr1, gr2 = "ICT-DEV-1", "ICT-DEV-2"

	t.Run("user-with-2-group", func(t *testing.T) {
		u := us.User{Username: "atararay", Password: "fdlfgkldfgk"}
		repo.CreateUser(ctx, &u)
		ToJSON(t, u, "CREATE NEW USER")
		ToJSON(t, lo.Must(repo.RoleService().GetUserPermissions(ctx, u.ID)), "PERMISSIONS")

		gr, err := repo.GroupService().CreateOrUpdateGroup(ctx, "ICT-DEV-1", "devs from 1st line")
		assert.NoError(t, err)
		ToJSON(t, gr, "GROUP CREATED")
		err = repo.GroupService().AssignRoleToGroup(ctx, 1, "vectordb_admin")
		assert.NoError(t, err)
		repo.GroupService().SyncUserADGroups(ctx, u.ID, []string{"ICT-DEV-1", "ICT-DEV-2"})
		ToJSON(t, lo.Must(repo.GetUserFull(ctx, u.ID)), "USER_FULL")
	})

	t.Run("group-operate", func(t *testing.T) {
		// remove all groups
		tx := repo.Db().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&us.Group{})
		assert.NoError(t, tx.Error)

		ToJSON(t, lo.Must(repo.GroupService().GetGroups(ctx)), "GROUPS")
	})

	t.Run("user-with-1-group", func(t *testing.T) {
		ToJSON(t, lo.Must(repo.GroupService().GetGroups(ctx)), "GROUPS")

		id := uint(6)
		repo.RoleService().AddRoleToUser(ctx, id, "admin")

		ToJSON(t, lo.Must(repo.RoleService().GetUserPermissions(ctx, id)), "PERMISSIONS")
		ToJSON(t, lo.Must(repo.GetUserFull(ctx, id)), "ROLES")

		isHavePerm, err := repo.RoleService().HasPermissionIncludingGroups(ctx, id, "vectordb:write")
		assert.NoError(t, err)
		t.Log("isHavePerm:", isHavePerm)
	})

	t.Run("user-with-1-role", func(t *testing.T) {
	})

	t.Run("user-with-1-role-2", func(t *testing.T) {
	})
}

func Test_gorm_sqlite2(t *testing.T) {
	repo, err := repos.NewUserStorageSQLite("db.sqlite", log, repos.WithDebug(false))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("user-with-2-roles", func(t *testing.T) {
		huan := us.User{Username: "Leo", Password: "fdlfgkldfgk"}
		repo.CreateUser(ctx, &huan)
		repo.RoleService().AddRoleToUser(ctx, huan.ID, "vectordb_admin")
		repo.RoleService().AddRoleToUser(ctx, huan.ID, "admin")
		huanNew, _ := repo.GetUserFull(ctx, huan.ID)
		ToJSON(t, huanNew, "HUAN NEW")
		perms, _ := repo.RoleService().GetUserPermissions(ctx, huan.ID)
		ToJSON(t, perms, "PERMISSIONS")
	})

	t.Run("user-with-1-role", func(t *testing.T) {
		huan := us.User{Username: "Sisiliya", Password: "vvvvvvvvv"}
		repo.CreateUser(ctx, &huan)
		repo.RoleService().AddRoleToUser(ctx, huan.ID, "vectordb_admin")
		huanNew, _ := repo.GetUserFull(ctx, huan.ID)
		ToJSON(t, huanNew, "HUAN NEW")
		perms, _ := repo.RoleService().GetUserPermissions(ctx, huan.ID)
		ToJSON(t, perms, "PERMISSIONS")
	})

	t.Run("user-with-1-role-2", func(t *testing.T) {
		huan := us.User{Username: "Sisiliya", Password: "vvvvvvvvv"}
		repo.CreateUser(ctx, &huan)
		repo.RoleService().AddRoleToUser(ctx, huan.ID, "vectordb_admin")
		huanNew, _ := repo.GetUserFull(ctx, huan.ID)
		ToJSON(t, huanNew, "HUAN NEW")
		perms, _ := repo.RoleService().GetUserPermissions(ctx, huan.ID)
		ToJSON(t, perms, "PERMISSIONS")
	})
}
func Test_gorm_sqlite(t *testing.T) {
	t.Log("Starting sqlite test")
	repo, err := repos.NewUserStorageSQLite("db.sqlite", log, repos.WithDebug(false))
	if err != nil {
		t.Fatal(err)
	}
	d := repo.Db()

	dima := us.User{Username: "dmykolen", Password: "XXXXXXXXXXXXX"}
	err = repo.CreateUser(ctx, &dima)
	assert.NoError(t, err)
	ToJSON(t, dima, "CREATE USER")

	var u us.User
	// var r us.Role
	// var p us.Permission
	var rr us.Role
	var pp []us.Permission

	d.WithContext(ctx).Model(us.User{}).First(&u)
	ToJSON(t, u, "USER")

	u.LastLoginTime = lo.ToPtr(time.Now().Add(time.Duration(3)*time.Minute + time.Duration(20)*time.Second))

	// update user in db
	d.WithContext(ctx).Save(&u)

	d.WithContext(ctx).Model(us.User{}).First(&u)
	ToJSON(t, u, "USER")

	d.WithContext(ctx).Model(us.User{}).First(&u)
	ToJSON(t, u, "USER")

	// d.WithContext(ctx).Model(us.Role{}).Preload("Permission").Find(&rr)
	d.WithContext(ctx).Model(us.Role{}).Preload("Permissions").Find(&rr)
	ToJSON(t, rr, "ROLE")

	d.WithContext(ctx).Model(us.Permission{}).Find(&pp)
	ToJSON(t, pp, "Permission")

	// d.WithContext(ctx).Model(us.Permission{}).Find(&pp, us.Permission{Code: "chat:write"})
	// d.WithContext(ctx).Model(us.Permission{}).Where("code like ?", "user%").Find(&pp)
	d.WithContext(ctx).Where("code like ?", "user%").Find(&pp)
	ToJSON(t, pp, "Permission")

	rol, e := repo.RoleService().GetRole(ctx, "admin")
	assert.NoError(t, e)
	ToJSON(t, rol, "repo.RoleService().GetRole")

}

func TestPerm_sqlite(t *testing.T) {
	t.Log("Starting sqlite test")
	store, err := repos.NewUserStorageSQLite("db2.sqlite", gologgers.Defult(), repos.WithDebug(true))
	if err != nil {
		t.Fatal(err)
	}

	if err := us.InitializeRolesAndPermissions(store.Db(), log.RecWithRid("444444444", "DB-REC"), false); err != nil {
		t.Fatalf("failed to initialize roles and permissions. Err=%v\n", err)
	}

	err = store.CreateUser(ctx, &us.User{
		Model:    gorm.Model{ID: 1},
		Username: "dmykolen",
		Password: "123456",
	})
	if err != nil {
		t.Errorf("create user failed: %v", err)
	}

	err = store.RoleService().AddRoleToUser(ctx, 1, "admin")
	if err != nil {
		t.Errorf("add role to user failed: %v", err)
	}

	t.Log("###########################################")
	// Preload("Roles.Permissions").
	u, _ := store.GetUser(ctx, 1)
	ToJSON(t, u)
	u, _ = store.GetUserFull(ctx, 1)
	ToJSON(t, u)
	t.Log("###########################################")
	r, _ := store.RoleService().GetUserRoles(ctx, 1)
	ToJSON(t, r)
	t.Log("###########################################")
	p, _ := store.RoleService().GetUserPermissions(ctx, 1)
	ToJSON(t, p)
	t.Log("###########################################")
	up, _ := store.RoleService().GetUserRolesAndPermissions(ctx, 1)
	ToJSON(t, up)
	t.Log("###########################################")
	up1, _ := store.RoleService().HasPermission(ctx, 1, "admin:full")
	ToJSON(t, up1, "USER HAS PERMISSION [admin:full]")
	t.Log("###########################################")
	up1, _ = store.RoleService().HasRole(ctx, 1, "admin")
	ToJSON(t, up1, "USER HAS Role [admin]")
	t.Log("###########################################")
	up1, _ = store.RoleService().HasRole(ctx, 1, "usual")
	ToJSON(t, up1, "USER HAS Role [usual]")

}

func TestGetUserPermissions(t *testing.T) {
	// Setup in-memory database
	// db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db, err := gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Migrate the schema
	// err = db.AutoMigrate(&us.User{}, &us.Role{}, &us.Permission{}, &us.UserRole{}, &us.RolePermission{})
	err = us.AutoMigrate(db)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	err = us.InitializeRolesAndPermissions(db, log.RecWithRid("444444444", "DB-REC"), false)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create test data
	user := us.User{Model: gorm.Model{ID: 1}}
	role := us.Role{ID: 1, Code: "admin"}
	permission := us.Permission{ID: 1, Code: "read"}

	db.Create(&user)
	db.Create(&role)
	db.Create(&permission)
	db.Create(&us.UserRole{UserID: user.ID, RoleID: role.ID})
	db.Create(&us.RolePermission{RoleID: role.ID, PermissionID: permission.ID})

	// Create UserRoleService
	service := us.NewUserRoleService(db, log)

	// Test GetUserPermissions
	t.Run("GetUserPermissions", func(t *testing.T) {
		permissions, err := service.GetUserPermissions(context.Background(), user.ID)
		assert.NoError(t, err)
		assert.Len(t, permissions, 1)
		assert.Equal(t, "read", permissions[0].Code)
	})

	// Test GetUserPermissions with no permissions
	t.Run("GetUserPermissionsNoPermissions", func(t *testing.T) {
		permissions, err := service.GetUserPermissions(context.Background(), 2) // Non-existent user
		assert.NoError(t, err)
		assert.Len(t, permissions, 0)
	})
}

func ToJSON(t *testing.T, v interface{}, msg ...string) {
	t.Helper()
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Error(err)
	}
	if len(msg) == 0 {
		msg = append(msg, "")
	}

	t.Logf("%s => %s", msg[0], bytes)
}
