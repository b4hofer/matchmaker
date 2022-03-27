package gnucash

import (
	"database/sql"
	"strings"
)

type Account struct {
	book *Book
	DbAccount
	Parent   *Account
	Children []*Account
	Splits   []*Split
}

type DbAccount struct {
	Guid          string
	Name          sql.NullString
	Type          sql.NullString `db:"account_type"`
	CommodityGuid sql.NullString `db:"commodity_guid"`
	CommodityScu  int            `db:"commodity_scu"`
	NonStdScu     int            `db:"non_std_scu"`
	ParentGuid    sql.NullString `db:"parent_guid"`
	Code          sql.NullString
	Description   sql.NullString
	Hidden        int
	Placeholder   int
}

func (a *Account) GetAccountByPath(path string) *Account {
	splt := strings.Split(path, ":")
	for _, child := range a.Children {
		if child.Name.String == splt[0] {
			if len(splt) > 1 {
				return child.GetAccountByPath(strings.Join(splt[1:], ":"))
			} else {
				return child
			}
		}
	}

	return nil
}

func (a *Account) AddLot(l *Lot) {
	if l.Guid == "" {
		l.Guid = NewGuid()
	}

	l.book = a.book
	a.book.lots = append(a.book.lots, l)
	l.write()
}
