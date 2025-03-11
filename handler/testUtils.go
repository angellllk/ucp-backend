package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"sarp_backend/config"
	"sarp_backend/model"
	"sarp_backend/repository"
	"sarp_backend/service"
	"testing"
	"time"
)

const (
	testUsername = "test"
	testEmail    = "test@test.ro"
	testPassword = "test123."
)

func testConfig() *config.Config {
	return &config.Config{
		Version: "test",
		Dsn:     "samp:password@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true", // Modify credentials
		Port:    ":3000",
	}
}

func testRepository(t *testing.T) *repository.UserRepository {
	t.Helper()

	cfg := testConfig()

	ucpRepo, errRepo := repository.New(cfg.Dsn)
	if errRepo != nil {
		t.Fatalf("Error creating test repository: %v", errRepo)
		return nil
	}

	if _, err := ucpRepo.DB.Exec("CREATE DATABASE IF NOT EXISTS test_schema"); err != nil {
		t.Fatalf("Error creating test database: %v", err)
		return nil
	}
	if _, err := ucpRepo.DB.Exec("USE test_schema"); err != nil {
		t.Fatalf("Error using test database: %v", err)
		return nil
	}

	if _, err := ucpRepo.DB.Exec(accountsTable); err != nil {
		t.Fatalf("Error creating accounts table: %v", err)
		return nil
	}
	if _, err := ucpRepo.DB.Exec(blacklistTable); err != nil {
		t.Fatalf("Error creating blacklist table: %v", err)
		return nil
	}
	if _, err := ucpRepo.DB.Exec(charactersTable); err != nil {
		t.Fatalf("Error creating characters table: %v", err)
		return nil
	}
	if _, err := ucpRepo.DB.Exec(housesTable); err != nil {
		t.Fatalf("Error creating characters table: %v", err)
		return nil
	}

	tables := []string{"accounts", "blacklist", "characters", "houses"}

	for _, table := range tables {
		_, err := ucpRepo.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
		if err != nil {
			t.Fatalf("Error truncating table %s: %v", table, err)
		}
	}

	return ucpRepo
}

func testServer(us *service.UserService, as *service.MockAuthService, es *service.MockEmailService, cs *service.CharacterService, ls *service.MockLoggerService) *fiber.App {
	handler := New(us, cs, as, ls, es)

	app := fiber.New()

	app.Post("/register", func(ctx *fiber.Ctx) error {
		// Handle register
		return handler.Register(ctx)
	})

	app.Get("/confirm", func(ctx *fiber.Ctx) error {
		return handler.Confirm(ctx)
	})

	app.Post("/login", func(ctx *fiber.Ctx) error {
		// Handles login
		return handler.Login(ctx)
	})

	app.Post("/logout", func(ctx *fiber.Ctx) error {
		// Handles logout
		return handler.Logout(ctx)
	})

	app.Get("/check-auth", func(ctx *fiber.Ctx) error {
		// Returns if there's a session for the user
		return handler.CheckAuth(ctx)
	})

	app.Get("/get-data", func(ctx *fiber.Ctx) error {
		// Retrieve user stats
		return handler.GetStats(ctx)
	})

	app.Get("/get-staff", func(ctx *fiber.Ctx) error {
		return handler.GetStaff(ctx)
	})

	app.Post("/create-character", func(ctx *fiber.Ctx) error {
		return handler.CreateCharacter(ctx)
	})

	app.Get("/server-stats", func(ctx *fiber.Ctx) error {
		return handler.ServerStats(ctx)
	})

	restricted := app.Group("restricted")
	{
		restricted.Get("/check", func(ctx *fiber.Ctx) error {
			return handler.CheckAdmin(ctx)
		})

		// Restricted API for Administrators
		restricted.Get("/waiting-list", func(ctx *fiber.Ctx) error {
			return handler.WaitingList(ctx)
		})

		restricted.Post("/accept-character", func(ctx *fiber.Ctx) error {
			return handler.AcceptCharacter(ctx)
		})

		restricted.Post("/reject-character", func(ctx *fiber.Ctx) error {
			return handler.RejectCharacter(ctx)
		})

		restricted.Post("/fetch-character", func(ctx *fiber.Ctx) error {
			return handler.FetchCharacter(ctx)
		})

		restricted.Get("/ban-list", func(ctx *fiber.Ctx) error {
			return handler.BanList(ctx)
		})

		restricted.Post("/ban", func(ctx *fiber.Ctx) error {
			return handler.Ban(ctx)
		})

		restricted.Post("/unban", func(ctx *fiber.Ctx) error {
			return handler.Unban(ctx)
		})

		restricted.Post("/ajail", func(ctx *fiber.Ctx) error {
			return handler.Ajail(ctx)
		})
	}

	// Route for 404
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendString("test")
	})

	return app
}

