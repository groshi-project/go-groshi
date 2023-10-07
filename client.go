package go_groshi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const timeFormat = time.RFC3339 // RFC-3339 is the time format which is used by groshi API

// GroshiAPIError represents groshi API error.
type GroshiAPIError struct {
	HTTPStatusCode int

	ErrorMessage string
	ErrorDetails []string
}

func (e GroshiAPIError) Error() string {
	if len(e.ErrorDetails) == 0 {
		return e.ErrorMessage
	} else {
		return fmt.Sprintf("%v (%v)", e.ErrorMessage, strings.Join(e.ErrorDetails, ", "))
	}
}

// GroshiAPIClient represents groshi API client and includes all groshi API methods.
type GroshiAPIClient struct {
	baseURL string
	token   string
}

// sendRequest is the basic method for sending HTTP requests to groshi API.
func (c *GroshiAPIClient) sendRequest(
	method string, path string, queryParams map[string]string, bodyParams map[string]any, authorize bool, v interface{},
) error {
	if authorize && c.token == "" {
		panic("`authorize` is set to true, but GroshiAPIClient's field `token` is an empty string")
	}

	// create URL object and set query params:
	urlObject, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}

	queryParamsObject := urlObject.Query()
	for key, value := range queryParams {
		queryParamsObject.Add(key, value)
	}
	urlObject.RawQuery = queryParamsObject.Encode()

	// encode request body:
	body, err := json.Marshal(bodyParams)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(method, urlObject.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	if authorize {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.token))
	}

	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	httpResponse, err := httpClient.Do(request)
	if err != nil {
		return err
	}

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}

	if httpResponse.StatusCode == http.StatusOK {
		if err := json.Unmarshal(responseBody, &v); err != nil {
			return err
		}
		return nil
	} else {
		errorModel := Error{}
		if err := json.Unmarshal(responseBody, &errorModel); err != nil {
			return err
		}
		return GroshiAPIError{
			ErrorMessage: errorModel.ErrorMessage,
			ErrorDetails: errorModel.ErrorDetails,

			HTTPStatusCode: httpResponse.StatusCode,
		}
	}
}

// SetToken is a setter method for authorization token.
// May be useful if you, for example, use GroshiAPIClient
// to create a new user and then perform some operations
// that require authorization. For example:
//
// client := NewGroshiAPIClient("http://localhost:8080", "") // create groshi client with empty token
// _, _ = client.UserCreate("username-1234", "password-1234")
// auth, _ := client.AuthLogin("username-1234", "password-1234")
// client.SetToken(auth.Token)
// currentUser, _ := client.UserRead()
// fmt.Printf("Authorized as %v", currentUser.Username)
func (c *GroshiAPIClient) SetToken(token string) {
	c.token = token
}

// Auth is a helper function that uses AuthLogin groshi API method to authorize user.
// It also sets Token field of the `c` to the received token. Example:
//
// client := NewGroshiAPIClient("http://localhost:8080", "")
// err := client.Auth("username-1234", "password-1234")
// currentUser, _ := client.UserRead()
// fmt.Printf("Authorized as %v", currentUser.Username)
func (c *GroshiAPIClient) Auth(username string, password string) error {
	authorization, err := c.AuthLogin(username, password)
	if err != nil {
		return err
	}
	c.SetToken(authorization.Token)
	return nil
}

// methods related to authorization:

func (c *GroshiAPIClient) AuthLogin(username string, password string) (*Authorization, error) {
	authorization := Authorization{}
	err := c.sendRequest(
		http.MethodPost,
		"/auth/login",
		nil,
		map[string]any{
			"username": username,
			"password": password,
		},
		false,
		&authorization,
	)
	if err != nil {
		return nil, err
	}
	return &authorization, nil
}

func (c *GroshiAPIClient) AuthRefresh() (*Authorization, error) {
	authorization := Authorization{}
	err := c.sendRequest(
		http.MethodPost,
		"/auth/refresh",
		nil,
		nil,
		true,
		&authorization,
	)
	if err != nil {
		return nil, err
	}
	return &authorization, nil
}

// methods related to user:

