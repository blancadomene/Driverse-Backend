package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
	//departurePointRadius := r.FormValue("DeparturePointRadius")
	arrLatLng := r.FormValue("ArrivalLatLng")
	//arrivalPointRadius := r.FormValue("ArrivalPointRadius")
	depHour := r.FormValue("DepartureHour")
	numbSeats := r.FormValue("NumberOfSeats")
	avWeek := r.FormValue("AvailableDaysOfWeek")

	// Do smth with latlng and radius
	query := fmt.Sprintf("SELECT * FROM rides WHERE (StartDate = STR_TO_DATE(\"%s\", \"%s\") AND EndDate = STR_TO_DATE(\"%s\", \"%s\") AND DepartureLatLng = \"%s\" AND ArrivalLatLng = \"%s\" AND DepartureHour = \"%s\" AND AvailableSeats = \"%s\" AND AvailableDaysOfWeek = \"%s\")",
		start, "%d/%m/%Y", end, "%d/%m/%Y", depLatLng, arrLatLng, depHour, numbSeats, avWeek)
	results, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	for results.Next() {

		var ride Ride
		err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &ride.DepartureLatLng, &ride.DepartureHour, &ride.ArrivalPoint, &ride.ArrivalLatLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
		if err != nil {
			log.Fatal(err)
		}

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
		err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &ride.DepartureLatLng, &ride.DepartureHour, &ride.ArrivalPoint, &ride.ArrivalLatLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
		if err != nil {
			log.Fatal(err)
		}

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

	query := fmt.Sprintf("INSERT INTO rides VALUES (\"%s\", \"%s\", STR_TO_DATE(\"%s\", \"%s\"), STR_TO_DATE(\"%s\", \"%s\"), \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\")",
		ride.ID, ride.Driver, ride.StartDate, "%d/%m/%Y", ride.EndDate, "%d/%m/%Y", ride.DeparturePoint, ride.DepartureLatLng, ride.DepartureHour, ride.ArrivalPoint, ride.ArrivalLatLng, ride.ArrivalHour, ride.AvailableSeats, ride.PricePerSeat, ride.AvailableDaysOfWeek)

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
			"ArrivalLatLng", "{ArrivalLatLng}",
			"DepartureHour", "{DepartureHour}",
			"AvailableSeats", "{AvailableSeats}",
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
