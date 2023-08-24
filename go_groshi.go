// Package go_groshi is a client library for groshi API (https://github.com/groshi-project/groshi).
// Any groshi API method can be easily called using this package.
package go_groshi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RFC-3339 is the conventional time format for groshi API
const timeFormat = time.RFC3339

var errResponseIsNotMap = errors.New(
	"impossible to represent the API response as map",
)
var errResponseIsNotSliceOfMaps = errors.New(
	"impossible to represent the API response as slice of maps",
)

type GroshiAPIResponse struct {
	Response any
}

// SliceOfMaps returns API response as []map[string]any.
// Returns errResponseIsNotSliceOfMaps error if it is not possible.
func (r *GroshiAPIResponse) SliceOfMaps() ([]map[string]any, error) {
	var objects []map[string]any

	// I am not sure why .([]map[string]any) does not succeed here, but it does not.
	// that's why an extra step with `objects` slice is required...
	rawObjects, ok := r.Response.([]any)
	if !ok {
		return nil, errResponseIsNotSliceOfMaps
	}

	for _, rawObject := range rawObjects {
		object, ok := rawObject.(map[string]any)
		if !ok {
			return nil, errResponseIsNotSliceOfMaps
		}
		objects = append(objects, object)
	}
	return objects, nil
}

// Map returns API response as map[string]any.
// Returns errResponseIsNotMap error if it is not possible.
func (r *GroshiAPIResponse) Map() (map[string]any, error) {
	result, ok := r.Response.(map[string]any)
	if !ok {
		return nil, errResponseIsNotMap
	}
	return result, nil
}

type GroshiAPIError struct {
	Description  string
	ErrorDetails []string
}

func (g *GroshiAPIError) Error() string {
	if len(g.ErrorDetails) == 0 {
		return g.Description
	} else {
		return fmt.Sprintf("%v (%v)", g.Description, strings.Join(g.ErrorDetails, ", "))
	}
}

type GroshiAPIClient struct {
	baseURL string
	jwt     string
}

func NewGroshiAPIClient(baseURL string, jwt string) *GroshiAPIClient {
	return &GroshiAPIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		jwt:     jwt,
	}
}

// sendRequest is the basic method for sending HTTP requests to groshi API.
// Returns GroshiAPIError if API error occurred.
func (g *GroshiAPIClient) sendRequest(
	method string, path string, queryParams map[string]string, bodyParams map[string]any, authorize bool) (*GroshiAPIResponse, error) {
	if authorize && g.jwt == "" {
		panic("`authorize` is true, but `g.jwt` is an empty string")
	}

	// create URL object and set query params:
	urlObject, err := url.Parse(g.baseURL + path)
	if err != nil {
		return nil, err
	}

	queryParamsObject := urlObject.Query()
	for key, value := range queryParams {
		queryParamsObject.Add(key, value)
	}
	urlObject.RawQuery = queryParamsObject.Encode()

	// encode request body:
	body, err := json.Marshal(bodyParams)
	if err != nil {
		return nil, err
	}

	// create request object:
	request, err := http.NewRequest(method, urlObject.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// set request headers:
	request.Header.Set("Content-Type", "application/json")
	if authorize {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", g.jwt))
	}

	// create HTTP client:
	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	// finally send the HTTP request:
	httpResponse, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	// read the response body:
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	// parse response:
	var response any // response is of type `any` because API can return either JSON object or JSON array of JSON objects
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}

	rawResponseObject := &GroshiAPIResponse{response}

	if httpResponse.StatusCode == http.StatusOK {
		return rawResponseObject, nil // return the response object if status is OK
	} else {
		// otherwise return GroshiAPIError object describing the error occurred
		responseUnpacked, err := rawResponseObject.Map() // response describing error is always map (JSON object)
		if err != nil {
			return nil, err
		}

		errorDescription := responseUnpacked["error_description"].(string)

		errorDetails := make([]string, 0)
		for _, detail := range responseUnpacked["error_details"].([]interface{}) {
			errorDetails = append(errorDetails, detail.(string))
		}

		return nil, &GroshiAPIError{
			Description:  errorDescription,
			ErrorDetails: errorDetails,
		}
	}
}

// methods related to authorization:

