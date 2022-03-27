package gnucash

type EntryPaymentType int

const (
	EntryPaymentTypeCash EntryPaymentType = iota + 1
	EntryPaymentTypeCard
)
