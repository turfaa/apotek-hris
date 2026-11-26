// Package vmedisproxy calls vmedis-proxy-api, the service in front of Vmedis.
package vmedisproxy

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 2 * time.Minute

// Config configures the Client.
type Config struct {
	// BaseURL is the proxy's API root, e.g. http://backend-server.vmedis/api.
	BaseURL string `mapstructure:"base_url"`
}

// Client calls vmedis-proxy-api. Writes are sent once, never retried: the
// caller retries with the same idempotency key instead.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(config Config) *Client {
	return &Client{
		baseURL:    strings.TrimRight(config.BaseURL, "/"),
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// Error is a response with a non-2xx status.
type Error struct {
	Status  int
	Message string
	// OutcomeUnknown is set when the proxy says Vmedis may have saved the
	// entry anyway (e.g. its save timed out).
	OutcomeUnknown bool
}

func (e *Error) Error() string {
	return fmt.Sprintf("vmedis-proxy responded %d: %s", e.Status, e.Message)
}

// OutcomeKnown reports whether err, from CreateJournalEntry, means for sure
// that nothing was saved: the proxy refused the request, or said Vmedis
// rejected it. Any other failure (a timeout, a lost connection, a 5xx without
// a known outcome) may have saved the entry.
func OutcomeKnown(err error) bool {
	var proxyErr *Error
	if !errors.As(err, &proxyErr) {
		return false
	}

	switch {
	case proxyErr.Status >= 400 && proxyErr.Status < 500:
		return true
	case proxyErr.Status == http.StatusBadGateway:
		return !proxyErr.OutcomeUnknown
	default:
		return false
	}
}

// IsStatus reports whether err is an Error with the status.
func IsStatus(err error, status int) bool {
	var proxyErr *Error
	return errors.As(err, &proxyErr) && proxyErr.Status == status
}

// CreateJournalEntryRequest is a journal entry to create in Vmedis: Amount
// moves from the credit account to the debit account.
type CreateJournalEntryRequest struct {
	// Date is YYYY-MM-DD.
	Date              string `json:"date"`
	Description       string `json:"description"`
	CreditAccountCode string `json:"creditAccountCode"`
	DebitAccountCode  string `json:"debitAccountCode"`
	// Amount is a decimal string of whole Rupiah.
	Amount string `json:"amount"`
}

// CreatedJournalEntry identifies a journal entry in Vmedis.
type CreatedJournalEntry struct {
	VoucherNo string `json:"voucherNo"`
	VmedisID  int64  `json:"vmedisId"`
	Replayed  bool   `json:"replayed"`
}

// CreateJournalEntry creates a journal entry in Vmedis, once per
// idempotencyKey. email is the X-Email the proxy authorises (as admin) and
// records.
func (c *Client) CreateJournalEntry(ctx context.Context, idempotencyKey, email string, req CreateJournalEntryRequest) (CreatedJournalEntry, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CreatedJournalEntry{}, fmt.Errorf("marshal request: %w", err)
	}

	var res CreatedJournalEntry
	err = c.do(ctx, http.MethodPost, "/v1/vmedis/journal-entries", nil, map[string]string{
		"Idempotency-Key": idempotencyKey,
		"X-Email":         email,
		"Content-Type":    "application/json",
	}, body, &res)
	if err != nil {
		return CreatedJournalEntry{}, fmt.Errorf("create journal entry: %w", err)
	}

	return res, nil
}

// JournalEntry is a journal entry listed by Vmedis.
type JournalEntry struct {
	VmedisID    int64     `json:"vmedisId"`
	VoucherNo   string    `json:"voucherNo"`
	Time        time.Time `json:"time"`
	Description string    `json:"description"`
	// Amount is a decimal string.
	Amount   string `json:"amount"`
	Username string `json:"username"`
}

// FindJournalEntries lists the active journal entries dated date (YYYY-MM-DD)
// whose description contains query.
func (c *Client) FindJournalEntries(ctx context.Context, email, date, query string) ([]JournalEntry, error) {
	var res struct {
		Entries []JournalEntry `json:"entries"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/vmedis/journal-entries", url.Values{"date": {date}, "q": {query}}, map[string]string{
		"X-Email": email,
	}, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("find journal entries: %w", err)
	}

	return res.Entries, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, headers map[string]string, body []byte, target any) error {
	if c.baseURL == "" {
		return errors.New("vmedis_proxy.base_url is not configured")
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		var errBody struct {
			Error          string `json:"error"`
			OutcomeUnknown bool   `json:"outcomeUnknown"`
		}
		message := strings.TrimSpace(string(resBody))
		if json.Unmarshal(resBody, &errBody) == nil && errBody.Error != "" {
			message = errBody.Error
		}
		if len(message) > 500 {
			message = message[:500]
		}

		return &Error{Status: res.StatusCode, Message: message, OutcomeUnknown: errBody.OutcomeUnknown}
	}

	if err := json.Unmarshal(resBody, target); err != nil {
		return fmt.Errorf("parse response %.200s: %w", resBody, err)
	}

	return nil
}
