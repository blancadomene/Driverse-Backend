package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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
	AvailableDaysOfWeek int    `json:"availableDaysOfWeek"`
}

var (
	database *sql.DB

	requetsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rides_requests_total",
		Help: "Processed requests.",
	}) // TODO: One per each get/post API within this service
)

func getMatchingRidesInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	requetsCounter.Inc()

	start := r.FormValue("StartDate")
	end := r.FormValue("EndDate")
	depLatLng := r.FormValue("DepartureLatLng")
	depLatLngSplit := strings.Split(depLatLng, ",")
	departurePointRadius := r.FormValue("DeparturePointRadius")
	arrLatLng := r.FormValue("ArrivalLatLng")
	arrLatLngSplit := strings.Split(arrLatLng, ",")
	arrivalPointRadius := r.FormValue("ArrivalPointRadius")
	depHour := r.FormValue("DepartureHour")
	numbSeats := r.FormValue("NumberOfSeats")
	avWeek := r.FormValue("AvailableDaysOfWeek")

	// FUTURE WORK: Use function for distance: https://stackoverflow.com/a/48263512/16030066
	query := fmt.Sprintf(`
		SELECT *
		FROM rides
		WHERE (
			StartDate = STR_TO_DATE("%s", "%s") AND
			EndDate = STR_TO_DATE("%s", "%s") AND 
			111.111 * DEGREES(
				ACOS(
					LEAST(1.0, 
						COS(RADIANS(DepartureLat))
						* COS(RADIANS(%s))
						* COS(RADIANS(DepartureLng - %s))
						+ SIN(RADIANS(DepartureLat))
						* SIN(RADIANS(%s))
					)
				)
			) * 1000 <= %s AND
			111.111 * DEGREES(
				ACOS(
					LEAST(1.0, 
						COS(RADIANS(ArrivalLat))
						* COS(RADIANS(%s))
						* COS(RADIANS(ArrivalLng - %s))
						+ SIN(RADIANS(ArrivalLat))
						* SIN(RADIANS(%s))
					)
				)
			) * 1000 <= %s AND 
			DepartureHour = "%s" AND 
			AvailableSeats >= %s AND
			AvailableDaysOfWeek & %s = %s
		)`,
		start, "%d/%m/%Y",
		end, "%d/%m/%Y",
		depLatLngSplit[0], depLatLngSplit[1], depLatLngSplit[0],
		departurePointRadius,
		arrLatLngSplit[0], arrLatLngSplit[1], arrLatLngSplit[0],
		arrivalPointRadius,
		depHour,
		numbSeats,
		avWeek, avWeek)

	fmt.Println(query)
	results, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	var rides []Ride

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

		rides = append(rides, ride)
	}

	fmt.Printf("Rides: %x\n", rides)
	err = json.NewEncoder(w).Encode(rides)
	if err != nil {
		log.Fatal(err)
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

	if !results.Next() {
		log.Fatal("No record for ID")
	}

	var ride Ride
	var departureLat, departureLng float32
	var arrivalLat, arrivalLng float32
	err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &departureLat, &departureLng, &ride.DepartureHour, &ride.ArrivalPoint, &arrivalLat, &arrivalLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
	if err != nil {
		log.Fatal(err)
	}

	ride.DepartureLatLng = fmt.Sprintf("%f,%f", departureLat, departureLng)
	ride.ArrivalLatLng = fmt.Sprintf("%f,%f", arrivalLat, arrivalLng)

	err = json.NewEncoder(w).Encode(ride)
	if err != nil {
		log.Fatal(err)
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

	ArrivalLatLngSplit := strings.Split(ride.ArrivalLatLng, ",")

	query := fmt.Sprintf(`
		INSERT INTO rides
		VALUES (
			"%s",
			"%s",
			STR_TO_DATE("%s", "%s"),
			STR_TO_DATE("%s", "%s"),
			"%s",
			"%s", "%s",
			"%s",
			"%s",
			"%s", "%s",
			"%s",
			"%s",
			"%s",
			%d
		)`,
		ride.ID,
		ride.Driver,
		ride.StartDate, "%d/%m/%Y",
		ride.EndDate, "%d/%m/%Y",
		ride.DeparturePoint,
		DepartureLatLngSplit[0], DepartureLatLngSplit[1],
		ride.DepartureHour,
		ride.ArrivalPoint,
		ArrivalLatLngSplit[0], ArrivalLatLngSplit[1],
		ride.ArrivalHour,
		ride.AvailableSeats,
		ride.PricePerSeat,
		ride.AvailableDaysOfWeek)

	fmt.Println(query)

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
			"DeparturePointRadius", "{DeparturePointRadius}",
			"ArrivalLatLng", "{ArrivalLatLng}",
			"ArrivalPointRadius", "{ArrivalPointRadius}",
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

	router.Handle("/metrics", promhttp.Handler())

	fmt.Println("Listening at 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
