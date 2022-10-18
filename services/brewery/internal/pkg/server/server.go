package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jbowl/hodlmybeer/services/brewery/internal/pkg/types"

	"github.com/google/uuid"
	"github.com/jbowl/hodlapi"
	"github.com/jbowl/mtlscreds"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	//	"github.com/aws/aws-sdk-go-v2/aws/session"
	//"github.com/aws/aws-sdk-go-v2/service/ssm/ssmiface"
)

type Server struct {
	hodlapi.UnimplementedBreweryServiceServer
	APIUrl string
	client *http.Client
}

// uuid to QueryResult
//var queryResults map[string]types.QueryResult

//var queryResults []types.QueryResult

func init() {
}

func serverOpts(rootCA string, svrCert string, svrKey string) ([]grpc.ServerOption, error) {

	svrCreds, err := mtlscreds.SvrCreds(rootCA, svrCert, svrKey)

	if err != nil {
		return nil, err
	}

	return []grpc.ServerOption{grpc.Creds(svrCreds)}, nil
}

func (s *Server) Start(port string) <-chan os.Signal {

	log.Println("Server Start ...")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		lis, err := net.Listen("tcp", ":"+port)
		if err != nil {
			log.Fatalln(err)
		}
		//s.client = &http.Client{Timeout: time.Duration(time.Second) * 3}
		s.client = &http.Client{Timeout: time.Minute * 5}

		var opts []grpc.ServerOption
		if os.Getenv("INSECURE") == "TRUE" {
			opts = []grpc.ServerOption{}
		} else {
			certCA := os.Getenv("CA_CERT")
			certPem := os.Getenv("SVR_CERT")
			keyPem := os.Getenv("SVR_KEY")

			if opts, err = serverOpts(certCA, certPem, keyPem); err != nil {
				log.Fatalln("cert stuff=", err)
			}

		}

		// use cert
		srv := grpc.NewServer(opts...)

		hodlapi.RegisterBreweryServiceServer(srv, s)

		log.Fatalln(srv.Serve(lis))

	}()

	return shutdown
}

