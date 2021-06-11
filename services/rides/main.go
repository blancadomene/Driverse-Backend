package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gorilla/mux"
)

type Ride struct {
	ID                  string `json:"id"`
	Driver              string `json:"driver"`
	StartDate           string `json:"startDate"`
	EndDate             string `json:"endDate"`
	DeparturePoint      string `json:"departurePoint"`
	DepartureLatLng     string `json:"departureLatLng"`
	DepartureHour       string `json:"departureHour"`
	ArrivalPoint        string `json:"arrivalPoint"`
	ArrivalLatLng       string `json:"arrivalLatLng"`
	ArrivalHour         string `json:"arrivalHour"`
	AvailableSeats      string `json:"availableSeats"`
	PricePerSeat        string `json:"pricePerSeat"`
	AvailableDaysOfWeek string `json:"availableDaysOfWeek"`
}

var database *sql.DB

func getMatchingRidesInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	start := r.FormValue("StartDate")
	end := r.FormValue("EndDate")
	depLatLng := r.FormValue("DepartureLatLng")
	depLatLngSplit := strings.Split(depLatLng, ",")
	//departurePointRadius := r.FormValue("DeparturePointRadius")
	arrLatLng := r.FormValue("ArrivalLatLng")
	arrLatLngSplit := strings.Split(arrLatLng, ",")
	//arrivalPointRadius := r.FormValue("ArrivalPointRadius")
	depHour := r.FormValue("DepartureHour")
	numbSeats := r.FormValue("NumberOfSeats")
	//avWeek := r.FormValue("AvailableDaysOfWeek")

	// Do smth with latlng and radius
	// Do smth with avWeek
	// NumSeats AND availableseats
	query := fmt.Sprintf(`
		SELECT *
		FROM rides
		WHERE (
			StartDate = STR_TO_DATE("%s", "%s") AND
			EndDate = STR_TO_DATE("%s", "%s") AND 
			DepartureLat = "%s" AND DepartureLng = "%s" AND 
			ArrivalLat = "%s" AND ArrivalLng = "%s" AND 
			DepartureHour = "%s" AND 
			AvailableSeats = "%s"
		)`,
		start, "%d/%m/%Y",
		end, "%d/%m/%Y",
		depLatLngSplit[0], depLatLngSplit[1],
		arrLatLngSplit[0], arrLatLngSplit[1],
		depHour,
		numbSeats)

	fmt.Println(query)
	results, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	for results.Next() {

		var ride Ride
		var departureLat, departureLng float32
		var arrivalLat, arrivalLng float32
		err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &departureLat, &departureLng, &ride.DepartureHour, &ride.ArrivalPoint, &arrivalLat, &arrivalLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
		if err != nil {
			log.Fatal(err)
		}

		ride.DepartureLatLng = fmt.Sprintf("%f,%f", departureLat, departureLng)
		ride.ArrivalLatLng = fmt.Sprintf("%f,%f", arrivalLat, arrivalLng)

		err := json.NewEncoder(w).Encode(ride)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getRideInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rideID := r.FormValue("ID")

	query := fmt.Sprintf("SELECT * FROM rides WHERE (ID = \"%s\")", rideID)
	results, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	for results.Next() {
		var ride Ride
		var departureLat, departureLng float32
		var arrivalLat, arrivalLng float32
		err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &departureLat, &departureLng, &ride.DepartureHour, &ride.ArrivalPoint, &arrivalLat, &arrivalLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
		if err != nil {
			log.Fatal(err)
		}

		ride.DepartureLatLng = fmt.Sprintf("%f,%f", departureLat, departureLng)
		ride.ArrivalLatLng = fmt.Sprintf("%f,%f", arrivalLat, arrivalLng)

		err := json.NewEncoder(w).Encode(ride)
		if err != nil {
			log.Fatal(err)
		}
		break
	}
}

func postRideInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var ride Ride
	err := json.NewDecoder(r.Body).Decode(&ride)
	if err != nil {
		log.Fatal(err)
	}

	DepartureLatLngSplit := strings.Split(ride.DepartureLatLng, ",")
	//DepartureLat, _ := strconv.ParseFloat(DepartureLatLngSplit[0], 32)
	//DepartureLng, _ := strconv.ParseFloat(DepartureLatLngSplit[1], 32)

	ArrivalLatLngSplit := strings.Split(ride.ArrivalLatLng, ",")
	//ArrivalLat, _ := strconv.ParseFloat(ArrivalLatLngSplit[0], 32)
	//ArrivalLng, _ := strconv.ParseFloat(ArrivalLatLngSplit[1], 32)

	query := fmt.Sprintf("INSERT INTO rides VALUES (\"%s\", \"%s\", STR_TO_DATE(\"%s\", \"%s\"), STR_TO_DATE(\"%s\", \"%s\"), \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\")",
		ride.ID, ride.Driver, ride.StartDate, "%d/%m/%Y", ride.EndDate, "%d/%m/%Y", ride.DeparturePoint, DepartureLatLngSplit[0], DepartureLatLngSplit[1], ride.DepartureHour, ride.ArrivalPoint, ArrivalLatLngSplit[0], ArrivalLatLngSplit[1], ride.ArrivalHour, ride.AvailableSeats, ride.PricePerSeat, ride.AvailableDaysOfWeek)

	insert, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	_ = insert
	defer insert.Close()
}

func main() {
	var err error

	database, err = sql.Open("mysql", "root:secret@tcp(database:3306)/driverse")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	var router = mux.NewRouter()
	router.
		Methods(http.MethodGet).
		Path("/info").
		Queries(
			"StartDate", "{StartDate}",
			"EndDate", "{EndDate}",
			"DepartureLatLng", "{DepartureLatLng}",
			"DeparturePointRatius", "{DeparturePointRatius}",
			"ArrivalLatLng", "{ArrivalLatLng}",
			"ArrivalPointRatius", "{ArrivalPointRatius}",
			"DepartureHour", "{DepartureHour}",
			"NumberOfSeats", "{NumberOfSeats}",
			"AvailableDaysOfWeek", "{AvailableDaysOfWeek}",
		).
		HandlerFunc(getMatchingRidesInfo)

	router.
		Methods(http.MethodPost).
		Path("/info").
		HandlerFunc(postRideInfo)

	router.
		Methods(http.MethodGet).
		Path("/info").
		Queries("ID", "{ID}").
		HandlerFunc(getRideInfo)

	fmt.Println("Listening at 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
