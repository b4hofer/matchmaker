package gnucash

type OwnerType int32

const (
	OwnerTypeNone OwnerType = iota
	OwnerTypeUndefined
	OwnerTypeCustomer
	OwnerTypeJob
	OwnerTypeVendor
	OwnerTypeEmployee
)
