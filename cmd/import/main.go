package main

// The import program represents a simple import script written in Go.
// The reason Go was selected was it happenes to be the only previously
// installed programming environment on my gaming PC, and I was feeling
// unmotivated to install compilers and IDEs there.
import (
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/seamuncle/dealer"
	"github.com/seamuncle/dealer/importer"
)

var config = importer.Config{}

// main is responisble for that really high-level stuff.
// On errors it does log.Fatal,
// It parses CLI flags and gets them where they need to go
// It instantiates a thing I called importer and feeds it to a thing I called a runner
func main() {
	flag.StringVar(&config.Filename, "file", "dealer_import.csv", "name of file this import is concerned with--with no prefix")
	flag.BoolVar(&config.DoProcessing, "process", true, "tells import to continue processing file once its been aquired")
	flag.Parse()

	// There's other approaches to DB initilization, but "things that fatal" belong in main
	db, err := gorm.Open("sqlite3", "file:dealer_import.db?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	db.LogMode(true)
	demo := DemoImporter{}
	runner := importer.FullReplaceRunner{
		Config: config,
	}

	if err := runner.Run(demo, db); err != nil {
		log.Fatal(err)
	}

	// Some people like to defer this close way up when it Opened,
	//  but really if the Close results in something going wrong, that should get logged
	if err = db.Close(); err != nil {
		log.Fatal(err)
	}
}

// DemoRecord would be the specific implementation for a record returned by LoadRecords
// and passed to ProcessRecord
// Its a set of specific CSV headers and CSV values--other imports could use other
// Records--the nitty-gritty of the representation is only a problem for a specific
// this could just as easily be a struct populated from JSON/XML or a map if that was
// in any way more simple or efficient
type DemoRecord struct {
	Headings []string
	Values   []string
}

// A greedy match here will xxtract the float portion of a string
var floatRegex = regexp.MustCompile(`[\.\d]+`)

// There's some goodness to extract about transmissions
var transmissionRegex = regexp.MustCompile(`(\d)-Spe*d (Automatic|Manual)`)

// DemoImporter is a concrete implementation of importer.Importer which knows to aquire data from
// gist.githubusercontent.com, and that said data will be a csv, and the specifics of the csv encoding,
// headers and how its values map into a dealer.Vehicle
type DemoImporter struct{}

// AquireRecords does an HTTP get to gist.githubusercontent.com and captures the passed filename as a local file.
// This is nothing like the real world--but we'll pretend naievely the only complexity is it might be desirable to
// retrieve differnt import files from the same path;
// not every lot is supposed to have their own file, addressed by a remote id; except dealer Bob, who for historical
// reasons has 3 files describing 1 lot...
func (i DemoImporter) AquireRecords(filename string) error {
	uri := "https://gist.githubusercontent.com/mm53bar/26bd794c9245191f7407a5c7441c4969/raw/87df2a61b650a43001c875cb203df7929580ba90/" + filename
	resp, err := http.Get(uri)
	if err != nil {
		return fmt.Errorf("Getting records at %s: %w", uri, err)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Reading all of http response: %w", err)
	}

	if err = ioutil.WriteFile(workingFileName(filename), b, 0644); err != nil {
		return fmt.Errorf("Writing http response to file %s: %w", workingFileName(filename), err)
	}

	return nil
}

// HasAquired looks in the place AquireRecords dropped its file and checks its there and at least
// one byte of it can be read--anything indicating this is not the case, will result in it returning false
func (i DemoImporter) HasAquired(filename string) bool {
	file, err := os.Open(workingFileName(filename))
	if err != nil {
		return false
	}

	smallbuffer := make([]byte, 1)
	n, err := file.Read(smallbuffer)
	if err != nil || n != 1 {
		return false
	}

	err = file.Close()
	if err != nil {
		return false
	}
	return true
}

// LoadRecords looks in the place AquireRecords dropped its file, opens it and uses the default
// golang CSV parser to make sense of it.  The classes in the returned interface are of type DemoRecord
func (i DemoImporter) LoadRecords(filename string) ([]interface{}, error) {

	reader, err := os.Open(workingFileName(filename))
	if err != nil {
		return nil, fmt.Errorf("Opening saved file for reading %s: %w", workingFileName(filename), err)
	}

	csvReader := csv.NewReader(reader)
	// read the title line
	headings, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("Reading csv headings: %w", err)
	}

	values, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Reading csv values: %w", err)
	}

	err = reader.Close()
	if err != nil {
		return nil, fmt.Errorf("Closing saved file for reading %s: %w", workingFileName(filename), err)
	}

	// Turn into values for ProcessRecord
	records := make([]interface{}, len(values))
	for i, value := range values {
		records[i] = DemoRecord{
			Headings: headings,
			Values:   value,
		}
	}

	return records, nil
}

