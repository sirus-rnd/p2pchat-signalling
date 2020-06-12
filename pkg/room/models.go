package room

// Models defined in room package
var Models = []interface{}{
	&RoomModel{},
	&UserModel{},
}

// RoomModel define room / channel information save on database
type RoomModel struct {
	ID          string       `gorm:"primary_key;not null;size:100"`
	Name        string       `gorm:"column:name;"`
	Description string       `gorm:"column:description;"`
	Photo       string       `gorm:"column:photo;"`
	Members     []*UserModel `gorm:"many2many:room_members;save_associations:false;"`
}

// UserModel define user information save on database
type UserModel struct {
	ID     string       `gorm:"primary_key;not null;size:100"`
	Name   string       `gorm:"column:name;"`
	Photo  string       `gorm:"column:photo;"`
	Online bool         `gorm:"column:online;default:false"`
	Rooms  []*RoomModel `gorm:"many2many:room_members;save_associations:false;"`
}
