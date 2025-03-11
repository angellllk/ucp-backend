package repository

import (
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"sort"
)

type UserRepository struct {
	DB *sqlx.DB
}

func New(dsn string) (*UserRepository, error) {
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &UserRepository{DB: db}, nil
}

func withTransaction(db *sqlx.DB, txFunc func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	if err = txFunc(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}
	return tx.Commit()
}

func (r *UserRepository) valueExists(query string, args ...interface{}) (bool, error) {
	var count int
	if err := r.DB.Get(&count, query, args...); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) Create(data *UserDB) error {
	existsQuery := "SELECT COUNT(*) FROM accounts WHERE Username = ? OR Email = ?"
	exists, err := r.valueExists(existsQuery, data.Username, data.Email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("account already exists")
	}

	insertQuery := `
		INSERT INTO accounts (Username, Email, NumeForum, NumePrenume, Password, Serial, RegisterDate, Characters, Activated, Admin, secret_code, Accepted, CaractereAcceptate)
		VALUES (?, ?, ?, 'N/A', ?, '000000', ?, 0, 0, 0, '000000', 2, 0)
	`

	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		result, errTx := tx.Exec(insertQuery, data.Username, data.Email, data.Username, data.Password, data.RegisterDate)
		if errTx != nil {
			return errTx
		}
		rows, errRows := result.RowsAffected()
		if errRows != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) Activate(email string) error {
	query := "UPDATE accounts SET Activated = 2 WHERE Email = ? AND Activated = 0"

	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		result, errTx := tx.Exec(query, email)
		if errTx != nil {
			return errTx
		}
		rows, errRows := result.RowsAffected()
		if errRows != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) CheckActivation(name string) (bool, error) {
	var activation int
	query := "SELECT Activated FROM accounts WHERE Username = ?"
	if err := r.DB.Get(&activation, query, name); err != nil {
		return false, err
	}

	return activation == 2, nil
}

func (r *UserRepository) Verify(data *UserDB) error {
	var name string
	query := "SELECT Username FROM accounts WHERE Username = ? AND Password = ?"

	return r.DB.Get(&name, query, data.Username, data.Password)
}

func (r *UserRepository) UpdatePassword(email, password string) error {
	query := "UPDATE accounts SET Password = ? WHERE Email = ?"

	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		result, errTx := tx.Exec(query, password, email)
		if errTx != nil {
			return errTx
		}
		rows, errRows := result.RowsAffected()
		if errRows != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) Fetch(name string, email string) (bool, error) {
	var fetch string
	query := "SELECT Username FROM accounts WHERE Username = ? OR Email = ?"
	if err := r.DB.Get(&fetch, query, name, email); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
	}

	return len(fetch) != 0, nil
}

func (r *UserRepository) FetchStats(name string) (*GetStatsDB, error) {
	var ret GetStatsDB
	query := "SELECT Admin, Tester, DonateRank, `Characters`, LoginDate FROM accounts WHERE Username = ?"
	if err := r.DB.Get(&ret, query, name); err != nil {
		return nil, err
	}

	var characters []CharacterStatsDB
	query = "SELECT `Character`, Created, Level, PlayingHours FROM characters WHERE Username = ? AND Created >= 0"
	if err := r.DB.Select(&characters, query, name); err != nil {
		return nil, err
	}

	ret.CharactersData = characters
	ret.Characters = len(characters)
	ret.Username = name
	return &ret, nil
}

func (r *UserRepository) FetchMail(name string) (string, error) {
	var mail string
	query := "SELECT Email FROM accounts WHERE Username = ?"
	if err := r.DB.Get(&mail, query, name); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		return "", err
	}

	return mail, nil
}

