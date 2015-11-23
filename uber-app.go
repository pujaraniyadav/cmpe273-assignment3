package main
	
import (
	"errors"
	"fmt"
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"github.com/gorilla/mux"
	"encoding/json"
	"github.com/anweiss/uber-api-golang/uber"
)

//
// Uber sandbox API
//
func HttpGet(endpoint string, params map[string]string) ([]byte, error) {
	urlParams := ""
	url := fmt.Sprintf("https://sandbox-api.uber.com/v1/%s%s", endpoint, urlParams)

	body, err := json.Marshal(params)
	if err != nil {
		return nil,err
	}

	req,_ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Add("Authorization", "Bearer wvL_d5XSDnAARmPuSj9hGVYsjgSr91zoO0KFYCg6")
	req.Header.Add("Content-Type", "application/json")
	fmt.Println(req)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil,err
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil,err
	}

	res.Body.Close()                                                

	fmt.Println(string(data[:]))

	return data,nil
}

type SandboxRequest struct {
	ProductId 		string 		`json:"product_id"`
	StartLatitude 	float64 	`json:"start_latitude"`
	StartLongitude 	float64 	`json:"start_longitude"`
	EndLatitude 	float64 	`json:"end_latitude"`
	EndLongitude 	float64 	`json:"end_longitude"`
}

type SandboxResponse struct {
	RequestId 		string 		`json:"request_id"`
	Status 			string 		`json:"status"`
	Vehicle 		string 		`json:"vehicle"`
	Driver 			string 		`json:"driver"`
	Location 		string 		`json:"location"`
	ETA 			string 		`json:"eta"`
	SurgeMultiplier string 		`json:"surge_multiplier"`
}

func GetRequest(req SandboxRequest, response *SandboxResponse) error {
	params := map [string]string{
		"product_id": 	 	req.ProductId,
		"start_latitude":  	strconv.FormatFloat(req.StartLatitude, 'f', 2, 32),
		"start_longitude": 	strconv.FormatFloat(req.StartLongitude, 'f', 2, 32),
		"end_latitude":    	strconv.FormatFloat(req.EndLatitude, 'f', 2, 32),
		"end_longitude":   	strconv.FormatFloat(req.EndLongitude, 'f', 2, 32),
	}

	data,err := HttpGet("requests", params)
	if err != nil {
		return err
	}
	
	err = json.Unmarshal(data, response)
	return err
}

//
// Trip database
//
type Trip struct {
	id int
	status string
	start string
	next int
	routes []string
	product_ids []string
	cost_estimate int
	distance_estimate float64
	duration_estimate int
}


type Location struct {
	address string
	lat float64
	long float64
}

var _trips map[int]Trip
var _locations map[string]Location
var _id int

func InitLocations() {
	_locations = map[string]Location{}
	_locations["0"] = Location{"Fairmont Hotel San Francisco (950 Mason St, San Francisco, CA 94108)", 37.792496, -122.410035}
	_locations["1"] = Location{"Golden Gate Bridge, California", 37.819929, -122.478255}
	_locations["2"] = Location{"Pier 39 (Beach Street & The Embarcadero, San Francisco, CA 94133)", 37.799263, -122.397673}
	_locations["3"] = Location{"Golden Gate Park", 37.769040, -122.483519}
	_locations["4"] = Location{"Twin Peaks (501 Twin Peaks Blvd, San Francisco, CA 94114)", 37.752556, -122.447619}
}


//
// Uber 
//
func GetUberClient() *uber.Client {
	uber_options := uber.RequestOptions {
  		ServerToken: "wvL_d5XSDnAARmPuSj9hGVYsjgSr91zoO0KFYCg6",
  		ClientId: "WO12XDLKoKlKeHJIMw_oNvpqOm30lteG",
  		ClientSecret: "NdktLcJcLmgVGo6-rcYFkFrvFYWAcIKgnBfixoIc",
  		AppName: "PujaAssignment",
  	}
	
	return uber.Create(&uber_options)	
}

//
// REST handlers
//
func HandleError(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method, " ", req.URL)
	http.Error(w, "Invalid request", http.StatusBadRequest)
}

type TripRequest struct {
	STARTING_FROM_LOCATION_ID 	string 		`json:"starting_from_location_id"`
	LOCATION_IDS []				string 		`json:"location_ids"`
}	

