package cmd

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bvorhofer.com/matchmaker/gnucash"
	"github.com/spf13/cobra"
)

var (
	monthlyCmd = &cobra.Command{
		Use:   "monthly FILE",
		Short: "Generate bills and vouchers from configs in current directory",
		Long:  `Generate bills and vouchers from configs in current directory`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Open GnuCash file
			book, err := gnucash.OpenBookFromSQLite(args[0])
			if err != nil {
				log.Fatal(err)
			}
			defer book.Close()

			// Process date args (if given)
			var startDateTime time.Time
			if startDate != "" {
				startDateTime, err = time.Parse("2006-01-02", startDate)
				if err != nil {
					log.Fatal("Invalid start date given, use format YYYY-MM-DD")
				}

				log.Println("Start date is", startDateTime.String())
			}

			var endDateTime time.Time
			if endDate != "" {
				endDateTime, err = time.Parse("2006-01-02", endDate)
				if err != nil {
					log.Fatal("Invalid end date given, use format YYYY-MM-DD")
				}

				// Add one day to end date so it's exclusive
				oneDay, _ := time.ParseDuration("24h")
				endDateTime = endDateTime.Add(oneDay)

				log.Println("End date (exclusive) is", endDateTime.String())
			}

			items, err := ioutil.ReadDir(".")
			if err != nil {
				log.Fatal("Error reading current directory: " + err.Error())
			}

			// Find payable account
			ap := book.GetAccountByPath(payableAccount)
			if ap == nil {
				log.Fatalf("Could not find account '%s'\n", payableAccount)
			}

			// Employee expense voucher entries, key is employee GUID
			voucherItems := make(map[string][]*gnucash.Entry)

			for _, item := range items {
				if !strings.HasSuffix(item.Name(), ".csv") {
					continue
				}

				// Open CSV config file
				f, err := os.Open(item.Name())
				if err != nil {
					log.Fatal("Error opening match file: " + err.Error())
				}

				r := csv.NewReader(f)
				matches, err := r.ReadAll()
				if err != nil {
					log.Fatal("Error reading match file: " + err.Error())
				}

				// Find vendor based on filename
				vendorName := strings.TrimSuffix(filepath.Base(f.Name()), filepath.Ext(f.Name()))
				vendor := book.GetVendorByName(vendorName)
				if vendor == nil {
					log.Fatalf("Unable to find vendor '%s'", vendorName)
				}

				entries := []*gnucash.Entry{}
				matchSplits := []*gnucash.Split{}

				// Search splits in payable account for matches
				// CSV format: SPLIT REGEX, TXN MEMO REGEX, ENTRY DESCR, DEST ACCT
				for _, s := range ap.Splits {
					if !s.Transaction.PostDate.Valid {
						log.Printf("WARNING: Transaction %s has no post date, skipping...\n", s.Transaction.Guid)
						continue
					}

					tDate, err := time.Parse("2006-01-02 15:04:05", s.Transaction.PostDate.String)
					if err != nil {
						log.Printf("WARNING: Transaction %s has invalid post date '%s', skipping...\n",
							s.Transaction.Guid, s.Transaction.PostDate.String)
						continue
					}

					// Ignore splits before start date (if specified)
					if startDate != "" && tDate.Before(startDateTime) {
						log.Println("> too early, ignoring...")
						continue
					}

					// Ignore splits after end date (if specified)
					if endDate != "" && tDate.After(endDateTime) {
						log.Println("> too late, ignoring...")
						continue
					}

					// Ignore splits which have been assigned as payment already
					if s.LotGuid.Valid && s.LotGuid.String != "" {
						continue
					}

					// Ignore splits which come from bills or were assigned payments
					if s.Action == "Payment" || s.Action == "Bill" {
						continue
					}

					for i, m := range matches {
						memoMatch := false
						if m[0] != "" {
							memoMatch, err = regexp.MatchString(m[0], s.Memo)
							if err != nil {
								log.Fatalf("Error in regex %s on line %d in bill generator config",
									m[0], i)
							}
						}

						descMatch := false
						if m[1] != "" {
							descMatch, err = regexp.MatchString(m[1],
								s.Transaction.Description.String)
							if err != nil {
								log.Fatalf("Error in regex %s on line %d in bill generator config",
									m[1], i)
							}
						}

						if memoMatch || descMatch {
							// Find destination account
							bAcc := book.GetAccountByPath(m[3])
							if bAcc == nil {
								log.Fatalf("Unable to find account '%s'", m[3])
							}

							// Create entry for this split
							date := gnucash.GetCurrentTimeString()
							if s.Transaction.PostDate.Valid {
								date = s.Transaction.PostDate.String
							}
							entry := &gnucash.Entry{
								DbEntry: gnucash.DbEntry{
									Date:          date,
									DateEntered:   sql.NullString{gnucash.GetCurrentTimeString(), true},
									Description:   sql.NullString{m[2], true},
									QuantityNum:   sql.NullInt64{1, true},
									QuantityDenom: sql.NullInt64{1, true},
									BAcct:         sql.NullString{bAcc.Guid, true},
									BPriceNum:     sql.NullInt64{s.ValueNum, true},
									BPriceDenom:   sql.NullInt64{s.ValueDenom, true},
								},
							}
							entries = append(entries, entry)
							matchSplits = append(matchSplits, s)

							// Find employees for reimbursements
							for j := 4; j < len(m)-1; j += 2 {
								e := book.GetEmployeeByUsername(m[j])
								if e == nil {
									log.Fatalf("Unable to find employee '%s'", m[j])
								}

								ss := strings.Split(m[j+1], "/")
								if len(ss) != 2 {
									log.Fatalf("Invalid rational '%s'", m[j+1])
								}
								num, err := strconv.ParseInt(ss[0], 10, 64)
								if err != nil {
									log.Fatalf("Invalid numerator in '%s'", m[j+1])
								}

								denom, err := strconv.ParseInt(ss[1], 10, 64)
								if err != nil {
									log.Fatalf("Invalid denominator in '%s'", m[j+1])
								}

								fmt.Printf("Entry %d/%d for %s\n", num, denom, e.AddrName.String)

								// Re-use (copy) bill entry for voucher, but update quantity to specified share
								// NOTE: quantity sign is reversed for credit notes (not sure why)
								ent := *entry
								ent.QuantityNum = sql.NullInt64{-num, true}
								ent.QuantityDenom = sql.NullInt64{denom, true}

								voucherItems[e.Guid] = append(voucherItems[e.Guid], &ent)
							}

							break
						}
					}
				}

				// If any entries were created, create a new invoice and add the
				// entries
				if len(entries) > 0 {
					invoice := &gnucash.Invoice{
						IsCreditNote: false,
						DbInvoice: gnucash.DbInvoice{
							Id:             fmt.Sprintf("%05d", book.GetBillCounter()),
							DateOpened:     sql.NullString{gnucash.GetCurrentTimeString(), true},
							Notes:          fmt.Sprintf("Generated by gncx (%s)", f.Name()),
							Active:         1,
							Currency:       book.GetDefaultCurrency().Guid,
							OwnerType:      sql.NullInt32{int32(gnucash.OwnerTypeVendor), true},
							OwnerGuid:      sql.NullString{vendor.Guid, true},
							ChargeAmtNum:   sql.NullInt64{0, true},
							ChargeAmtDenom: sql.NullInt64{1, true},
						},
					}
					book.AddInvoice(invoice)

					for _, e := range entries {
						invoice.AddEntry(e)
					}

					// Post the invoice to A/P account
					invoice.Post(ap)

					for _, s := range matchSplits {
						invoice.AssignPayment(s)
					}
				}
			}

			// Generate expense vouchers
			for empGuid, entries := range voucherItems {
				voucher := &gnucash.Invoice{
					IsCreditNote: true,
					DbInvoice: gnucash.DbInvoice{
						Id:             fmt.Sprintf("%05d", book.GetExpenseVoucherCounter()),
						DateOpened:     sql.NullString{gnucash.GetCurrentTimeString(), true},
						Notes:          "Generated by gncx",
						Active:         1,
						Currency:       book.GetDefaultCurrency().Guid,
						OwnerType:      sql.NullInt32{int32(gnucash.OwnerTypeEmployee), true},
						OwnerGuid:      sql.NullString{empGuid, true},
						ChargeAmtNum:   sql.NullInt64{0, true},
						ChargeAmtDenom: sql.NullInt64{1, true},
					},
				}
				book.AddInvoice(voucher)

				for _, entry := range entries {
					voucher.AddEntry(entry)
				}

				voucher.Post(ap)
			}
		},
	}
)
