package types

type QueryResult struct {
	Query  string
	Filter Filter
	Pages  []Page
}

type Page struct {
	ID        int
	Breweries []BreweryResult
}

type Filter struct {
	By_city   string
	By_dist   string
	By_name   string
	By_state  string
	By_postal string
	By_type   string
	Page      int
	Per_page  int
	Sort      string
	Count     int
	Location  string
}

type BreweryResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	BreweryType string `json:"brewery_type"`

	Street string `json:"street"`
	//	address_2: null,
	//	address_3: null,
	City  string `json:"city"`
	State string `json:"state"`

	CountryProvince string `json:"CountryProvince"`
	PostalCode      string `json:"postal_code"`
	Country         string `json:"country"`
	Longitude       string `json:"longitude"`
	Latitude        string `json:"latitude"`
	Phone           string `json:"phone"`

	Website string `json:"website_url"`
	Updated string `json:"updated_at"`
	Created string `json:"created_at"`
	//	updated_at: "2018-08-23T23:24:11.758Z",
	//	created_at: "2018-08-23T23:24:11.758Z"
	//	MapURL string `json:"mapurl"`
}