type TripResponse struct {
	ID 							int 		`json:"id"`
	STATUS 						string 		`json:"status"`
	STARTING_FROM_LOCATION_ID 	string 		`json:"starting_from_location_id"`
	BEST_ROUTE_LOCATION_IDS 	[]string 	`json:"best_route_location_ids"`
	TOTAL_UBER_COST 			int 		`json:"total_uber_cost"`
	TOTAL_UBER_DURATION 		int 		`json:"total_uber_duration"`
	TOTAL_DISTANCE 				float64		`json:"total_distance"`
}

type TripResponse2 struct {
	ID 								int 		`json:"id"`
	STATUS 							string 		`json:"status"`
	STARTING_FROM_LOCATION_ID 		string 		`json:"starting_from_location_id"`
	NEXT_DESTINATION_LOCATION_ID	string 		`json:"next_destination_location_id"`
	BEST_ROUTE_LOCATION_IDS 		[]string 	`json:"best_route_location_ids"`
	TOTAL_UBER_COST 				int 		`json:"total_uber_cost"`
	TOTAL_UBER_DURATION 			int 		`json:"total_uber_duration"`
	TOTAL_DISTANCE 					float64		`json:"total_distance"`
	ETA 							int 		`json:"eta"`
}


func GetETA(product_id string, lat float64, long float64) (int,error) {
	te := &uber.TimeEstimates{}
	te.StartLatitude = lat
	te.StartLongitude = long
	client := GetUberClient()
	if e := client.Get(te); e != nil {
		return 0,e;
	}

	for _, eta := range te.Times {
		fmt.Println(eta.ProductId + ": " + strconv.Itoa(eta.Estimate/60))			
		return eta.Estimate/60,nil
	}

	return 0,errors.New("Product ID not found")
}

func BestP2P(start string, routes []string) string {
	i := 0
	end := routes[i]
	_,cost,duration,_,_ := Calc(start, end)
	path_cost := cost * duration
	i = i + 1
	for i < len(routes) {
		t := routes[i]
		_,cost,duration,_,_ = Calc(start, t)
		t_cost := cost * duration
		if t_cost < path_cost {
			path_cost = t_cost
			end = t
		}
		i = i + 1
	}
	return end
}

func difference(slice1 []string, slice2 []string) ([]string){
    diffStr := []string{}
    m :=map [string]int{}

    for _, s1Val := range slice1 {
        m[s1Val] = 1
    }
    for _, s2Val := range slice2 {
        m[s2Val] = m[s2Val] + 1
    }

    for mKey, mVal := range m {
        if mVal==1 {
            diffStr = append(diffStr, mKey)
        }
    }

    return diffStr
}

func BestRoute(start string, routes []string) []string {
	begin := start
	ret := []string{}
	i := 0
	for i < len(routes) {
		t := difference(routes, ret)
		ret = append(ret, BestP2P(begin, t))
		begin = routes[i]
		i = i + 1
	}

	return ret
}

func Calc(start string, end string) (string,int,int,float64,error) {
	start_loc,ok := _locations[start]
	if !ok {
		return "",0,0,0,errors.New("Location not found")
	}

	end_loc,ok := _locations[end]
	if !ok {
		return "",0,0,0,errors.New("Location not found")
	}

	fmt.Println("Sending request to uber")

	client := GetUberClient()
	pe := &uber.PriceEstimates{}
	pe.StartLatitude = start_loc.lat
	pe.StartLongitude = start_loc.long
	pe.EndLatitude = end_loc.lat
	pe.EndLongitude = end_loc.long
	if e := client.Get(pe); e != nil {
		fmt.Println("Error talking with uber.", e)
		return "",0,0,0,e
	}

	cost := 0
	duration := 0
	distance := 0.0
	product_id := ""
	iter0 := true
	for _, price := range pe.Prices {
		if price.Estimate == "Metered" {
			continue
		}

		fmt.Println(price.ProductId, ":", price.Estimate)
		
		if iter0  {
			product_id = price.ProductId
			cost = price.HighEstimate
			duration = price.Duration / 60
			distance = price.Distance
			iter0 = false
		} else if price.HighEstimate < cost {
			product_id = price.ProductId
			cost = price.HighEstimate
			duration = price.Duration / 60
			distance = price.Distance
		}
	}
	
	return product_id,cost,duration,distance,nil
}

func CalcRoute(req TripRequest) ([]string,[]string, int, int, float64, error) {
	route := BestRoute(req.STARTING_FROM_LOCATION_ID, req.LOCATION_IDS)
	product_ids := []string{}
	cost := 0
	distance := 0.0
	duration := 0
	start := req.STARTING_FROM_LOCATION_ID
	for _,loc := range route {
		pid,price,time,dist,err := Calc(start, loc)
		if err != nil {
			return nil,nil,0,0,0,err
		}
		product_ids = append(product_ids, pid)
		cost += price
		distance += dist
		duration += time
		start = loc
	} 

	fmt.Println(product_ids)

	return product_ids,route,cost,duration,distance,nil
}