// queryImp - perform http request
func (s *Server) queryImp(request string) (*http.Response, error) {

	req, err := http.NewRequest("GET", request, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := s.client.Do(req)

	return resp, err

}

func newBreweryResult(brewery *hodlapi.Brewery) types.BreweryResult {
	return types.BreweryResult{
		ID:          brewery.Id,
		Name:        brewery.Name,
		BreweryType: brewery.BreweryType,

		Street: brewery.Street,
		//	address_2: null,
		//	address_3: null,
		City:  brewery.City,
		State: brewery.State,

		CountryProvince: brewery.CountryProvince,
		PostalCode:      brewery.PostalCode,
		Country:         brewery.Country,
		Longitude:       brewery.Longitude,
		Latitude:        brewery.Latitude,
		Phone:           brewery.Phone,
		Website:         brewery.WebsiteUrl,
		Updated:         brewery.UpdatedAt,
		Created:         brewery.CreatedAt,
		//	updated_at: "2018-08-23T23:24:11.758Z",
		//	created_at: "2018-08-23T23:24:11.758Z"
	}
}

// newBrewery - convert from internal JSON type to protobuf type
func newBrewery(brewery types.BreweryResult) *hodlapi.Brewery {

	return &hodlapi.Brewery{Id: brewery.ID,
		Name:        brewery.Name,
		BreweryType: brewery.BreweryType,
		Street:      brewery.Street,
		City:        brewery.City,
		State:       brewery.State,

		CountryProvince: brewery.CountryProvince,
		PostalCode:      brewery.PostalCode,
		Country:         brewery.Country,
		Longitude:       brewery.Longitude,
		Latitude:        brewery.Latitude,
		Phone:           brewery.Phone,
		WebsiteUrl:      brewery.Website,
		UpdatedAt:       brewery.Updated,
		CreatedAt:       brewery.Created,
	}
}

// startQuery - make subsequent paginanted API calls until none remain
//
//	writing to channel each record of each page
func (s *Server) startQuery(qr types.QueryResult, pipe chan types.BreweryResult) {

	request := qr.Query
	page := 0
	workingPage := request

	// if qr.Filter.Page == 0  loop for all
	//   else break after first
	for {

		if qr.Filter.Page == 0 {
			page += 1
			workingPage = fmt.Sprintf("%s&page=%d", request, page)
		}

		resp, err := s.queryImp(workingPage)

		if err != nil {
			fmt.Printf("working=%s\n", workingPage)
			fmt.Println(err)
			close(pipe)
			return
		}

		/*
			// forward the headers from upstream call
			for key, value := range resp.Header {
				w.Header().Set(key, value[0])
			}
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)

		*/
		defer resp.Body.Close()

		var b []types.BreweryResult

		if err = json.NewDecoder(resp.Body).Decode(&b); err != nil {
			log.Printf("oops")
		}

		if resp.StatusCode > 299 {
			log.Printf("oops")
		}

		fmt.Printf("count=%v\n", len(b))

		if len(b) < 1 {
			fmt.Printf("storing request=%s\n", request)
			// cache
			// queryResults[qr.Filter] = qr
			if pipe != nil {
				close(pipe) // end here
			}
			return
		}

		// add a page and empty Breweries slice
		// cache
		// qr.Pages = append(qr.Pages, types.Page{ID: page, Breweries: make([]types.BreweryResult, 0)})

		for _, brewery := range b {
			//write to channel
			if pipe != nil {
				fmt.Println(brewery.Name)
				pipe <- brewery
			}
			// cache
			// qr.Pages[page-1].Breweries = append(qr.Pages[page-1].Breweries, brewery)
		}

		if qr.Filter.Page != 0 {
			if pipe != nil {
				close(pipe) // end here
			}
			return
		}
		//		if page == 1 && pipe != nil {
		//			close(pipe)
		//		}
	}

}

// parse -
func parse(reqURI string) (types.Filter, error) {

	var filter types.Filter // 0 value

	// query portion to map[string][]string
	queries, err := url.ParseQuery(reqURI)
	if err != nil {
		return filter, err
	}

	//filter.Query = reqURI
	filter.By_city = queries.Get("by_city")
	filter.By_dist = queries.Get("by_dist")
	filter.By_name = queries.Get("by_name")
	filter.By_state = queries.Get("by_state")
	filter.By_postal = queries.Get("by_postal")
	filter.By_type = queries.Get("by_type")

	filter.Location = queries.Get("location")

	perpage := queries.Get("per_page")
	if len(perpage) > 0 {
		filter.Per_page, err = strconv.Atoi(perpage)
	}

	page := queries.Get("page")
	if len(page) > 0 {
		filter.Page, err = strconv.Atoi(page)
	}

	return filter, nil
}

func page(reqURI string) int {

	//	parsed, err := url.ParseRequestURI(reqURI)
	//	if err != nil {
	//		return 0
	//	}

	// query portion to map[string][]string
	queries, err := url.ParseQuery(reqURI)
	if err != nil {
		return 0
	}

	page := queries.Get("page")

	rc, _ := strconv.Atoi(page)
	return rc
}

func isuuid(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// change holdapi.Filter to hold only query, so

/// headers https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#sending-and-receiving-metadata---server-side

// ListBreweries
func (s *Server) ListBreweries(pbFilter *hodlapi.Filter, stream hodlapi.BreweryService_ListBreweriesServer) error {

	var filter types.Filter
	//	if isuuid(pbFilter.Query)

	filter, err := parse(pbFilter.Query)
	if err != nil {
		return err
	}

	// how to return custom headers in a gRPC request
	// future intent to allow request from client with header returned form earlier request
	id := uuid.New()
	header := metadata.Pairs("location", id.String())
	stream.SetHeader(header)

	// notes: Want the future ability to stream one page back to client , while continuing to cache the full result set

	breweryChannel := make(chan types.BreweryResult)
	//	defer close(breweryChannel)
	// Create new Caldera operation , run in a seperate goroutine

	// future : cache results and check for queryresult in local short-term cache
	// qr := cache["query string"]
	var qr types.QueryResult

	qr.Query = fmt.Sprintf("%s/breweries?%s", s.APIUrl, pbFilter.Query)
	qr.Filter = filter
	//go s.startQuery(qr, breweryChannel)

	go s.startQuery(qr, breweryChannel)

	for brewery := range breweryChannel {
		if err := stream.Send(newBrewery(brewery)); err != nil {
			return err
		}
	}

	return nil
}

// BreweryCount - loop over all return count of all elements in all pages
func (s *Server) BreweryCount(ctx context.Context, pbFilter *hodlapi.Filter) (*hodlapi.BreweryCountRespose, error) {

	resp := &hodlapi.BreweryCountRespose{
		Count: 0,
	}
	var filter types.Filter

	//	if isuuid(pbFilter.Query)

	filter, err := parse(pbFilter.Query)
	if err != nil {
		return resp, err
	}
	qr := types.QueryResult{Query: fmt.Sprintf("%s/breweries?%s", s.APIUrl, pbFilter.Query), Filter: filter}

	breweryChannel := make(chan types.BreweryResult)

	go s.startQuery(qr, breweryChannel)

	var count int32 = 0

	for brewery := range breweryChannel {
		count = count + 1
		fmt.Println(brewery.ID)
	}

	resp.Count = count

	/*
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {

			defer wg.Done()
			s.startQuery(qr, nil)

		}()

		wg.Wait()
	*/

	return resp, nil
}

// SearchBreweries - Future todo
func (s *Server) SearchBreweries(filter *hodlapi.Filter, stream hodlapi.BreweryService_SearchBreweriesServer) error {
	return nil
}
