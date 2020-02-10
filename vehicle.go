package dealer

import (
	"time"
)

// VehicleState marks something about a vehicle's known relation to the database
type VehicleState int

const (
	// StateUnknown indeicates probably not in the database
	StateUnknown VehicleState = iota
	// StatePersisted indicates this vehicle has been loaded directly from its db counterpart
	StatePersisted
	// StateAltered indicates there is a difference between a feed vehicle and its db counterpart
	StateAltered
	// StateUnaltered indicates there is no difference between a feed vehicle and its db counterpart
	StateUnaltered
)

// VehicleKey is a reaonable way to uniquely identify a vehicle--given the high likelyhood
// of a missing VIN or missing or non-unique stock number
// In the event both are missing or vin is missing and the stock number is duplicate,
// its fair to point at garbage-in, garbage out--but we can put a little effort in
type VehicleKey struct {
	VIN   string `gorm:"column:vin"`
	Stock string `gorm:"column:stock_id"` // varchar(4) is gonna be a world of hurt IRL
}

// Vehicle represents a composite representation of a vehicle on an import feed
// and in the database
type Vehicle struct {
	ID           int       `gorm:"column:v_id;primary_key"`
	Created      time.Time `gorm:"column:created_time"`
	TheGuilty    string    `gorm:"column:last_modified_by"`
	LastModified time.Time `gorm:"column:last_modified_time"`
	Lot          `gorm:"embedded"`
	FeedVehicle  `gorm:"embedded"`
	State        VehicleState `gorm:"-"`
}

// TableName oerrides the default table name "vehicle" for the gorm library
func (Vehicle) TableName() string {
	return "inventory"
  }

// FeedVehicle represents the bits of the vehcile record a feed is explicitly responsible for
// and may be modified from the feed without repcercussions
type FeedVehicle struct {
	VehicleKey  `gorm:"embedded"`
	Year               int     `gorm:"column:year"`
	Make               string  `gorm:"column:make"`
	Model              string  `gorm:"column:model"`
	Trim               string  `gorm:"column:trim"`
	Body               string  `gorm:"column:body_style"`
	Doors              int     `gorm:"column:doors"`
	InteriorColour     string  `gorm:"column:interior_colour"`
	ExteriorColour     string  `gorm:"column:exterior_colour"`
	IntColourGeneric   string  `gorm:"column:interior_colour_generic"`
	ExtColourGeneric   string  `gorm:"column:exterior_colour_generic"`
	Configuration      string  `gorm:"column:configuration"`
	Cylinders          int     `gorm:"column:cylinders"`
	Displacement       float64 `gorm:"column:displacement"`
	Fuel               string  `gorm:"column:fuel_type"`
	TransmissionType   string  `gorm:"column:transmission_type"`
	TransmissionSpeeds int     `gorm:"column:transmission_speeds"`
	TransmissionDesc   string  `gorm:"column:transmission_description"`
	Drive              string  `gorm:"column:drivetrain"`
	Odometer           int     `gorm:"column:odometer"`
	Price              float64 `gorm:"column:price"`
	MSRP               float64 `gorm:"column:msrp"`
	Description        string  `gorm:"column:description"`
	Passengers         int     `gorm:"column:passengers"`
}