func HandleCreateTrips(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method, " ", req.URL)
	
	//
	// Decode request
	//
	decoder := json.NewDecoder(req.Body)
	var trip_req TripRequest
	err := decoder.Decode(&trip_req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	fmt.Println("Got Request. ", trip_req)

	var trip_res TripResponse

	product_ids,routes,cost,duration,distance,err := CalcRoute(trip_req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	trip := Trip{
		_id,
		"planning",
		trip_req.STARTING_FROM_LOCATION_ID,
		0,
		routes,
		product_ids,
		cost,
		distance,
		duration,
	}

	_trips[_id] = trip

	trip_res.ID = _id
	trip_res.STATUS = "planning"
	trip_res.STARTING_FROM_LOCATION_ID = trip_req.STARTING_FROM_LOCATION_ID
	trip_res.BEST_ROUTE_LOCATION_IDS = routes
	trip_res.TOTAL_UBER_COST = cost
	trip_res.TOTAL_UBER_DURATION = duration
	trip_res.TOTAL_DISTANCE = distance

	_id = _id + 1

	out,err := json.Marshal(trip_res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//
	// Write result
	//
	w.WriteHeader(http.StatusCreated)
	w.Write(out)
}

func HandleGetTrips(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method, " ", req.URL, " ", mux.Vars(req))

	id,_ := strconv.Atoi(mux.Vars(req)["trip_id"])

	trip,ok := _trips[id]
	if !ok {
		http.Error(w, "trip id not found", http.StatusInternalServerError)
		return	
	}

	var trip_res TripResponse
	trip_res.ID = trip.id
	trip_res.STATUS = trip.status
	trip_res.STARTING_FROM_LOCATION_ID = trip.start
	trip_res.BEST_ROUTE_LOCATION_IDS = trip.routes
	trip_res.TOTAL_UBER_COST = trip.cost_estimate
	trip_res.TOTAL_UBER_DURATION = trip.duration_estimate
	trip_res.TOTAL_DISTANCE = trip.distance_estimate


	out,err := json.Marshal(trip_res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//
	// Write result
	//
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func HandleTripsRequest(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method, " ", req.URL, " ", mux.Vars(req))

	id,_ := strconv.Atoi(mux.Vars(req)["trip_id"])

	trip,ok := _trips[id]
	if !ok {
		http.Error(w, "trip id not found", http.StatusInternalServerError)
		return	
	}

	if trip.next > len(trip.routes) {
		http.Error(w, "trip complete", http.StatusInternalServerError)
		return
	}

	var curr Location
	var product_id string
	if trip.next == len(trip.routes) {
		product_id = trip.product_ids[trip.next - 1] 
		curr = _locations[trip.routes[trip.next - 1]]
	} else {
		product_id = trip.product_ids[trip.next]
		curr = _locations[trip.routes[trip.next]]
	}

	eta,err := GetETA(product_id, curr.lat, curr.long)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return	
	}
	
	var next_id string
	if trip.next == len(trip.routes) {
		next_id = trip.start
	} else {
		next_id = trip.routes[trip.next]
	}
	trip.next = trip.next + 1
	trip.status = "requesting"
	_trips[id] = trip

	var trip_res TripResponse2
	trip_res.ID = trip.id
	trip_res.STATUS = trip.status
	trip_res.STARTING_FROM_LOCATION_ID = trip.start
	trip_res.NEXT_DESTINATION_LOCATION_ID = next_id
	trip_res.BEST_ROUTE_LOCATION_IDS = trip.routes
	trip_res.TOTAL_UBER_COST = trip.cost_estimate
	trip_res.TOTAL_UBER_DURATION = trip.duration_estimate
	trip_res.TOTAL_DISTANCE = trip.distance_estimate
	trip_res.ETA = eta



	out,err := json.Marshal(trip_res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//
	// Write result
	//
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func StartHttpServer(addr string) {
	router := mux.NewRouter()
	router.HandleFunc("/trips", HandleCreateTrips).Methods("POST")
	router.HandleFunc("/trips/{trip_id:[0-9]+}", HandleGetTrips).Methods("GET")
	router.HandleFunc("/trips/{trip_id:[0-9]+}/request", HandleTripsRequest).Methods("PUT")
	router.HandleFunc("/", HandleError)

	http.Handle("/", router)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	_id = 0
	_trips = map[int]Trip{}
	InitLocations()  
	StartHttpServer(":12345")
}
