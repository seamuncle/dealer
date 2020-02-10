package importer

import (
	"github.com/jinzhu/gorm"
	"github.com/seamuncle/dealer"
)

// NewInventorySet does what it says on the box
func NewInventorySet(lot dealer.Lot, db *gorm.DB) InventorySet {
	set := InventorySet{
		lot:      lot,
		vehicles: map[dealer.VehicleKey]dealer.Vehicle{},
	}

	var vehicles []*dealer.Vehicle
	db.Where("d_id = ? AND stock_type = ?", lot.DealerID, lot.LotType).Find(&vehicles)
	for _, vehicle := range vehicles {
		// StatePersisted is the default, but lets be explicit for clarity
		vehicle.State = dealer.StatePersisted
		set.SetVehicle(*vehicle)
	}
	return set
}

// InventorySet  uses a map of fully populated and partially populated VehicleKeys to reference Vehicle entries
// Assumption: at least one of VIN or stock will be present for both a record and a previously persisted vehicle
type InventorySet struct {
	lot      dealer.Lot
	vehicles map[dealer.VehicleKey]dealer.Vehicle
}

// FullReplace performsa full replacement import based on the VehicleState of all of its elements
// and then updates the database accordingly.
func (set InventorySet) FullReplace(db *gorm.DB) error {

	// StateUnknown indeicates probably not in the database
	Unknowns := []dealer.Vehicle{}
	// StatePersisted indicates this vehicle has been loaded directly from its db counterpart
	Persisteds := []dealer.Vehicle{}
	// StateAltered indicates there is a difference between a feed vehicle and its db counterpart
	Altereds := []dealer.Vehicle{}

	for key, vehicle := range set.vehicles {
		// Dedupe vehicles referred to by synthetically partial keys
		if key != vehicle.VehicleKey {
			continue
		}

		// Add vehicle to its collection according to state
		// Note that dealer.StateUnaltered vehicles require no further action
		switch vehicle.State {
		case dealer.StateUnknown:
			Unknowns = append(Unknowns, vehicle)
		case dealer.StatePersisted:
			Persisteds = append(Persisteds, vehicle)
		case dealer.StateAltered:
			Altereds = append(Altereds, vehicle)
		}
	}

	// run database operations based on vehicle state...
	// The Unknowns should be inserted to dealer inventoryas they are unknown to the system
	// The Persisteds should be deleted as have not been deemed Unaltered, which means they exist only in the DB
	// The Altereds should be updated as there is some descrepency between the DB and the feed
	for _, vehicle := range Unknowns {
		db.Create(&vehicle)
	}
	for _, vehicle := range Persisteds {
		db.Delete(&vehicle)
	}

	model := db.Model(&dealer.Vehicle{})
	for _, vehicle := range Altereds {
		model.Updates(vehicle)
	}

	return nil
}

// Lot returns the lot asociated wtih the InventorySet
func (set InventorySet) Lot() dealer.Lot {
	return set.lot
}

// SetVehicle adds the given vehicle to the InventorySet
func (set InventorySet) SetVehicle(vehicle dealer.Vehicle) {
	// If an existing key is better, run with that
	key := set.bestKey(vehicle.VehicleKey)

	// while the set is copied, the map is implicitly a pointer, so we can write to the original map
	// using a copy of the set.  Yay!  Go's half-assed data protection
	set.vehicles[key] = vehicle
	if len(key.VIN) != 0 {
		set.vehicles[dealer.VehicleKey{VIN: key.VIN, Stock: ""}] = vehicle
	}
	if len(key.Stock) != 0 {
		set.vehicles[dealer.VehicleKey{VIN: "", Stock: key.Stock}] = vehicle
	}
}

// convenience method that looks for a better key in the event one passeed is partial
func (set InventorySet) bestKey(key dealer.VehicleKey) dealer.VehicleKey {
	best := key
	if len(best.VIN) == 0 || len(best.Stock) == 0 {
		if vehicle, ok := set.MatchingVehicle(key); ok {
			// There's no promise this is actually better; but given one of the current keys is unset; it can't be worse
			best = vehicle.VehicleKey
		}
	}
	return best
}

// ClearVehicle removes the given vehicle from the InventorySet
// Turns out to be unnecessary, but completeness
func (set InventorySet) ClearVehicle(vehicle dealer.Vehicle) {
	key := set.bestKey(vehicle.VehicleKey)
	// while the set is copied, the map is implicitly a pointer, so we can write to the original map
	// using a copy of the set.  Yay!  Go's half-assed data protection
	delete(set.vehicles, key)
	delete(set.vehicles, dealer.VehicleKey{VIN: key.VIN, Stock: ""})
	delete(set.vehicles, dealer.VehicleKey{VIN: "", Stock: key.Stock})
}

// MatchingVehicle looks for the given vehicle in the InventorySet and if found,
// returns true and the vehicle.  If not found the Vehicle it returns false and the
// vehicle has a zero-vehicle characteristics
func (set InventorySet) MatchingVehicle(key dealer.VehicleKey) (dealer.Vehicle, bool) {
	if v, ok := set.vehicles[key]; ok {
		return v, true
	}
	// Check for vin only
	if len(key.VIN) != 0 {
		if v, ok := set.vehicles[dealer.VehicleKey{VIN: key.VIN, Stock: ""}]; ok {
			return v, true
		}
	}
	// Check for stock only
	if len(key.Stock) != 0 {
		if v, ok := set.vehicles[dealer.VehicleKey{VIN: "", Stock: key.Stock}]; ok {
			return v, ok
		}
	}
	return dealer.Vehicle{}, false
}
