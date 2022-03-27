package cmd

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"regexp"

	"github.com/spf13/cobra"
)

var (
	skip               int
	matchSkip          int
	defaultAccount     string
	delimiter          string
	generateConfigFile string
	payableAccount     string
	startDate          string
	endDate            string

	rootCmd = &cobra.Command{
		Use:   "matchmaker FILE MATCHFILE",
		Short: "CSV preprocessor for auto-matching GnuCash imports",
		Long: `Matchmaker adds an additional 'Matched Account' column to a CSV exported from a bank
so they can be automatically assigned when imported into GnuCash.`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if matchSkip < 0 {
				matchSkip = skip
			}

			if len(delimiter) != 1 {
				log.Fatal("Delimiter must be of length 1")
			}

			// Load match file
			f, err := os.Open(args[1])
			if err != nil {
				log.Fatal("Error opening match file: " + err.Error())
			}

			r := csv.NewReader(f)
			matches, err := r.ReadAll()
			if err != nil {
				log.Fatal("Error reading match file: " + err.Error())
			}

			matches = matches[matchSkip:]

			// Load source file
			f, err = os.Open(args[0])
			if err != nil {
				log.Fatal("Error opening file: " + err.Error())
			}

			r = csv.NewReader(f)
			r.Comma = []rune(delimiter)[0]
			var out [][]string

			line := 0
			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Fatal(err)
				}

				var outRecord []string
				outRecord = append(outRecord, record...)

				if line < skip {
					outRecord = append(outRecord, "Matched Account")
					out = append(out, outRecord)
					line++
					continue
				}

				// Find match
				matchFound := false
				for ml, m := range matches {
					if len(m) != len(record)+1 {
						log.Fatalf("Matches file must have exactly one more column than source file (matches file has"+
							" %d, source file has %d)\n", len(m), len(record))
					}

					blank := true
					for i := 0; i < len(record); i++ {
						if m[i] == "" {
							continue
						}

						match, err := regexp.MatchString(m[i], record[i])
						if err != nil {
							log.Fatalf("Error in regular expression '%s' on line %d", m[i], ml+matchSkip)
						}

						if match {
							// matchFound is only set to true if all fields have been blank so far!
							// Otherwise, a mismatch could be overridden by a later match.
							matchFound = matchFound || blank
						} else {
							matchFound = false
						}

						blank = false
					}

					if matchFound {
						outRecord = append(outRecord, m[len(record)])
						break
					}

				}

				if !matchFound {
					outRecord = append(outRecord, defaultAccount)
				}

				out = append(out, outRecord)

				line++
			}

			// Write output to stdout
			w := csv.NewWriter(os.Stdout)
			w.WriteAll(out)

			if err := w.Error(); err != nil {
				log.Fatal("Error writing csv: ", err)
			}
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().IntVarP(&skip, "skip", "s", 1, "number of lines to skip (default 1)")
	rootCmd.Flags().IntVarP(&matchSkip, "matchskip", "m", -1, "number of lines to skip in match file (defaults to skip"+
		" parameter)")
	rootCmd.Flags().StringVarP(&defaultAccount, "default-account", "a", "Imbalance-EUR", "account to assign when no"+
		" match has been found, default is 'Imbalance-EUR'")
	rootCmd.Flags().StringVarP(&delimiter, "delimiter", "d", ",", "CSV delimiter for source file, default ','")

	monthlyCmd.Flags().StringVarP(&generateConfigFile, "generate", "g", "", "generate bills using specified CSV config file")
	monthlyCmd.Flags().StringVarP(&payableAccount, "payable-account", "a", "Liabilities:Accounts Payable",
		"account used to search for matches when generating bills")
	monthlyCmd.Flags().StringVarP(&startDate, "start-date", "s", "", "start date of transactions to include (YYYY-MM-DD, optional)")
	monthlyCmd.Flags().StringVarP(&endDate, "end-date", "e", "", "end date of transactions to include (YYYY-MM-DD, optional)")

	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(monthlyCmd)
}

func initConfig() {

}