func testCleanup(t *testing.T, repo *repository.UserRepository) {
	t.Helper()

	tables := []string{"accounts", "characters", "blacklist"}
	for _, table := range tables {
		sql := fmt.Sprintf("TRUNCATE TABLE %s", table)
		if _, err := repo.DB.Exec(sql); err != nil {
			t.Logf("Error truncating table %s: %v", table, err)
		}
	}

	if err := repo.DB.Close(); err != nil {
		t.Logf("Error closing database connection: %v", err)
	}
}

func registerAccount(t *testing.T, app *fiber.App) {
	t.Helper()

	resp := testSendRequest(t, app, http.MethodPost, "/register", model.RegisterAPI{
		Username: testUsername,
		Email:    testEmail,
		Password: testPassword,
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Unexpected response HTTP status")
}

func registerAndConfirmAccount(t *testing.T, app *fiber.App) {
	t.Helper()

	resp := testSendRequest(t, app, http.MethodPost, "/register", model.RegisterAPI{
		Username: testUsername,
		Email:    testEmail,
		Password: testPassword,
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Unexpected response HTTP status")

	ts := time.Now().Unix()
	target := fmt.Sprintf("/confirm?email=%s&token=%s&timestamp=%d", testEmail, service.GenerateToken(testEmail, ts), ts)
	resp = testSendRequest(t, app, http.MethodGet, target, nil)
	assert.Equal(t, http.StatusFound, resp.StatusCode, "Unexpected response HTTP status")
}

func createCharacter(t *testing.T, app *fiber.App) {
	t.Helper()

	resp := testSendRequest(t, app, http.MethodPost, "/create-character", model.CharacterDataAPI{
		Username:        testUsername,
		CharacterName:   "Test_Test",
		CharacterAge:    18,
		CharacterGender: 0,
		CharacterOrigin: "test",
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Unexpected response HTTP status")
}

func testSendRequest(t *testing.T, app *fiber.App, method, target string, body interface{}) *http.Response {
	t.Helper()

	var err error
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Error marshalling test body %v: %v", body, err)
		}
	}

	req := httptest.NewRequest(method, target, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Error sending test request for %s with body %s: %v", target, body, err)
	}

	return resp
}

const accountsTable = `
create table if not exists accounts 
(
    id                 int auto_increment 
        primary key,
    Username           varchar(250)                                                            null,
    Email              varchar(250)                                                            not null,
    NumePrenume        varchar(50)                                                             not null,
    NumeForum          varchar(200)                                                            not null,
    Password           varchar(129)                                                            null,
    Serial             varchar(128)                                                            not null,
    RegisterDate       varchar(36)                                                             null,
    LoginDate          int                           default 0                                 null,
    IP                 varchar(16)                   default 'n/a'                             null,
    secret_code        varchar(24)                                                             null,
    Characters         int                           default 0                                 not null,
    Tokens             float                         default 0                                 not null,
    Admin              int                           default 0                                 not null,
    Tester             int                           default 0                                 not null,
    DonateRank         int                           default 0                                 not null,
    DonateExpired      varchar(128)                  default '0000-00-00'                      not null,
    VoucherPhone       int                           default 0                                 not null,
    VoucherName        int                           default 0                                 not null,
    VoucherCK          int                           default 0                                 not null,
    VoucherForum       int                           default 0                                 not null,
    ConthorizeAdmin    int                           default 0                                 not null,
    ConthorizeTester   int                           default 0                                 not null,
    AcceptedBy         varchar(500)                  default ''                                not null,
    Ziua               int                           default 0                                 not null,
    Luna               varchar(50)                   default '0'                               not null,
    Anul               int                           default 0                                 not null,
    Ora                int                           default 0                                 not null,
    Minut              int                           default 0                                 not null,
    Accepted           int                           default 0                                 not null,
    Raspuns1           varchar(3000) charset utf8    default ''                                not null,
    Raspuns2           varchar(3000) charset utf8    default ''                                not null,
    Raspuns3           varchar(3000) charset utf8    default ''                                not null,
    Raspuns4           varchar(3000) charset utf8    default ''                                not null,
    Raspuns5           varchar(1000) charset utf8mb4 default ''                                not null,
    Motiv              varchar(500) charset utf8     default '0'                               not null,
    A1                 int                           default 0                                 not null,
    A2                 int                           default 0                                 not null,
    A3                 int                           default 0                                 not null,
    A4                 int                           default 0                                 not null,
    A5                 int                           default 0                                 not null,
    A6                 int                           default 0                                 not null,
    A7                 int                           default 0                                 not null,
    B1                 varchar(100)                  default ''                                not null,
    B2                 varchar(100)                  default ''                                not null,
    B3                 varchar(100)                  default ''                                not null,
    B4                 varchar(100)                  default ''                                not null,
    B5                 varchar(100)                  default ''                                not null,
    B6                 varchar(100)                  default ''                                not null,
    B7                 varchar(100)                  default ''                                not null,
    Activated          int                           default 0                                 not null,
    ConturiAcceptate   int                           default 0                                 not null,
    ConturiRefuzate    int                           default 0                                 not null,
    CaractereAcceptate int                           default 0                                 not null,
    CaractereRefuzate  int                           default 0                                 not null,
    Avatar             varchar(250)                  default 'https://i.imgur.com/LXXYhKw.png' not null,
    DescriereaMea      varchar(5000) charset utf8    default '-'                               not null
)
    charset = latin1;
`

const blacklistTable = `
create table if not exists blacklist
(
    ID       int auto_increment 
        primary key,
    IP       varchar(16)  default '0.0.0.0' null,
    Username varchar(24)                    null,
    BannedBy varchar(24)                    null,
    Reason   varchar(128)                   null,
    Date   varchar(36)                    null,
    perm     int          default 1         not null,
    Expire   varchar(250) default ''        null
)
    charset = latin1;
`

const charactersTable = `
create table if not exists characters
(
    ID               int auto_increment 
        primary key,
    Username         varchar(24)                                                                             null,
    ` + "`Character`" + `        varchar(24)                                                                             null,
    Created          int          default 0                                                                  null,
    Age              int          default 4                                                                  not null,
    Level            int          default 1                                                                  not null,
    Experience       int          default 0                                                                  not null,
    Gender           int          default 0                                                                  null,
    Origin           varchar(32)  default 'Nespecificat'                                                     null,
    Skin             int          default 299                                                                null,
    Status           int          default 0                                                                  null,
    PosX             float        default 0                                                                  null,
    PosY             float        default 0                                                                  null,
    PosZ             float        default 0                                                                  null,
    PosA             float        default 0                                                                  null,
    Interior         int          default 0                                                                  not null,
    World            int          default 0                                                                  null,
    Money            int          default 2500                                                               not null,
    BankMoney        int          default 2500                                                               null,
    Savings          int          default 0                                                                  null,
    JailTime         int          default 0                                                                  null,
    Muted            int          default 0                                                                  null,
    MuteTime         int          default 0                                                                  not null,
    CreateDate       varchar(250) default '0'                                                                null,
    LastLogin        int          default 0                                                                  null,
    Gun1             int          default 0                                                                  null,
    Gun2             int          default 0                                                                  null,
    Gun3             int          default 0                                                                  null,
    Gun4             int          default 0                                                                  null,
    Gun5             int          default 0                                                                  null,
    Gun6             int          default 0                                                                  null,
    Gun7             int          default 0                                                                  null,
    Gun8             int          default 0                                                                  null,
    Gun9             int          default 0                                                                  null,
    Gun10            int          default 0                                                                  null,
    Gun11            int          default 0                                                                  null,
    Gun12            int          default 0                                                                  null,
    Gun13            int          default 0                                                                  null,
    Ammo1            int          default 0                                                                  null,
    Ammo2            int          default 0                                                                  null,
    Ammo3            int          default 0                                                                  null,
    Ammo4            int          default 0                                                                  null,
    Ammo5            int          default 0                                                                  null,
    Ammo6            int          default 0                                                                  null,
    Ammo7            int          default 0                                                                  null,
    Ammo8            int          default 0                                                                  null,
    Ammo9            int          default 0                                                                  null,
    Ammo10           int          default 0                                                                  null,
    Ammo11           int          default 0                                                                  null,
    Ammo12           int          default 0                                                                  null,
    Ammo13           int          default 0                                                                  null,
    House            int          default -1                                                                 null,
    Business         int          default -1                                                                 null,
    Journey          int          default -1                                                                 not null,
    Phone            int          default 0                                                                  null,
    PlayingHours     int          default 0                                                                  null,
    Minutes          int          default 0                                                                  null,
    ArmorStatus      float        default 0                                                                  null,
    Entrance         int          default 0                                                                  null,
    Job              int          default 0                                                                  null,
    Faction          int          default -1                                                                 null,
    FactionRank      int          default 0                                                                  null,
    Prisoned         int          default 0                                                                  null,
    Injured          int          default 0                                                                  null,
    Health           float        default 100                                                                null,
    Warnings         int          default 0                                                                  null,
    Warn1            varchar(32)                                                                             null,
    Warn2            varchar(32)                                                                             null,
    MaskID           int          default 0                                                                  null,
    FactionMod       int          default 0                                                                  null,
    PropertyMod      int          default 0                                                                  not null,
    Capacity         int          default 35                                                                 null,
    AdminHide        int          default 0                                                                  null,
    SpawnPoint       int          default 0                                                                  not null,
    StopJob          int          default 0                                                                  not null,
    PayCheck         int          default 0                                                                  not null,
    Abandon          int          default 0                                                                  not null,
    Badge            int          default 0                                                                  not null,
    PhoneRingtone    int          default 0                                                                  not null,
    Wanteds          int          default 0                                                                  not null,
    Timeout          int          default 0                                                                  not null,
    SModel           int          default 0                                                                  not null,
    OnDuty           int          default 0                                                                  not null,
    TimeDuty         int          default 0                                                                  not null,
    pCarKey          int          default 9999                                                               not null,
    pDupKey          int          default 9999                                                               not null,
    Drugs            varchar(216) default '0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00' not null,
    DrugPerm         int          default 0                                                                  not null,
    Ingredients      int          default 0                                                                  not null,
    Addiction        float        default 0                                                                  null,
    FightStyle       int          default 0                                                                  not null,
    Talk             int          default 0                                                                  not null,
    Walk             int          default 0                                                                  not null,
    HouseSpawn       int          default -1                                                                 not null,
    Online           int          default 0                                                                  not null,
    Cell             int          default -1                                                                 not null,
    Swat             int          default 0                                                                  not null,
    GrantB           int          default -1                                                                 not null,
    Hire             int          default -1                                                                 not null,
    Unit             int          default -1                                                                 not null,
    GraffitiText     varchar(128)                                                                            null,
    GraffitiFont     int          default 0                                                                  not null,
    GraffitiType     int          default 0                                                                  not null,
    Clothes          varchar(128) default '-1|-1|-1|-1|-1'                                                   not null,
    SprayPermission  int          default 0                                                                  not null,
    LeoLicense       int          default 0                                                                  not null,
    MarijuanaLicense int          default 0                                                                  not null,
    Renting          int          default 0                                                                  not null,
    RentKey          int          default -1                                                                 not null,
    Biografie        varchar(5000) charset utf8                                                              null,
    CanPry           int          default 0                                                                  not null,
    PaperPerm        int          default 0                                                                  not null,
    FakeID           int          default 0                                                                  not null,
    FakeName         varchar(64)                                                                             null,
    FakeSign         varchar(32)                                                                             null,
    FakeAge          int          default 0                                                                  not null,
    FakeOrigin       varchar(32)                                                                             null,
    FakeSex          int          default 0                                                                  not null,
    FakeLicense      int          default 0                                                                  not null,
    FakeNameLicense  varchar(64)                                                                             null,
    FakeSignLicense  varchar(64)                                                                             null,
    FakeDriving      int          default 0                                                                  not null,
    FakeWeapon       int          default 0                                                                  not null,
    Channels         varchar(256) default '-1|-1|-1|-1|-1|-1|-1|-1|-1|-1'                                    not null,
    Slots            varchar(256) default '-1|-1|-1|-1|-1|-1|-1|-1|-1|-1'                                    not null,
    Cards1           int          default 0                                                                  not null,
    Cards2           int          default 0                                                                  not null,
    AcceptedBy       varchar(500)                                                                            null,
    AutoLights       int          default 0                                                                  not null,
    UseArmourDrug    int          default 0                                                                  not null,
    UseHealthDrug    int          default 0                                                                  not null,
    TimeBuy          int          default 0                                                                  not null
)
    charset = latin1
`

const housesTable = `
create table if not exists houses
(
    houseID           int auto_increment
        primary key,
    houseOwner        int          default 0                                                                    null,
    housePrice        int          default 0                                                                    null,
    houseAddress      varchar(32)                                                                               null,
    housePosX         float        default 0                                                                    null,
    housePosY         float        default 0                                                                    null,
    housePosZ         float        default 0                                                                    null,
    housePosA         float        default 0                                                                    null,
    houseIntX         float        default 0                                                                    null,
    houseIntY         float        default 0                                                                    null,
    houseIntZ         float        default 0                                                                    null,
    houseIntA         float        default 0                                                                    null,
    houseInterior     int          default 0                                                                    null,
    houseExterior     int          default 0                                                                    null,
    houseExteriorVW   int          default 0                                                                    null,
    houseLocked       int          default 0                                                                    null,
    houseWeapon1      int          default 0                                                                    null,
    houseAmmo1        int          default 0                                                                    null,
    houseWeapon2      int          default 0                                                                    null,
    houseAmmo2        int          default 0                                                                    null,
    houseWeapon3      int          default 0                                                                    null,
    houseAmmo3        int          default 0                                                                    null,
    houseWeapon4      int          default 0                                                                    null,
    houseAmmo4        int          default 0                                                                    null,
    houseWeapon5      int          default 0                                                                    null,
    houseAmmo5        int          default 0                                                                    null,
    houseWeapon6      int          default 0                                                                    null,
    houseAmmo6        int          default 0                                                                    null,
    houseWeapon7      int          default 0                                                                    null,
    houseAmmo7        int          default 0                                                                    null,
    houseWeapon8      int          default 0                                                                    null,
    houseAmmo8        int          default 0                                                                    null,
    houseWeapon9      int          default 0                                                                    null,
    houseAmmo9        int          default 0                                                                    null,
    houseWeapon10     int          default 0                                                                    null,
    houseAmmo10       int          default 0                                                                    null,
    houseWeapon11     int          default 0                                                                    not null,
    houseAmmo11       int          default 0                                                                    not null,
    houseWeapon12     int          default 0                                                                    not null,
    houseAmmo12       int          default 0                                                                    not null,
    houseWeapon13     int          default 0                                                                    not null,
    houseAmmo13       int          default 0                                                                    not null,
    houseWeapon14     int          default 0                                                                    not null,
    houseAmmo14       int          default 0                                                                    not null,
    houseWeapon15     int          default 0                                                                    not null,
    houseAmmo15       int          default 0                                                                    not null,
    houseWeapon16     int          default 0                                                                    not null,
    houseAmmo16       int          default 0                                                                    not null,
    houseWeapon17     int          default 0                                                                    not null,
    houseAmmo17       int          default 0                                                                    not null,
    houseWeapon18     int          default 0                                                                    not null,
    houseAmmo18       int          default 0                                                                    not null,
    houseWeapon19     int          default 0                                                                    not null,
    houseAmmo19       int          default 0                                                                    not null,
    houseWeapon20     int          default 0                                                                    not null,
    houseAmmo20       int          default 0                                                                    not null,
    houseMoney        int          default 0                                                                    null,
    housePrepare      int          default -1                                                                   not null,
    housePrepareDrugs int          default -1                                                                   not null,
    housePrepareTime  int          default -1                                                                   not null,
    housePrepareID    int          default -1                                                                   not null,
    houseDrugs        varchar(256) default '0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00|0.00\t' not null,
    houseRentable     int          default 0                                                                    not null,
    houseRentPrice    int          default 0                                                                    not null
)
    charset = latin1;

`
