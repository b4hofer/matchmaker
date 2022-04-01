package gnucash

import (
	"database/sql"
	"log"
	"regexp"

	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

type Book struct {
	DB          *sqlx.DB
	RootAccount *Account
	Accounts    []*Account
	DbBook
	slots        []*Slot
	lots         []*Lot
	commodities  []*Commodity
	vendors      []*Vendor
	employees    []*Employee
	invoices     []*Invoice
	entries      []*Entry
	transactions []*Transaction
	splits       []*Split
}

type DbBook struct {
	Guid             string
	RootAccountGuid  string `db:"root_account_guid"`
	RootTemplateGuid string `db:"root_template_guid"`
}

func OpenBookFromSQLite(path string) (*Book, error) {
	db, _ := sqlx.Open("sqlite3", path)

	// Force connection to report potential errors
	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	var dbBook DbBook
	err = db.Get(&dbBook, "SELECT * FROM books")
	if err != nil {
		log.Fatal(err)
	}

	book := &Book{
		DB:     db,
		DbBook: dbBook,
	}

	// Load accounts
	as := []DbAccount{}
	err = db.Select(&as, "SELECT * FROM accounts")
	if err != nil {
		log.Fatal(err)
	}

	for _, a := range as {
		acc := &Account{
			book:      book,
			DbAccount: a,
			Parent:    nil,
		}

		book.Accounts = append(book.Accounts, acc)

		if a.Guid == dbBook.RootAccountGuid {
			book.RootAccount = acc
		}
	}

	// Resolve parent/child relationships
	for i := range book.Accounts {
		if !book.Accounts[i].ParentGuid.Valid {
			continue
		}

		book.Accounts[i].Parent = book.GetAccountByGUID(
			book.Accounts[i].ParentGuid.String)
		book.Accounts[i].Parent.Children =
			append(book.Accounts[i].Parent.Children, book.Accounts[i])
	}

	// Load lots
	{
		ls := []DbLot{}
		err = db.Select(&ls, "SELECT * FROM lots")
		if err != nil {
			log.Fatal(err)
		}

		for _, dbl := range ls {
			l := &Lot{
				book:  book,
				DbLot: dbl,
			}

			book.lots = append(book.lots, l)
		}
	}

	// Load slots
	ss := []DbSlot{}
	err = db.Select(&ss, "SELECT * FROM slots")
	if err != nil {
		log.Fatal(err)
	}

	for _, dbs := range ss {
		s := &Slot{
			book:   book,
			DbSlot: dbs,
		}

		book.slots = append(book.slots, s)
	}

	// Load commodities
	cs := []DbCommodity{}
	err = db.Select(&cs, "SELECT * FROM commodities")
	if err != nil {
		log.Fatal(err)
	}

	for _, dbc := range cs {
		c := &Commodity{
			dbc,
		}
		book.commodities = append(book.commodities, c)
	}

	// Load vendors
	vs := []DbVendor{}
	err = db.Select(&vs, "SELECT * FROM vendors")
	if err != nil {
		log.Fatal(err)
	}

	for _, dbv := range vs {
		v := &Vendor{
			dbv,
		}
		book.vendors = append(book.vendors, v)
	}

	// Load employees
	{
		es := []DbEmployee{}
		err = db.Select(&es, "SELECT * FROM employees")
		if err != nil {
			log.Fatal(err)
		}

		for _, dbe := range es {
			e := &Employee{
				book:       book,
				DbEmployee: dbe,
			}
			book.employees = append(book.employees, e)
		}
	}

	// Load invoices
	is := []DbInvoice{}
	err = db.Select(&is, "SELECT * FROM invoices")
	if err != nil {
		log.Fatal(err)
	}

	for _, dbi := range is {
		cnslot := book.getSlotForObjByName(dbi.Guid, "credit-note")
		if cnslot == nil {
			log.Fatalf("Credit note slot for invoice %s not found\n", dbi.Guid)
		}
		if cnslot.SlotType != int(SlotTypeInt64) {
			log.Fatalf("Credit note slot for invoice %s has wrong type\n",
				dbi.Guid)
		}
		if !cnslot.Int64Val.Valid {
			log.Fatalf("Credit note slot for invoice %s has null value\n",
				dbi.Guid)
		}

		i := &Invoice{
			book:         book,
			IsCreditNote: (cnslot.Int64Val.Int64 != 0),
			DbInvoice:    dbi,
		}

		book.invoices = append(book.invoices, i)
	}

	// Load entries
	es := []DbEntry{}
	err = db.Select(&es, "SELECT * FROM entries")
	if err != nil {
		log.Fatal(err)
	}

	for _, dbe := range es {
		e := &Entry{
			book:    book,
			DbEntry: dbe,
		}

		// Find associated invoice
		if dbe.Invoice.Valid {
			invoice := book.GetInvoiceByGUID(dbe.Invoice.String)
			if invoice == nil {
				log.Fatalf("Invoice %s not found for entry %s\n",
					dbe.Invoice.String, dbe.Guid)
			}

			// Connect invoice and entry to each other
			invoice.Entries = append(invoice.Entries, e)
		}

		book.entries = append(book.entries, e)
	}

	// Load transactions
	{
		ts := []DbTransaction{}
		err = db.Select(&ts, "SELECT * FROM transactions")
		if err != nil {
			log.Fatal(err)
		}

		for _, dbt := range ts {
			t := &Transaction{
				book:          book,
				DbTransaction: dbt,
			}

			book.transactions = append(book.transactions, t)
		}
	}

	// Load splits
	{
		ss := []DbSplit{}
		err = db.Select(&ss, "SELECT * FROM splits")
		if err != nil {
			log.Fatal(err)
		}

		for _, dbs := range ss {
			s := &Split{
				book:    book,
				DbSplit: dbs,
			}

			// Find associated transaction and connect
			txn := book.GetTransactionByGUID(s.TxGuid)
			if txn == nil {
				log.Fatalf("Associated transaction %s for split %s not found\n",
					s.TxGuid, s.Guid)
			}

			s.Transaction = txn
			txn.Splits = append(txn.Splits, s)

			// Find associated account and connect
			acc := book.GetAccountByGUID(s.AccountGuid)
			if acc == nil {
				log.Fatalf("Associated account %s for split %s not found\n",
					s.AccountGuid, s.Guid)
			}

			acc.Splits = append(acc.Splits, s)
			s.Account = acc

			book.splits = append(book.splits, s)
		}
	}

	return book, nil
}

func (b *Book) GetAccountByGUID(guid string) *Account {
	for _, acc := range b.Accounts {
		if acc.Guid == guid {
			return acc
		}
	}

	return nil
}

func (b *Book) GetTransactionByGUID(guid string) *Transaction {
	for _, t := range b.transactions {
		if t.Guid == guid {
			return t
		}
	}

	return nil
}

func (b *Book) GetLotByGUID(guid string) *Lot {
	for _, l := range b.lots {
		if l.Guid == guid {
			return l
		}
	}

	return nil
}

func (b *Book) GetAccountByPath(path string) *Account {
	return b.RootAccount.GetAccountByPath(path)
}

func (b *Book) incrementCounter(name string) int64 {
	s := b.getSlotByName(name)
	if s != nil && s.Int64Val.Valid {
		ret := s.Int64Val.Int64
		s.Int64Val.Int64 = s.Int64Val.Int64 + 1
		s.Write()
		return ret
	}

	return -1
}

func (b *Book) incrementBillCounter() int64 {
	return b.incrementCounter("counters/gncBill")
}

func (b *Book) incrementInvoiceCounter() int64 {
	return b.incrementCounter("counters/gncInvoice")
}

func (b *Book) incrementExpenseVoucherCounter() int64 {
	return b.incrementCounter("counters/gncExpVoucher")
}

func (b *Book) AddSlotIfNotExist(s *Slot) *Slot {
	existing := b.getSlotByName(s.Name)
	if existing != nil {
		return existing
	}

	b.AddSlot(s)
	return s
}

func (b *Book) getCounterFormat(name string) string {
	// Ensure 'counter_formats' slot exists
	cfs := &Slot{
		DbSlot: DbSlot{
			ObjGuid:  b.Guid,
			Name:     "counter_foramts",
			SlotType: int(SlotTypeFrame),
			GuidVal:  sql.NullString{NewGuid(), true},
		},
	}
	cfs = b.AddSlotIfNotExist(cfs)

	s := &Slot{
		DbSlot: DbSlot{
			ObjGuid:   cfs.ObjGuid,
			Name:      name,
			SlotType:  int(SlotTypeString),
			StringVal: sql.NullString{"%05li", true},
		},
	}
	s = b.AddSlotIfNotExist(s)
	if s != nil && s.StringVal.Valid {
		// Replace 'li' with 'd' (as Go's printf doesn't seem to support li)
		r := regexp.MustCompile(`(%-?\d*)li`)
		return r.ReplaceAllString(s.StringVal.String, "${1}d")
	}

	return ""
}

func (b *Book) getCounter(name string) int64 {
	// Ensure 'counters' slot exists
	cfs := &Slot{
		DbSlot: DbSlot{
			ObjGuid:  b.Guid,
			Name:     "counters",
			SlotType: int(SlotTypeFrame),
			GuidVal:  sql.NullString{NewGuid(), true},
		},
	}
	cfs = b.AddSlotIfNotExist(cfs)

	s := &Slot{
		DbSlot: DbSlot{
			ObjGuid:  cfs.ObjGuid,
			Name:     name,
			SlotType: int(SlotTypeInt64),
			Int64Val: sql.NullInt64{0, true},
		},
	}
	s = b.AddSlotIfNotExist(s)

	if s != nil && s.Int64Val.Valid {
		return s.Int64Val.Int64
	}

	return -1
}

func (b *Book) GetBillCounterFormat() string {
	return b.getCounterFormat("counter_formats/gncBill")
}

func (b *Book) GetInvoiceCounterFormat() string {
	return b.getCounterFormat("counter_formats/gncInvoice")
}

func (b *Book) GetExpenseVoucherCounterFormat() string {
	return b.getCounterFormat("counter_formats/gncExpVoucher")
}

func (b *Book) GetBillCounter() int64 {
	return b.getCounter("counters/gncBill")
}

func (b *Book) GetInvoiceCounter() int64 {
	return b.getCounter("counters/gncInvoice")
}

func (b *Book) GetExpenseVoucherCounter() int64 {
	return b.getCounter("counters/gncExpVoucher")
}

func (b *Book) getSlotByName(name string) *Slot {
	for _, s := range b.slots {
		if s.Name == name {
			return s
		}
	}

	return nil
}

func (b *Book) getSlotsByObjGUID(guid string) []*Slot {
	slots := []*Slot{}
	for _, s := range b.slots {
		if s.ObjGuid == guid {
			slots = append(slots, s)
		}
	}

	return slots
}

func (b *Book) getSlotForObjByName(objGuid string, slotName string) *Slot {
	slots := b.getSlotsByObjGUID(objGuid)
	for _, s := range slots {
		if s.Name == slotName {
			return s
		}
	}

	return nil
}

func (b *Book) AddInvoice(i *Invoice) {
	// Generate Guid if not set
	if i.Guid == "" {
		i.Guid = NewGuid()
	}

	// Generate Id if not set
	if i.Id == "" {
		switch i.GetOwnerType() {
		case OwnerTypeVendor:
			b.GetBillCounter()
		}
	}

	i.book = b
	i.Write()

	// Create credit-note slot indicating whether this invoice is a credit note
	slotVal := 0
	if i.IsCreditNote {
		slotVal = 1
	}

	s := &Slot{
		DbSlot: DbSlot{
			ObjGuid:  i.Guid,
			Name:     "credit-note",
			SlotType: int(SlotTypeInt64),
			Int64Val: sql.NullInt64{int64(slotVal), true},
		},
	}
	b.AddSlot(s)

	// Increment correct counter
	switch i.GetOwnerType() {
	case OwnerTypeVendor:
		b.incrementBillCounter()
	case OwnerTypeCustomer:
		b.incrementInvoiceCounter()
	case OwnerTypeEmployee:
		b.incrementExpenseVoucherCounter()
	default:
		log.Fatal("Unknown invoice type not implemented")
	}

	b.invoices = append(b.invoices, i)
}

func (b *Book) GetInvoiceByGUID(guid string) *Invoice {
	for _, i := range b.invoices {
		if i.Guid == guid {
			return i
		}
	}

	return nil
}

func (b *Book) AddSlot(s *Slot) {
	s.book = b
	s.create()
	b.slots = append(b.slots, s)
}

func (b *Book) RemoveSlot(s *Slot) {
	// Remove from book slots slice
	for i, x := range b.slots {
		if x == s {
			b.slots[i] = b.slots[len(b.slots)-1]
			break
		}
	}
	b.slots = b.slots[:len(b.slots)-1]

	// Remove from database
	s.remove()
}

func (b *Book) AddTransaction(t *Transaction) {
	if t.Guid == "" {
		t.Guid = NewGuid()
	}

	t.book = b
	t.write()
	b.transactions = append(b.transactions, t)
}

func (b *Book) GetCommodityByGUID(guid string) *Commodity {
	for _, c := range b.commodities {
		if c.Guid == guid {
			return c
		}
	}

	return nil
}

func (b *Book) GetCommodityByMnemonic(mnemonic string) *Commodity {
	for _, c := range b.commodities {
		if c.Mnemonic == mnemonic {
			return c
		}
	}

	return nil
}

func (b *Book) GetEmployeeByUsername(username string) *Employee {
	for _, e := range b.employees {
		if e.Username == username {
			return e
		}
	}

	return nil
}

func (b *Book) GetEmployeeByName(name string) *Employee {
	for _, e := range b.employees {
		if e.AddrName.Valid && e.AddrName.String == name {
			return e
		}
	}

	return nil
}

func (b *Book) GetEmployeeByGUID(guid string) *Employee {
	for _, e := range b.employees {
		if e.Guid == guid {
			return e
		}
	}

	return nil
}

func (b *Book) GetVendorByName(name string) *Vendor {
	for _, v := range b.vendors {
		if v.Name == name {
			return v
		}
	}

	return nil
}

func (b *Book) GetVendorByGUID(guid string) *Vendor {
	for _, v := range b.vendors {
		if v.Guid == guid {
			return v
		}
	}

	return nil
}

// Returns the currency (commodity) of the root account
func (b *Book) GetDefaultCurrency() *Commodity {
	if !b.RootAccount.CommodityGuid.Valid {
		log.Fatal("Root account commodity_guid is null")
	}

	return b.GetCommodityByGUID(b.RootAccount.CommodityGuid.String)
}

func (b *Book) Close() {
	b.DB.Close()
}
