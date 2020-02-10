package importer

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/seamuncle/dealer" // relative pathing would be cool, but that's not what Go does
)

// Importer represents a specific understanding for importing a specific feed's data.
// It provides methods to aquire data for an import, a means of staging aquired data
// into an arraoy of records and a means of translating a record into a Vehicle
// for further processing by a FullReplaceRunner
type Importer interface {
	// AquireRecords providers an Implementation the opportunity aquire new records
	// to a file specified by filename in a working directory chosen by the Importer
	AquireRecords(filename string) error
	// HasAquired reports if a successful call to AquireRecords has previouly occurred--
	// mostly it reports a matching filename exists in a working directory chosen by the Importer
	// If it returns false, a caller should make a point of calling AquireRecords again
	// before further processing occurs
	HasAquired(filename string) bool
	// LoadRecords takes aquired records from the filename specifid and returns them as an
	// array of *something* that the same implementation's ProcessRecord understands.
	// Note that its feasible AquireRecords was not run by the same process but may have been
	// called by another process with saveAquisition set to true
	LoadRecords(filename string) ([]interface{}, error)
	// ProcessRecord takes a sungle element from the array of *something* generated by LoadRecords
	// and turns it into a dealer.Vehicle for further processing by the FullReplaceRunner
	ProcessRecord(record interface{}) (dealer.Vehicle, error)
}

// Config sets default behaviors when calling an Importor or FullReplaceRunner
type Config struct {
	DoProcessing bool
	Filename     string
}

// FullReplaceRunner applies the logic of a rull-replacement import, given a specific Importer implementation
type FullReplaceRunner struct {
	Config Config
}

// Run is our heavy-lifter.  Take some cues from "big data" and apply some functional programming
// across the data set, though we're not actually doing big data processing, and gain none of the 
// cache-consistency that result from a big-data approach.  It does leave our Importer quite testable
// and if I had time to generate mocks and tests; this should be relatively testable as well.
func (runner FullReplaceRunner) Run(importer Importer, db *gorm.DB) error {

	filename := runner.Config.Filename
	if !importer.HasAquired(filename) {
		importer.AquireRecords(filename)
	}

	if !runner.Config.DoProcessing {
		return nil
	}

	records, err := importer.LoadRecords(filename)
	if err != nil {
		return fmt.Errorf("Loading records: %w", err)
	}

	var set InventorySet

	for i, record := range records {
		vehicle, err := importer.ProcessRecord(record)
		if err != nil {
			return fmt.Errorf("Processing record %d: %w", i, err)
		}

		lot := set.Lot()
		if lot != vehicle.Lot {
			// Cheating here--there is no d_id == 0 so its easy to tell when we're on the first record
			if lot.DealerID != 0 {
				// Before the lot changes, capture the state of the InventorySet
				set.FullReplace(db)
			}

			set = NewInventorySet(vehicle.Lot, db)
		}

		// All the bits have been extracted at this point
		matchingVehicle, found := set.MatchingVehicle(vehicle.VehicleKey)
		now := time.Now()
		if !found {
			vehicle.TheGuilty = "IMPORT"
			vehicle.LastModified = now
			vehicle.Created = now
			vehicle.State = dealer.StateUnknown
		} else if matchingVehicle.FeedVehicle != vehicle.FeedVehicle {
			// Struct equivalency and assignment is a convenient Go hack
			// Its also how data you may not want set, gets unset--but
			// for this exercise, assume the feed is sourse of truth
			matchingVehicle.FeedVehicle = vehicle.FeedVehicle
			vehicle = matchingVehicle
			vehicle.TheGuilty = "IMPORT"
			vehicle.LastModified = now
			vehicle.State = dealer.StateAltered
		} else {
			vehicle = matchingVehicle
			vehicle.State = dealer.StateUnaltered
		}
		set.SetVehicle(vehicle)
	}

	// Capture the state of the InventorySet after the last Lot in the feed
	return set.FullReplace(db)
}
