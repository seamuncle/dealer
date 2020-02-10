package dealer

// LotType represents the type of lot a bit of inventory is addressing
type LotType string

const (
	// TypeNew represents an inventorySet or Vehicle attached to a "lot of" new vehicles
	TypeNew LotType = "NEW"
	// TypeUsed represents an inventorySet or Vehicle attached to a "lot of" used vehicles
	TypeUsed = "USED"
)

// Lot represents an abstraction for where a vehicle belongs.
// It will belong to a dealer--with an ID and Name, and a lot type
type Lot struct {
	DealerID   int     `gorm:"column:d_id"`
	DealerName string  `gorm:"column:d_name"` // Third normal brain is screaming at me
	LotType    LotType `gorm:"column:stock_type"`
}
