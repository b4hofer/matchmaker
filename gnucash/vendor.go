package gnucash

import "database/sql"

type Vendor struct {
	DbVendor
}

type DbVendor struct {
	Guid        string
	Name        string
	Id          string
	Notes       string
	Currency    string
	Active      int
	TaxOverride int            `db:"tax_override"`
	AddrName    sql.NullString `db:"addr_name"`
	AddrAddr1   sql.NullString `db:"addr_addr1"`
	AddrAddr2   sql.NullString `db:"addr_addr2"`
	AddrAddr3   sql.NullString `db:"addr_addr3"`
	AddrAddr4   sql.NullString `db:"addr_addr4"`
	AddrPhone   sql.NullString `db:"addr_phone"`
	AddrFax     sql.NullString `db:"addr_fax"`
	AddrEmail   sql.NullString `db:"addr_email"`
	Terms       sql.NullString `db:"terms"`
	TaxInc      sql.NullString `db:"tax_inc"`
	TaxTable    sql.NullString `db:"tax_table"`
}
