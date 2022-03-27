package gnucash

import (
	"database/sql"
	"log"
)

type Employee struct {
	book *Book
	DbEmployee
}

type DbEmployee struct {
	Guid         string
	Username     string
	Id           string
	Language     string
	Acl          string
	Active       int
	Currency     string
	CCardGuid    sql.NullString `db:"ccard_guid"`
	WorkdayNum   sql.NullInt64  `db:"workday_num"`
	WorkdayDenom sql.NullInt64  `db:"workday_denom"`
	RateNum      sql.NullInt64  `db:"rate_num"`
	RateDenom    sql.NullInt64  `db:"rate_denom"`
	AddrName     sql.NullString `db:"addr_name"`
	AddrAddr1    sql.NullString `db:"addr_addr1"`
	AddrAddr2    sql.NullString `db:"addr_addr2"`
	AddrAddr3    sql.NullString `db:"addr_addr3"`
	AddrAddr4    sql.NullString `db:"addr_addr4"`
	AddrPhone    sql.NullString `db:"addr_phone"`
	AddrFax      sql.NullString `db:"addr_fax"`
	AddrEmail    sql.NullString `db:"addr_email"`
}

func (e *Employee) write() {
	query := `INSERT OR REPLACE INTO "employees" ("guid", "username", "id",
			"language", "acl", "active", "currency", "ccard_guid",
			"workday_num", "workday_denom", "rate_num", "rate_denom",
			"addr_name", "addr_addr1", "addr_addr2", "addr_addr3", "addr_addr4",
			"addr_phone", "addr_fax", "addr_email")
		VALUES (:guid, :username, :id, :language, :acl, :active, :currency,
			:ccard_guid, :workday_num, :workday_denom, :rate_num, :rate_denom,
			:addr_name, :addr_addr1, :addr_addr2, :addr_addr3, :addr_addr4,
			:addr_phone, :addr_fax, :addr_email)`

	_, err := e.book.DB.NamedExec(query, e)
	if err != nil {
		log.Fatal(err)
	}
}
