package gnucash

import (
	"database/sql"
	"log"
	"math/big"
)

// Represents an invoice, bill, expense voucher or credit note.
type Invoice struct {
	book         *Book
	IsCreditNote bool
	Entries      []*Entry
	DbInvoice
}

type DbInvoice struct {
	Guid           string
	Id             string
	DateOpened     sql.NullString `db:"date_opened"`
	DatePosted     sql.NullString `db:"date_posted"`
	Notes          string
	Active         int
	Currency       string
	OwnerType      sql.NullInt32  `db:"owner_type"`
	OwnerGuid      sql.NullString `db:"owner_guid"`
	Terms          sql.NullString
	BillingId      sql.NullString `db:"billing_id"`
	PostTxn        sql.NullString `db:"post_txn"`
	PostLot        sql.NullString `db:"post_lot"`
	PostAcc        sql.NullString `db:"post_acc"`
	BilltoType     sql.NullInt32  `db:"billto_type"`
	BilltoGuid     sql.NullString `db:"billto_guid"`
	ChargeAmtNum   sql.NullInt64  `db:"charge_amt_num"`
	ChargeAmtDenom sql.NullInt64  `db:"charge_amt_denom"`
}

func (i *Invoice) GetOwnerType() OwnerType {
	if !i.OwnerType.Valid {
		return OwnerTypeUndefined
	}

	return OwnerType(i.OwnerType.Int32)
}

func (i *Invoice) GetVendor() *Vendor {
	if i.GetOwnerType() != OwnerTypeVendor || !i.OwnerGuid.Valid {
		return nil
	}

	return i.book.GetVendorByGUID(i.OwnerGuid.String)
}

func (i *Invoice) GetEmployee() *Employee {
	if i.GetOwnerType() != OwnerTypeEmployee || !i.OwnerGuid.Valid {
		return nil
	}

	return i.book.GetEmployeeByGUID(i.OwnerGuid.String)
}

func (i *Invoice) Write() {
	query := `INSERT OR REPLACE INTO invoices ("guid", "id", "date_opened", "date_posted",
			"notes", "active", "currency", "owner_type", "owner_guid", "terms",
			"billing_id", "post_txn", "post_lot", "post_acc", "billto_type",
			"billto_guid", "charge_amt_num", "charge_amt_denom")
		VALUES (:guid, :id, :date_opened, :date_posted, :notes, :active,
			:currency, :owner_type, :owner_guid, :terms, :billing_id, :post_txn,
			:post_lot, :post_acc, :billto_type, :billto_guid, :charge_amt_num,
			:charge_amt_denom)`
	_, err := i.book.DB.NamedExec(query, i.DbInvoice)
	if err != nil {
		log.Fatal(err)
	}
}

func (i *Invoice) AddEntry(e *Entry) {
	if e.Guid == "" {
		e.Guid = NewGuid()
	}

	e.book = i.book

	switch i.GetOwnerType() {
	case OwnerTypeCustomer:
		e.Invoice.Valid = true
		e.Invoice.String = i.Guid
	case OwnerTypeVendor:
		fallthrough
	case OwnerTypeEmployee:
		e.Bill.Valid = true
		e.Bill.String = i.Guid
	default:
		log.Fatalf("Unimplemented invoice type %d in AddEntry()\n",
			i.GetOwnerType())
	}

	e.Write()

	i.book.entries = append(i.book.entries, e)
	i.Entries = append(i.Entries, e)
}

func (i *Invoice) GetTotal() *big.Rat {
	tot := big.NewRat(0, 1)

	for _, e := range i.Entries {
		tot.Add(tot, e.GetSubtotal())
	}

	return tot
}

