package gnucash

import (
	"database/sql"
	"log"
)

type Slot struct {
	book *Book
	DbSlot
}

type DbSlot struct {
	Id              int
	ObjGuid         string `db:"obj_guid"`
	Name            string
	SlotType        int             `db:"slot_type"`
	Int64Val        sql.NullInt64   `db:"int64_val"`
	StringVal       sql.NullString  `db:"string_val"`
	DoubleVal       sql.NullFloat64 `db:"double_val"`
	TimespecVal     sql.NullString  `db:"timespec_val"`
	GuidVal         sql.NullString  `db:"guid_val"`
	NumericValNum   sql.NullInt64   `db:"numeric_val_num"`
	NumericValDenom sql.NullInt64   `db:"numeric_val_denom"`
	GdateVal        sql.NullString  `db:"gdate_val"`
}

// Same as write, but will not replace existing entry and lets sqlite assign id
// automatically
func (s *Slot) create() {
	query := `INSERT INTO slots(obj_guid, name, slot_type,
			int64_val, string_val, double_val, timespec_val, guid_val,
			numeric_val_num, numeric_val_denom, gdate_val)
		VALUES(:obj_guid, :name, :slot_type, :int64_val, :string_val,
			:double_val, :timespec_val, :guid_val, :numeric_val_num,
			:numeric_val_denom, :gdate_val)`
	_, err := s.book.DB.NamedExec(query, s.DbSlot)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Slot) Write() {
	query := `INSERT OR REPLACE INTO slots(id, obj_guid, name, slot_type,
			int64_val, string_val, double_val, timespec_val, guid_val,
			numeric_val_num, numeric_val_denom, gdate_val)
		VALUES(:id, :obj_guid, :name, :slot_type, :int64_val, :string_val,
			:double_val, :timespec_val, :guid_val, :numeric_val_num,
			:numeric_val_denom, :gdate_val)`
	_, err := s.book.DB.NamedExec(query, s.DbSlot)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Slot) remove() {
	query := `DELETE FROM slots WHERE id=:id`
	_, err := s.book.DB.NamedExec(query, s.DbSlot)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Slot) GetChildren() []*Slot {
	if !s.GuidVal.Valid {
		log.Fatal("Cannot GetChildren() when GuidVal of slot is null")
	}

	return s.book.getSlotsByObjGUID(s.GuidVal.String)
}

func (s *Slot) AddChild(c *Slot) {
	if !s.GuidVal.Valid {
		log.Fatal("Cannot add child to slot if parent GuidVal is null")
	}

	c.ObjGuid = s.GuidVal.String
	s.book.AddSlot(c)
}
