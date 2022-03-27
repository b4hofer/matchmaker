package gnucash

import (
	"database/sql"
	"log"
)

type Lot struct {
	book *Book
	DbLot
}

type DbLot struct {
	Guid        string
	AccountGuid sql.NullString `db:"account_guid"`
	IsClosed    int            `db:"is_closed"`
}

func (l *Lot) write() {
	query := `INSERT OR REPLACE INTO "lots" ("guid", "account_guid", "is_closed")
		VALUES (:guid, :account_guid, :is_closed)`

	_, err := l.book.DB.NamedExec(query, l.DbLot)
	if err != nil {
		log.Fatal(err)
	}
}

func (l *Lot) GetAccount() *Account {
	if !l.AccountGuid.Valid {
		return nil
	}

	return l.book.GetAccountByGUID(l.AccountGuid.String)
}

func (l *Lot) SetInvoice(i *Invoice) {
	pSlot := l.book.getSlotForObjByName(l.Guid, "gncInvoice")
	if pSlot == nil {
		// If slot doesn't exist, create it
		pSlot = &Slot{
			DbSlot: DbSlot{
				ObjGuid:  l.Guid,
				Name:     "gncInvoice",
				SlotType: int(SlotTypeFrame),
				GuidVal:  sql.NullString{NewGuid(), true},
			},
		}
		l.book.AddSlot(pSlot)
	}

	var cSlot *Slot
	for _, c := range pSlot.GetChildren() {
		if c.Name == "gncInvoice/invoice-guid" {
			cSlot = c
			break
		}
	}

	if cSlot == nil {
		// If child slot doesn't exist, create it
		cSlot = &Slot{
			DbSlot: DbSlot{
				Name:     "gncInvoice/invoice-guid",
				SlotType: int(SlotTypeGuid),
				GuidVal:  sql.NullString{i.Guid, true},
			},
		}
		pSlot.AddChild(cSlot)
	} else {
		cSlot.GuidVal = sql.NullString{i.Guid, true}
		cSlot.Write()
	}
}