// Post invoice to account a
func (i *Invoice) Post(a *Account) {
	if i.DatePosted.Valid {
		log.Fatalf("Could not post invoice %s because it is already posted\n",
			i.Guid)
	}

	// Create lot for txns
	lot := &Lot{
		DbLot: DbLot{
			AccountGuid: sql.NullString{a.Guid, true},
			IsClosed:    -1, // this is a cache value, -1 means cache invalid
		},
	}
	a.AddLot(lot)

	lot.SetInvoice(i)

	desc := "ERROR"
	switch i.GetOwnerType() {
	case OwnerTypeVendor:
		desc = i.GetVendor().Name
	case OwnerTypeEmployee:
		desc = i.GetEmployee().AddrName.String
	}

	// Create txn
	txn := &Transaction{
		DbTransaction: DbTransaction{
			CurrencyGuid: i.Currency,
			Num:          i.Id,
			EnterDate:    sql.NullString{GetCurrentTimeString(), true},
			PostDate:     sql.NullString{GetCurrentTimeString(), true},
			Description:  sql.NullString{desc, true},
		},
	}
	i.book.AddTransaction(txn)

	txn.SetType(TransactionTypeInvoice)
	txn.SetReadOnly(true)

	// Create splits (accumulate per dest. account)
	splits := []*Split{}
	for _, e := range i.Entries {
		// Attempt to find existing split for dest. account
		found := false
		for _, s := range splits {
			var destAcc string
			if e.BAcct.Valid {
				destAcc = e.BAcct.String
			} else if e.IAcct.Valid {
				destAcc = e.IAcct.String
			} else {
				log.Fatalf("Entry %s has neither b_acct nor i_acct set", e.Guid)
			}

			if s.AccountGuid == destAcc {
				val := big.NewRat(s.ValueNum, s.ValueDenom)
				val.Add(val, e.GetSubtotal())
				s.ValueNum = val.Num().Int64()
				s.ValueDenom = val.Denom().Int64()
				s.QuantityNum = s.ValueNum
				s.QuantityDenom = s.ValueDenom
				found = true
				break
			}
		}

		// If no existing split has been found, create a new one
		if !found {
			var acc *Account
			var action string
			if e.BAcct.Valid {
				action = "Bill"
				acc = i.book.GetAccountByGUID(e.BAcct.String)
			} else if e.IAcct.Valid {
				action = "Invoice"
				acc = i.book.GetAccountByGUID(e.IAcct.String)
			}

			if acc == nil {
				log.Fatalf("Entry %s has no destination account (or dest. account GUID is invalid)")
			}

			val := e.GetSubtotal()

			s := &Split{
				Account: acc,
				DbSplit: DbSplit{
					Action:         action,
					ReconcileState: "n",
					ValueNum:       val.Num().Int64(),
					ValueDenom:     val.Denom().Int64(),
					QuantityNum:    val.Num().Int64(),
					QuantityDenom:  val.Denom().Int64(),
					LotGuid:        sql.NullString{lot.Guid, true},
				},
			}
			splits = append(splits, s)
		}
	}

	// Create A/P split with total amount
	var action string
	switch i.GetOwnerType() {
	case OwnerTypeVendor:
		action = "Bill"
	case OwnerTypeCustomer:
		action = "Invoice"
	case OwnerTypeEmployee:
		action = "Credit Note"
	default:
		log.Fatalf("Invoice posting for owner type %d not implemented\n", i.GetOwnerType())
	}

	val := i.GetTotal()
	val.Neg(val)

	s := &Split{
		Account: a,
		DbSplit: DbSplit{
			Action:         action,
			ReconcileState: "n",
			ValueNum:       val.Num().Int64(),
			ValueDenom:     val.Denom().Int64(),
			QuantityNum:    val.Num().Int64(),
			QuantityDenom:  val.Denom().Int64(),
			LotGuid:        sql.NullString{lot.Guid, true},
		},
	}
	txn.AddSplit(s)

	// Add generated (and accumulated) splits to txn
	for _, s := range splits {
		txn.AddSplit(s)
	}

	// Set post date, acc, txn and lot
	i.DatePosted = sql.NullString{GetCurrentTimeString(), true}
	i.PostAcc = sql.NullString{a.Guid, true}
	i.PostTxn = sql.NullString{txn.Guid, true}
	i.PostLot = sql.NullString{lot.Guid, true}
	i.Write()
}

func (i *Invoice) GetPostLot() *Lot {
	if !i.PostLot.Valid {
		return nil
	}

	return i.book.GetLotByGUID(i.PostLot.String)
}

func (i *Invoice) AssignPayment(s *Split) {
	// Assign invoice post lot to split
	lot := i.GetPostLot()
	if lot == nil {
		log.Fatalf("could not get post lot for invoice %s\n", i.Guid)
	}

	s.LotGuid = sql.NullString{lot.Guid, true}
	s.Write()

	// Set txn type to payment
	s.Transaction.SetType(TransactionTypePayment)

	// Set action for all splits in txn to payment
	for _, ts := range s.Transaction.Splits {
		ts.Action = "Payment"
		ts.Write()
	}
}
