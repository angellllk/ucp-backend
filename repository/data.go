package repository

type UserDB struct {
	Username      string `db:"Username"`
	Email         string `db:"Email"`
	Password      string `db:"Password"`
	RegisterDate  string `db:"RegisterDate"`
	LoginDate     string `db:"LoginDate"`
	IP            string `db:"IP"`
	Characters    string `db:"Characters"`
	Admin         int    `db:"Admin"`
	Tester        int    `db:"Tester"`
	DonateRank    int    `db:"DonateRank"`
	DonateExpired int    `db:"DonateExpired"`
}

type CharacterDB struct {
	Username     string `db:"Username"`
	Character    string `db:"Character"`
	Age          int    `db:"Age"`
	Level        int    `db:"Level"`
	Gender       int    `db:"Gender"`
	Origin       string `db:"Origin"`
	Skin         int    `db:"Skin"`
	PlayingHours int    `db:"PlayingHours"`
}

type CharacterStatsDB struct {
	Name         string `json:"character_name" db:"Character"`
	Created      int    `json:"character_status" db:"Created"`
	Level        int    `json:"character_level" db:"Level"`
	PlayingHours int    `json:"playing_hours" db:"PlayingHours"`
}

type GetStatsDB struct {
	Username       string `db:"Username"`
	Admin          int    `db:"Admin"`
	Tester         int    `db:"Tester"`
	DonateRank     int    `db:"DonateRank"`
	Characters     int    `db:"Characters"`
	LoginDate      int64  `db:"LoginDate"`
	CharactersData []CharacterStatsDB
}

type GetServerStatsDB struct {
	Online     int
	Bans       int
	Houses     int
	Staff      int
	Accounts   int
	Characters int
}

type BlacklistDB struct {
	IP         string `db:"IP"`
	Username   string `db:"Username"`
	BannedBy   string `db:"BannedBy"`
	Reason     string `db:"Reason"`
	Date       string `db:"Date"`
	Expire     uint   `db:"Expire"`
	Characters []CharacterDB
}

type AjailDB struct {
	Character string `db:"Character"`
	Prisoned  int    `db:"Prisoned"`
	JailTime  int    `db:"JailTime"`
}
