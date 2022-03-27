package gnucash

import "database/sql"

type Commodity struct {
	DbCommodity
}

type DbCommodity struct {
	Guid        string
	Namespace   string
	Mnemonic    string
	FullName    sql.NullString
	Cusip       sql.NullString
	Fraction    int
	QuoteFlag   int            `db:"quote_flag"`
	QuoteSource sql.NullString `db:"quote_source"`
	QuoteTz     sql.NullString `db:"quote_tz"`
}