func (c *GroshiAPIClient) UserCreate(username string, password string) (*User, error) {
	user := User{}
	err := c.sendRequest(
		http.MethodPost,
		"/user",
		nil,
		map[string]any{
			"username": username,
			"password": password,
		},
		false,
		&user,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *GroshiAPIClient) UserRead() (*User, error) {
	user := User{}
	err := c.sendRequest(
		http.MethodGet,
		"/user",
		nil,
		nil,
		true,
		&user,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *GroshiAPIClient) UserUpdate(newUsername *string, newPassword *string) (*User, error) {
	bodyParams := make(map[string]any)
	if newUsername != nil {
		bodyParams["new_username"] = *newUsername
	}
	if newPassword != nil {
		bodyParams["new_password"] = *newPassword
	}

	user := User{}
	err := c.sendRequest(
		http.MethodPut,
		"/user",
		nil,
		bodyParams,
		true,
		&user,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *GroshiAPIClient) UserDelete() (*User, error) {
	user := User{}
	err := c.sendRequest(
		http.MethodDelete,
		"/user",
		nil,
		nil,
		true,
		&user,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// methods related to transactions:

func (c *GroshiAPIClient) TransactionsCreate(amount int, currency string, description *string, timestamp *time.Time) (*Transaction, error) {
	bodyParams := map[string]any{
		"amount":   amount,
		"currency": currency,
	}
	if description != nil {
		bodyParams["description"] = *description
	}
	if timestamp != nil {
		bodyParams["timestamp"] = *timestamp
	}

	transaction := Transaction{}
	err := c.sendRequest(
		http.MethodPost,
		"/transactions",
		nil,
		bodyParams,
		true,
		&transaction,
	)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (c *GroshiAPIClient) TransactionsReadOne(uuid string, currency *string) (*Transaction, error) {
	var queryParams map[string]string
	if currency != nil {
		queryParams = make(map[string]string) // initialize the map only if it is needed
		queryParams["currency"] = *currency
	}

	transaction := Transaction{}
	err := c.sendRequest(
		http.MethodGet,
		fmt.Sprintf("/transactions/%v", uuid),
		queryParams,
		nil,
		true,
		&transaction,
	)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (c *GroshiAPIClient) TransactionsReadMany(startTime time.Time, endTime *time.Time, currency *string) ([]*Transaction, error) {
	queryParams := map[string]string{
		"start_time": startTime.Format(timeFormat),
	}
	if endTime != nil {
		queryParams["end_time"] = (*endTime).Format(timeFormat)
	}
	if currency != nil {
		queryParams["currency"] = *currency
	}

	transactions := make([]*Transaction, 0)
	err := c.sendRequest(
		http.MethodGet,
		"/transactions",
		queryParams,
		nil,
		true,
		&transactions,
	)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (c *GroshiAPIClient) TransactionsUpdate(
	uuid string, newAmount *int, newCurrency *string, newDescription *string, newTimestamp *time.Time,
) (*Transaction, error) {
	bodyParams := make(map[string]any)
	if newAmount != nil {
		bodyParams["new_amount"] = *newAmount
	}
	if newCurrency != nil {
		bodyParams["new_currency"] = *newCurrency
	}
	if newDescription != nil {
		bodyParams["new_description"] = *newDescription
	}
	if newTimestamp != nil {
		bodyParams["new_timestamp"] = (*newTimestamp).Format(timeFormat)
	}

	transaction := Transaction{}
	err := c.sendRequest(
		http.MethodPut,
		fmt.Sprintf("/transactions/%v", uuid),
		nil,
		bodyParams,
		true,
		&transaction,
	)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (c *GroshiAPIClient) TransactionsDelete(uuid string) (*Transaction, error) {
	transaction := Transaction{}
	err := c.sendRequest(
		http.MethodDelete,
		fmt.Sprintf("/transactions/%v", uuid),
		nil,
		nil,
		true,
		&transaction,
	)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (c *GroshiAPIClient) TransactionsReadSummary(currency string, startTime time.Time, endTime *time.Time) (*TransactionsSummary, error) {
	queryParams := map[string]string{
		"currency":   currency,
		"start_time": startTime.Format(timeFormat),
	}
	if endTime != nil {
		queryParams["end_time"] = (*endTime).Format(timeFormat)
	}

	transactionsSummary := TransactionsSummary{}
	err := c.sendRequest(
		http.MethodGet,
		"/transactions/summary",
		queryParams,
		nil,
		true,
		&transactionsSummary,
	)
	if err != nil {
		return nil, err
	}
	return &transactionsSummary, nil
}

// methods related to transactions:

// CurrenciesRead returns slice of available currencies.
func (c *GroshiAPIClient) CurrenciesRead() (*[]Currency, error) {
	var currencies []Currency
	err := c.sendRequest(
		http.MethodGet,
		"/currencies",
		nil,
		nil,
		false,
		&currencies,
	)
	if err != nil {
		return nil, err
	}
	return &currencies, nil
}

// NewGroshiAPIClient creates a new GroshiAPIClient instance and returns pointer to it.
// It is the recommended method to produce GroshiAPIClient.
func NewGroshiAPIClient(baseURL string, token string) *GroshiAPIClient {
	return &GroshiAPIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
	}
}