// ProcessRecord takes a DemoRecord as returned by LoadRecords and
// after casting it appropriately, iterates across all the headers in the record
// --mapping each to the corresponding value and determining what it goes on a
// dealer.Vehcile via a switch statement.  I don't even know where to start with
// real world complexities here, but our example is naievely quite similar and a
// simple switch with trivial error handling seemed best.
func (i DemoImporter) ProcessRecord(record interface{}) (dealer.Vehicle, error) {
	// This version of match is build around redord looking like
	// a string slice representing keys and a string slice representing values
	// the keys will all be mapped to a function provided by Matcher which in turn
	// will update the appropriate structs

	// We could do a graceful typecast here, but I'd just as soon explode given the context
	demoRecord := record.(DemoRecord)

	// this is going to hold all the processed record values
	vehicle := dealer.Vehicle{}

	for i, heading := range demoRecord.Headings {
		value := demoRecord.Values[i]

		// also explode rather than gracefully checking the map--there's no elegant solution better than this
		switch heading {
		case "DealerID":
			dealerID, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse DealerID (%s): %w", value, err)
			}
			vehicle.DealerID = int(dealerID)

		case "DealerName":
			vehicle.DealerName = value

		case "Type":
			if value == "New" {
				vehicle.LotType = dealer.TypeNew
			} else {
				vehicle.LotType = dealer.TypeUsed
			}

		case "Stock":
			vehicle.Stock = value

		case "VIN":
			vehicle.VIN = value

		case "Year":
			year, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse Year (%s): %w", value, err)
			}
			vehicle.Year = int(year)

		case "Make":
			vehicle.Make = value

		case "Model":
			vehicle.Model = value

		case "Trim":
			vehicle.Trim = value

		case "Body":
			vehicle.Body = value

		case "Doors":
			doors, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse Doors (%s): %w", value, err)
			}
			vehicle.Doors = int(doors)

		case "ExtColor":
			vehicle.ExteriorColour = value

		case "IntColor":
			vehicle.InteriorColour = value

		case "EngCylinders":
			cylinders, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse EngCylinders (%s): %w", value, err)
			}
			vehicle.Cylinders = int(cylinders)

		case "EngDisplacement":
			f := floatRegex.FindString(value)
			displacement, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse EngCylinders (%s): %w", value, err)
			}
			vehicle.Displacement = displacement

		case "Transmission":
			vehicle.TransmissionDesc = value
			if value == "CVT" {
				vehicle.TransmissionType = "CVT"
				continue
			}
			// This regex shoud return 3 submatches if we're going to trust it
			bits := transmissionRegex.FindStringSubmatch(value)
			if len(bits) == 3 {
				speeds, err := strconv.ParseInt(bits[1], 10, 32)
				if err != nil {
					return vehicle, fmt.Errorf("Unable to extract Transmission speeds (%s): %w", value, err)
				}
				vehicle.TransmissionSpeeds = int(speeds)
				vehicle.TransmissionType = bits[2]
			}

		case "Odometer":
			odometer, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse Odometer (%s): %w", value, err)
			}
			vehicle.Cylinders = int(odometer)

		case "Price":
			f := floatRegex.FindString(value)
			price, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse Price (%s): %w", value, err)
			}
			vehicle.Price = price

		case "MSRP":
			f := floatRegex.FindString(value)
			msrp, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse MSRP (%s): %w", value, err)
			}
			vehicle.MSRP = msrp

		case "Certified":
			// Mark the heading as understood and No-op as vehcile has no correlating field

		case "DateInStock":
			// There's a solid case for this to be the vehicle.Created field; but that sounds like a discussion

		case "Description":
			// A special mention by any other name, will still drive you insane
			vehicle.Description = value

		case "EngType":
			vehicle.Configuration = value

		case "EngFuel":
			vehicle.Fuel = value

		case "Drivetrain":
			vehicle.Drive = value

		case "ExtColorGeneric":
			vehicle.ExtColourGeneric = value

		case "IntColorGeneric":
			vehicle.IntColourGeneric = value

		case "PassengerCount":
			passengers, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return vehicle, fmt.Errorf("Unable to parse PassengerCount (%s): %w", value, err)
			}
			vehicle.Passengers = int(passengers)

		default:
			return vehicle, fmt.Errorf("Unable to process unknown heading %s' in column %d", heading, i)
		}
	}

	return vehicle, nil
}

// utility method used by DemoImporter so all methods have a consistent means of globally addressing
// the passed filename
func workingFileName(filename string) string {
	return "/tmp/" + filename
}