func (r *UserRepository) FetchStaff() ([]GetStatsDB, error) {
	var ret []GetStatsDB
	query := "SELECT Username, Admin, Tester FROM accounts WHERE Admin > 0 OR Tester > 0"
	if err := r.DB.Select(&ret, query); err != nil {
		return nil, err
	}

	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Tester < ret[j].Tester {
			return true
		}
		if ret[i].Tester > ret[j].Tester {
			return false
		}

		// Dacă ambele sunt Admin, sortăm descrescător după nivelul Admin
		if ret[i].Admin != ret[j].Admin {
			return ret[i].Admin > ret[j].Admin
		}

		// Dacă sunt ambii Tester sau Admin cu același nivel, sortăm alfabetic după Username
		return ret[i].Username < ret[j].Username
	})

	return ret, nil
}

func (r *UserRepository) FetchServerStats() (*GetServerStatsDB, error) {
	var ret GetServerStatsDB
	query := "SELECT COUNT(*) FROM characters WHERE Online = 1"
	if err := r.DB.Get(&ret.Online, query); err != nil {
		return nil, err
	}

	query = "SELECT COUNT(*) FROM blacklist WHERE perm = 1 OR Expire >= NOW()"
	if err := r.DB.Get(&ret.Bans, query); err != nil {
		return nil, err
	}

	query = "SELECT COUNT(*) FROM houses"
	if err := r.DB.Get(&ret.Houses, query); err != nil {
		return nil, err
	}

	query = "SELECT COUNT(*) FROM accounts WHERE Admin > 0 OR Tester > 0"
	if err := r.DB.Get(&ret.Staff, query); err != nil {
		return nil, err
	}

	query = "SELECT COUNT(*) FROM  accounts WHERE Activated = 2"
	if err := r.DB.Get(&ret.Accounts, query); err != nil {
		return nil, err
	}

	query = "SELECT COUNT(*) FROM characters WHERE Created > 0"
	if err := r.DB.Get(&ret.Characters, query); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *UserRepository) FetchTesterLevel(name string) (bool, error) {
	var testerLevel int
	query := "SELECT Tester FROM accounts WHERE Username = ?"
	if err := r.DB.Get(&testerLevel, query, name); err != nil {
		return false, err
	}

	return testerLevel > 0, nil
}

func (r *UserRepository) FetchAdminLevel(name string) (bool, error) {
	var adminLevel int
	query := "SELECT Admin FROM accounts WHERE Username = ?"
	if err := r.DB.Get(&adminLevel, query, name); err != nil {
		return false, err
	}

	return adminLevel > 0, nil
}

func (r *UserRepository) CreateCharacter(data *CharacterDB) error {
	var ret string
	query := "SELECT `Character` FROM characters WHERE `Character` = ?"
	if err := r.DB.Get(&ret, query, data.Character); err != nil && err.Error() != "sql: no rows in result set" {
		return err
	}
	if len(ret) > 0 {
		return errors.New(fmt.Sprintf("character with name %s already exists", data.Character))
	}

	insertQuery := "INSERT INTO characters(Username, `Character`, Level, Created, Age, Gender, Origin, Skin, Status, AcceptedBy) " +
		"VALUES (?, ?, 1, 0, ?, ?, ?, ?, 0, 'N/A');"

	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		result, errTx := tx.Exec(insertQuery, data.Username, data.Character, data.Age, data.Gender, data.Origin, data.Skin)
		if errTx != nil {
			return errTx
		}
		rows, errRows := result.RowsAffected()
		if errRows != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) FetchWaitingCharacters() ([]CharacterDB, error) {
	var characters []CharacterDB
	query := "SELECT Username, `Character`, Age, Gender, Origin FROM characters WHERE Created = 0"

	if err := r.DB.Select(&characters, query); err != nil {
		return nil, err
	}
	return characters, nil
}

