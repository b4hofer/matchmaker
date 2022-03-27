package gnucash

import (
	"database/sql"
	"log"
)

type Split struct {
	book *Book
	DbSplit
	Account     *Account
	Transaction *Transaction
}

type DbSplit struct {
	Guid           string
	TxGuid         string `db:"tx_guid"`
	AccountGuid    string `db:"account_guid"`
	Memo           string
	Action         string
	ReconcileState string         `db:"reconcile_state"`
	ReconcileDate  sql.NullString `db:"reconcile_date"`
	ValueNum       int64          `db:"value_num"`
	ValueDenom     int64          `db:"value_denom"`
	QuantityNum    int64          `db:"quantity_num"`
	QuantityDenom  int64          `db:"quantity_denom"`
	LotGuid        sql.NullString `db:"lot_guid"`
}

func (s *Split) Write() {
	query := `INSERT OR REPLACE INTO "splits" ("guid", "tx_guid",
			"account_guid", "memo", "action", "reconcile_state",
			"reconcile_date", "value_num", "value_denom", "quantity_num",
			"quantity_denom", "lot_guid")
		VALUES (:guid, :tx_guid, :account_guid, :memo, :action,
			:reconcile_state, :reconcile_date, :value_num, :value_denom,
			:quantity_num, :quantity_denom, :lot_guid)`
	_, err := s.book.DB.NamedExec(query, s.DbSplit)
	if err != nil {
		log.Fatal(err)
	}
}