func (g *GroshiAPIClient) AuthLogin(username string, password string) (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodPost,
		"/auth/login",
		nil,
		map[string]any{
			"username": username,
			"password": password,
		},
		false,
	)
}

func (g *GroshiAPIClient) AuthLogout() (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodPost,
		"/auth/logout",
		nil,
		nil,
		true,
	)
}

func (g *GroshiAPIClient) AuthRefresh() (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodPost,
		"/auth/refresh",
		nil,
		nil,
		true,
	)
}

// methods related to API users:

func (g *GroshiAPIClient) UserCreate(username string, password string) (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodPost,
		"/user",
		nil,
		map[string]any{
			"username": username,
			"password": password,
		},
		false,
	)
}

func (g *GroshiAPIClient) UserRead() (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodGet,
		"/user",
		nil,
		nil,
		true,
	)
}

func (g *GroshiAPIClient) UserUpdate(newUsername *string, newPassword *string) (*GroshiAPIResponse, error) {
	bodyParams := make(map[string]any)
	if newUsername != nil {
		bodyParams["new_username"] = *newUsername
	}

	if newPassword != nil {
		bodyParams["new_password"] = *newPassword
	}

	return g.sendRequest(
		http.MethodPut,
		"/user",
		nil,
		bodyParams,
		true,
	)
}

func (g *GroshiAPIClient) UserDelete() (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodDelete,
		"/user",
		nil,
		nil,
		true,
	)
}

// methods related to transactions:

func (g *GroshiAPIClient) TransactionsCreate(amount int, currency string, description *string, date *time.Time) (*GroshiAPIResponse, error) {
	bodyParams := map[string]any{
		"amount":   amount,
		"currency": currency,
	}
	if description != nil {
		bodyParams["description"] = *description
	}
	if date != nil {
		bodyParams["date"] = *date
	}

	return g.sendRequest(
		http.MethodPost,
		"/transactions",
		nil,
		bodyParams,
		true,
	)
}

func (g *GroshiAPIClient) TransactionsReadMany(startTime time.Time, endTime *time.Time) (*GroshiAPIResponse, error) {
	queryParams := map[string]string{
		"start_time": startTime.Format(timeFormat),
	}

	if endTime != nil {
		queryParams["end_time"] = (*endTime).Format(timeFormat)
	}

	return g.sendRequest(
		http.MethodGet,
		"/transactions",
		queryParams,
		nil,
		true,
	)
}

func (g *GroshiAPIClient) TransactionsReadOne(uuid string) (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodGet,
		fmt.Sprintf("/transactions/%v", uuid),
		nil,
		nil,
		true,
	)
}

func (g *GroshiAPIClient) TransactionsUpdate(uuid string, newAmount *int, newCurrency *string, newDescription *string, newDate *time.Time) (*GroshiAPIResponse, error) {
	bodyParams := make(map[string]any)
	if newAmount != nil {
		bodyParams["new_amount"] = *newAmount
	}

	if newCurrency != nil {
		bodyParams["new_currency"] = newCurrency
	}

	if newDescription != nil {
		bodyParams["new_description"] = *newDescription
	}

	if newDate != nil {
		bodyParams["new_date"] = (*newDate).Format(timeFormat)
	}

	return g.sendRequest(
		http.MethodPut,
		fmt.Sprintf("/transactions/%v", uuid),
		nil,
		nil,
		true,
	)
}

func (g *GroshiAPIClient) TransactionsDelete(uuid string) (*GroshiAPIResponse, error) {
	return g.sendRequest(
		http.MethodDelete,
		fmt.Sprintf("/transactions/%v", uuid),
		nil,
		nil,
		true,
	)
}

func (g *GroshiAPIClient) TransactionsReadSummary(startTime time.Time, currency string, endTime *time.Time) (*GroshiAPIResponse, error) {
	queryParams := map[string]string{
		"start_time": startTime.Format(timeFormat),
		"currency":   currency,
	}

	if endTime != nil {
		queryParams["end_time"] = (*endTime).Format(timeFormat)
	}

	return g.sendRequest(
		http.MethodGet,
		fmt.Sprintf("/transactions/summary"),
		queryParams,
		nil,
		true,
	)
}
