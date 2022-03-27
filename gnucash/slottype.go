package gnucash

type SlotType int

const (
	SlotTypeInvalid SlotType = iota - 1
	_
	SlotTypeInt64
	SlotTypeDouble
	SlotTypeNumeric
	SlotTypeString
	SlotTypeGuid
	SlotTypeTime64
	SlotTypePlaceholder
	SlotTypeGlist
	SlotTypeFrame
	SlotTypeGdate
)
