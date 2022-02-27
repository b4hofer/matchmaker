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
	skip           int
	matchSkip      int
	defaultAccount string

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

				if line < skip {
					line++
					continue
				}

				// Find match
				var outRecord []string
				outRecord = append(outRecord, record...)

				match := false
				for ml, m := range matches {
					if len(m) != len(record)+1 {
						log.Fatal("Matches file must have exactly one more column than source file")
					}

					for i := 0; i < len(record); i++ {
						if m[i] == "" {
							continue
						}

						match, err = regexp.MatchString(m[i], record[i])
						if err != nil {
							log.Fatalf("Error in regular expression '%s' on line %d", m[i], ml+matchSkip)
						}

						if match {
							outRecord = append(outRecord, m[len(record)])
							break
						}
					}

					if match {
						break
					}

				}

				if !match {
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
	rootCmd.Flags().StringVarP(&defaultAccount, "default-account", "d", "Imbalance-EUR", "account to assign when no"+
		" match has been found, default is 'Imbalance-EUR'")
}

func initConfig() {

}
