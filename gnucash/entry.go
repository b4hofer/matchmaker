package gnucash

import (
	"database/sql"
	"log"
	"math/big"
)

type Entry struct {
	book *Book
	DbEntry
}

type DbEntry struct {
	Guid           string
	Date           string
	DateEntered    sql.NullString `db:"date_entered"`
	Description    sql.NullString
	Action         sql.NullString
	Notes          sql.NullString
	QuantityNum    sql.NullInt64  `db:"quantity_num"`
	QuantityDenom  sql.NullInt64  `db:"quantity_denom"`
	IAcct          sql.NullString `db:"i_acct"`
	IPriceNum      sql.NullInt64  `db:"i_price_num"`
	IPriceDenom    sql.NullInt64  `db:"i_price_denom"`
	IDiscountNum   sql.NullInt64  `db:"i_discount_num"`
	IDiscountDenom sql.NullInt64  `db:"i_discount_denom"`
	Invoice        sql.NullString `db:"invoice"`
	IDiscType      sql.NullString `db:"i_disc_type"`
	IDiscHow       sql.NullString `db:"i_disc_how"`
	ITaxable       sql.NullInt32  `db:"i_taxable"`
	ITaxincluded   sql.NullInt32  `db:"i_taxincluded"`
	ITaxtable      sql.NullString `db:"i_taxtable"`
	BAcct          sql.NullString `db:"b_acct"`
	BPriceNum      sql.NullInt64  `db:"b_price_num"`
	BPriceDenom    sql.NullInt64  `db:"b_price_denom"`
	Bill           sql.NullString
	BTaxable       sql.NullInt32  `db:"b_taxable"`
	BTaxincluded   sql.NullInt32  `db:"b_taxincluded"`
	BTaxtable      sql.NullString `db:"b_taxtable"`
	BPaytype       sql.NullInt32  `db:"b_paytype"`
	Billable       sql.NullInt32
	BilltoType     sql.NullInt32  `db:"billto_type"`
	BilltoGuid     sql.NullString `db:"billto_guid"`
	OrderGuid      sql.NullString `db:"order_guid"`
}

func (e *Entry) GetInvoice() *Invoice {
	if !e.Invoice.Valid {
		return nil
	}

	return e.book.GetInvoiceByGUID(e.Invoice.String)
}

func (e *Entry) GetBill() *Invoice {
	if !e.Bill.Valid {
		return nil
	}

	return e.book.GetInvoiceByGUID(e.Bill.String)
}

func (e *Entry) Write() {
	query := `INSERT INTO "entries" ("guid", "date", "date_entered",
			"description", "action", "notes", "quantity_num", "quantity_denom",
			"i_acct", "i_price_num", "i_price_denom", "i_discount_num",
			"i_discount_denom", "invoice", "i_disc_type", "i_disc_how",
			"i_taxable", "i_taxincluded", "i_taxtable", "b_acct", "b_price_num",
			"b_price_denom", "bill", "b_taxable", "b_taxincluded", "b_taxtable",
			"b_paytype", "billable", "billto_type", "billto_guid", "order_guid")
		VALUES (:guid, :date, :date_entered, :description, :action, :notes,
			:quantity_num, :quantity_denom, :i_acct, :i_price_num,
			:i_price_denom, :i_discount_num, :i_discount_denom, :invoice,
			:i_disc_type, :i_disc_how, :i_taxable, :i_taxincluded, :i_taxtable,
			:b_acct, :b_price_num, :b_price_denom, :bill, :b_taxable,
			:b_taxincluded, :b_taxtable, :b_paytype, :billable, :billto_type,
			:billto_guid, :order_guid)`
	_, err := e.book.DB.NamedExec(query, e.DbEntry)
	if err != nil {
		log.Fatal(err)
	}
}

// WARNING: Does not support discounts!
func (e *Entry) GetSubtotal() *big.Rat {
	quantity := big.NewRat(0, 1)
	if e.QuantityNum.Valid && e.QuantityDenom.Valid {
		quantity = big.NewRat(e.QuantityNum.Int64, e.QuantityDenom.Int64)
	}

	price := big.NewRat(0, 1)
	if e.BPriceNum.Valid && e.BPriceDenom.Valid {
		price = big.NewRat(e.BPriceNum.Int64, e.BPriceDenom.Int64)
	} else {
		log.Fatal("GetSubtotal only supported for vendor bills")
	}

	return quantity.Mul(quantity, price)
}