func (r *UserRepository) AcceptCharacter(username, characterName, acceptedBy string) error {
	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		charCountQuery := "SELECT Characters FROM accounts WHERE Username = ?"
		var charCount int
		if err := tx.Get(&charCount, charCountQuery, username); err != nil {
			return err
		}

		updateCharQuery := "UPDATE characters SET Created = 1, Status = 1 WHERE `Character` = ?"
		result, errTx := tx.Exec(updateCharQuery, characterName)
		if errTx != nil {
			return errTx
		}
		rows, errRows := result.RowsAffected()
		if errRows != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}

		updateUserQuery := "UPDATE accounts SET Characters = ?, AcceptedBy = ?, Accepted = 2 WHERE Username = ?"
		result, errTx = tx.Exec(updateUserQuery, charCount+1, acceptedBy, username)
		if errTx != nil {
			return errTx
		}
		rows, errRows = result.RowsAffected()
		if errRows != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) DeclineCharacter(characterName string) error {
	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		query := "DELETE FROM characters WHERE `Character` = ? AND Status = 0"
		result, err := tx.Exec(query, characterName)
		if err != nil {
			return err
		}
		rows, errRows := result.RowsAffected()
		if errRows != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) FetchCharacter(character string) (*CharacterDB, error) {
	var data CharacterDB
	query := "SELECT Username, `Character` FROM characters WHERE `Character` = ?"
	if err := r.DB.Get(&data, query, character); err != nil {
		return nil, err
	}
	if data.Username == "" || data.Character == "" {
		return nil, errors.New("unexpected empty values fetched")
	}

	return &data, nil
}

func (r *UserRepository) CheckForBan(name string) (bool, error) {
	var ret string
	query := "SELECT Username FROM blacklist WHERE Username = ? AND Expire >= NOW()"
	if err := r.DB.Get(&ret, query, name); err != nil && err.Error() != "sql: no rows in result set" {
		return false, err
	}

	return len(ret) > 0, nil
}

func (r *UserRepository) AddBan(data *BlacklistDB) error {
	selectIP := "SELECT IP FROM accounts WHERE Username = ?"
	if err := r.DB.Get(&data.IP, selectIP, data.Username); err != nil {
		return err
	}
	if data.IP == "" {
		return errors.New("can't get IP")
	}

	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		query := "INSERT INTO `blacklist` (`IP`,`Username`,`BannedBy`,`Reason`,`perm`, `Date`, `expire`) " +
			"VALUES (?, ?, ?, ?, 0, ?, DATE_ADD(NOW(),INTERVAL ? DAY))"

		result, err := tx.Exec(query, data.IP, data.Username, data.BannedBy, data.Reason, data.Date, data.Expire)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) FetchBans() ([]BlacklistDB, error) {
	var bans []BlacklistDB
	queryBans := "SELECT Username, BannedBy, Reason, Expire FROM blacklist WHERE Expire >= NOW()"
	if err := r.DB.Select(&bans, queryBans); err != nil {
		return nil, err
	}

	for _, ban := range bans {
		var name string
		queryChar := "SELECT `Character` FROM characters WHERE Username = ? AND Status = 1"
		if err := r.DB.Select(&name, queryChar); err != nil {
			return nil, err
		}
		ban.Characters = append(ban.Characters, CharacterDB{Character: name})
	}

	return bans, nil
}

func (r *UserRepository) Unban(name string) error {
	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		query := "UPDATE blacklist SET expire = DATE_SUB(NOW(), INTERVAL 1 SECOND) WHERE Username = ?"
		result, err := tx.Exec(query, name)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) Ajail(data *AjailDB) error {
	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		query := "UPDATE characters SET JailTime = ?, Prisoned = ? WHERE `Character` = ?"
		result, err := tx.Exec(query, data.JailTime, data.Prisoned, data.Character)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows == 0 {
			return errors.New("no rows affected, expected one")
		}
		return nil
	})
}

func (r *UserRepository) FetchLogs(logsType string) ([]map[string]interface{}, error) {
	q := fmt.Sprintf("SELECT * FROM %s ORDER BY ID DESC LIMIT 100", logsType)
	rows, err := r.DB.Queryx(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err = rows.MapScan(row); err != nil {
			return nil, err
		}
		if row["IP"] != "" {
			row["IP"] = "-"
		}
		logs = append(logs, row)
	}

	return logs, nil
}

func (r *UserRepository) DeleteExp() error {
	return withTransaction(r.DB, func(tx *sqlx.Tx) error {
		query := "DELETE FROM characters WHERE Created = -1"
		_, err := tx.Exec(query)
		if err != nil {
			return err
		}
		return nil
	})
}
