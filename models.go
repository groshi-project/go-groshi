package go_groshi

import "time"

// Empty represents empty response.
type Empty struct{}

// Authorization represents successful response containing JWT to the authorization request.
type Authorization struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// User represents response containing information about user.
type User struct {
	Username string `json:"username"`
}

// Transaction represents response containing transaction information.
type Transaction struct {
	UUID string `json:"uuid"`

	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	Date        string `json:"date"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TransactionsSummary represents summary of transactions, returned by transactionsReadSummary handler.
type TransactionsSummary struct {
	Currency string `json:"currency" example:"USD"`

	Income  int `json:"income"`
	Outcome int `json:"outcome"`
	Total   int `json:"total"`

	TransactionsCount int `json:"transactions_count"`
}

// Error represents response containing information about API error.
type Error struct {
	ErrorMessage string   `json:"error_message" example:"example error message"`
	ErrorDetails []string `json:"error_details"`
}
