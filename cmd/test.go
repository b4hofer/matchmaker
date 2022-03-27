package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"bvorhofer.com/matchmaker/gnucash"
	"github.com/spf13/cobra"
)

var (
	testCmd = &cobra.Command{
		Use:   "test FILE",
		Short: "Test command",
		Long:  `Test command for testing`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			book, err := gnucash.OpenBookFromSQLite(args[0])
			if err != nil {
				log.Fatal(err)
			}
			defer book.Close()

			curr := book.GetDefaultCurrency()

			swm := book.GetVendorByName("SWM")
			if swm == nil {
				log.Fatal("SWM not found")
			}

			chris := book.GetEmployeeByUsername("chris")
			//if chris == nil {
			//	log.Fatal("Chris not found")
			//}

			ap := book.GetAccountByPath("Liabilities:Accounts Payable")
			if ap == nil {
				log.Fatal("A/P account not found")
			}

			for _, e := range ap.Splits {
				fmt.Println(e.Transaction.Description, ":", gnucash.GncRationalToString(e.ValueNum, e.ValueDenom))
			}

			return

			invoice := &gnucash.Invoice{
				IsCreditNote: true,
				DbInvoice: gnucash.DbInvoice{
					Id:             strconv.FormatInt(book.GetBillCounter()+1, 10),
					Active:         1,
					DateOpened:     sql.NullString{gnucash.GetCurrentTimeString(), true},
					Currency:       curr.Guid,
					OwnerType:      sql.NullInt32{int32(gnucash.OwnerTypeEmployee), true},
					OwnerGuid:      sql.NullString{chris.Guid, true},
					ChargeAmtNum:   sql.NullInt64{0, true},
					ChargeAmtDenom: sql.NullInt64{1, true},
				},
			}

			book.AddInvoice(invoice)

			entry := &gnucash.Entry{
				DbEntry: gnucash.DbEntry{
					Date:          gnucash.GetCurrentTimeString(),
					Description:   sql.NullString{"Test Entry", true},
					QuantityNum:   sql.NullInt64{1, true},
					QuantityDenom: sql.NullInt64{1, true},
					BPriceNum:     sql.NullInt64{1240, true},
					BPriceDenom:   sql.NullInt64{100, true},
					ITaxable:      sql.NullInt32{1, true},
					BPaytype:      sql.NullInt32{int32(gnucash.EntryPaymentTypeCash), true},
				},
			}

			invoice.AddEntry(entry)
		},
	}
)

func printAccounts(acc *gnucash.Account) {
	_printAccounts(acc, "")
}

func _printAccounts(acc *gnucash.Account, prefix string) {
	fmt.Println(prefix + acc.Name.String)
	for _, a := range acc.Children {
		_printAccounts(a, prefix+"  ")
	}
}
