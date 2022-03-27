package gnucash

import (
	"database/sql"
	"log"
)

type Transaction struct {
	book *Book
	DbTransaction
	Splits []*Split
}

type DbTransaction struct {
	Guid         string
	CurrencyGuid string `db:"currency_guid"`
	Num          string
	PostDate     sql.NullString `db:"post_date"`
	EnterDate    sql.NullString `db:"enter_date"`
	Description  sql.NullString
}

type TransactionType int

const (
	TransactionTypeInvoice TransactionType = iota
	TransactionTypePayment
)

func (t TransactionType) String() string {
	switch t {
	case TransactionTypeInvoice:
		return "I"
	case TransactionTypePayment:
		return "P"
	default:
		log.Fatal("Invalid TransactionType value in String()")
	}

	return ""
}

func (t *Transaction) write() {
	query := `INSERT OR REPLACE INTO "transactions" ("guid", "currency_guid",
			"num", "post_date", "enter_date", "description")
		VALUES (:guid, :currency_guid, :num, :post_date, :enter_date,
			:description)`
	_, err := t.book.DB.NamedExec(query, t.DbTransaction)
	if err != nil {
		log.Fatal(err)
	}
}

func (t *Transaction) GetCurrency() *Commodity {
	c := t.book.GetCommodityByGUID(t.CurrencyGuid)
	if c == nil {
		log.Fatalf("Currency %s for transaction %s not found\n",
			t.CurrencyGuid, t.Guid)
	}

	return c
}

func (t *Transaction) AddSplit(s *Split) {
	if s.Guid == "" {
		s.Guid = NewGuid()
	}

	s.book = t.book
	s.TxGuid = t.Guid
	s.AccountGuid = s.Account.Guid
	s.Write()
	t.book.splits = append(t.book.splits, s)
	t.Splits = append(t.Splits, s)
	s.Transaction = t
}

func (t *Transaction) SetType(tp TransactionType) {
	slot := t.book.getSlotForObjByName(t.Guid, "trans-txn-type")
	if slot == nil {
		// If slot doesn't exist, create it
		slot := &Slot{
			DbSlot: DbSlot{
				ObjGuid:   t.Guid,
				Name:      "trans-txn-type",
				SlotType:  int(SlotTypeString),
				StringVal: sql.NullString{tp.String(), true},
			},
		}
		t.book.AddSlot(slot)
	} else {
		slot.StringVal = sql.NullString{tp.String(), true}
		slot.Write()
	}
}

func (t *Transaction) SetReadOnly(ro bool) {
	slot := t.book.getSlotForObjByName(t.Guid, "trans-read-only")
	if ro {
		if slot == nil {
			slot := &Slot{
				DbSlot: DbSlot{
					ObjGuid:   t.Guid,
					Name:      "trans-read-only",
					SlotType:  int(SlotTypeString),
					StringVal: sql.NullString{"Generated from an invoice. Try unposting the invoice.", true},
				},
			}
			t.book.AddSlot(slot)
		} else {
			if slot.SlotType != int(SlotTypeString) {
				log.Fatal("transaction-read-only slot found, but it's not of string type")
			}

			slot.StringVal = sql.NullString{"Generated from an invoice. Try unposting the invoice.", true}
			slot.Write()
		}
	} else {
		if slot != nil {
			t.book.RemoveSlot(slot)
		}
	}
}
