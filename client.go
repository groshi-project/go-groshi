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

// GroshiAPIClient TODO
type GroshiAPIClient struct {
	baseURL string
	Token   string
}

// sendRequest is the basic method for sending HTTP requests to groshi API.
func (c *GroshiAPIClient) sendRequest(
	method string, path string, queryParams map[string]string, bodyParams map[string]any, authorize bool, v interface{},
) error {
	if authorize && c.Token == "" { // todo: ??
		panic("`authorize` is set to true, but no authorization token was provided")
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
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.Token))
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
		groshiAPIError := GroshiAPIError{}
		if err := json.Unmarshal(responseBody, &groshiAPIError); err != nil {
			return err
		}
		return &groshiAPIError
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
	c.Token = token
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
	return &authorization, err
}

func (c *GroshiAPIClient) AuthLogout() (*Empty, error) {
	empty := Empty{}
	err := c.sendRequest(
		http.MethodPost,
		"/auth/logout",
		nil,
		nil,
		true,
		&empty,
	)
	return &empty, err
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
	return &authorization, err
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
	return &user, err
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
	return &user, err
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
	return &user, err
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
	return &user, err
}

// methods related to transactions:

func (c *GroshiAPIClient) TransactionsCreate(amount int, currency string, description *string, date *time.Time) (*Transaction, error) {
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

	transaction := Transaction{}
	err := c.sendRequest(
		http.MethodPost,
		"/transactions",
		nil,
		bodyParams,
		true,
		&transaction,
	)
	return &transaction, err
}

func (c *GroshiAPIClient) TransactionsReadOne(uuid string) (*Transaction, error) {
	transaction := Transaction{}
	err := c.sendRequest(
		http.MethodGet,
		fmt.Sprintf("/transactions/%v", uuid),
		nil,
		nil,
		true,
		&transaction,
	)
	return &transaction, err
}

func (c *GroshiAPIClient) TransactionsReadMany(startTime time.Time, endTime *time.Time) ([]*Transaction, error) {
	queryParams := map[string]string{
		"start_time": startTime.Format(timeFormat),
	}
	if endTime != nil {
		queryParams["end_time"] = (*endTime).Format(timeFormat)
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
	return transactions, err
}

func (c *GroshiAPIClient) TransactionsUpdate(
	uuid string, newAmount *int, newCurrency *string, newDescription *string, newDate *time.Time,
) (*Transaction, error) {
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

	transaction := Transaction{}
	err := c.sendRequest(
		http.MethodPut,
		fmt.Sprintf("/transactions/%v", uuid),
		nil,
		nil,
		true,
		&transaction,
	)
	return &transaction, err
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
	return &transaction, err
}

func (c *GroshiAPIClient) TransactionsReadSummary(startTime time.Time, currency string, endTime *time.Time) (*TransactionsSummary, error) {
	queryParams := map[string]string{
		"start_time": startTime.Format(timeFormat),
		"currency":   currency,
	}
	if endTime != nil {
		queryParams["end_time"] = (*endTime).Format(timeFormat)
	}

	transactionsSummary := TransactionsSummary{}
	err := c.sendRequest(
		http.MethodGet,
		fmt.Sprintf("/transactions/summary"),
		queryParams,
		nil,
		true,
		&transactionsSummary,
	)
	return &transactionsSummary, err
}

// NewGroshiAPIClient creates a new GroshiAPIClient instance and returns pointer to it.
// It is the recommended method to produce GroshiAPIClient.
func NewGroshiAPIClient(baseURL string, token string) *GroshiAPIClient {
	return &GroshiAPIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
	}
}