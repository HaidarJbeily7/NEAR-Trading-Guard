// Package intents integrates the mainnet 1Click API. It never signs transactions.
package intents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const BaseURL = "https://1click.chaindefuser.com"
const QuoteURL = BaseURL + "/v0/quote"

// Client uses only the official HTTPS API. HTTP may be supplied for tests.
// No retry is performed: callers must explicitly decide whether to request a new quote.
type Client struct {
	HTTP *http.Client
	JWT  string
}

type Token struct {
	AssetID         string  `json:"assetId"`
	Decimals        int     `json:"decimals"`
	Blockchain      string  `json:"blockchain"`
	Symbol          string  `json:"symbol"`
	ContractAddress *string `json:"contractAddress"`
}

// Request intentionally supports only exact-input, origin-chain deposits and
// destination-chain withdrawals. Amounts are strings in smallest token units.
type Request struct {
	Dry               bool   `json:"dry"`
	SwapType          string `json:"swapType"`
	SlippageTolerance int    `json:"slippageTolerance"`
	OriginAsset       string `json:"originAsset"`
	DepositType       string `json:"depositType"`
	DestinationAsset  string `json:"destinationAsset"`
	Amount            string `json:"amount"`
	Recipient         string `json:"recipient"`
	RecipientType     string `json:"recipientType"`
	RefundTo          string `json:"refundTo"`
	RefundType        string `json:"refundType"`
	Deadline          string `json:"deadline"`
}

var integer = regexp.MustCompile(`^[1-9][0-9]{0,77}$`)
var account = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*$`)

func validAccount(s string) bool { return len(s) >= 2 && len(s) <= 64 && account.MatchString(s) }

func (r Request) Validate(now time.Time) error {
	if r.SwapType != "EXACT_INPUT" || r.DepositType != "ORIGIN_CHAIN" || r.RecipientType != "DESTINATION_CHAIN" || r.RefundType != "ORIGIN_CHAIN" {
		return errors.New("only EXACT_INPUT with origin-chain deposit/refund and destination-chain recipient is supported")
	}
	if r.OriginAsset == "" || r.DestinationAsset == "" || r.OriginAsset == r.DestinationAsset {
		return errors.New("choose two distinct supported assets")
	}
	if !integer.MatchString(r.Amount) {
		return errors.New("amount must be a positive integer string in smallest token units (maximum 78 digits)")
	}
	if r.SlippageTolerance < 0 || r.SlippageTolerance > 100 {
		return errors.New("slippage must be between 0 and 100 basis points")
	}
	if r.Recipient == "" || r.RefundTo == "" {
		return errors.New("recipient and refundTo are required")
	}
	deadline, err := time.Parse(time.RFC3339, r.Deadline)
	if err != nil || !deadline.After(now.Add(30*time.Second)) || deadline.After(now.Add(15*time.Minute)) {
		return errors.New("deadline must be between 30 seconds and 15 minutes in the future")
	}
	return nil
}

// ValidateRoute bounds this first release to NEAR-to-NEAR reserve conversion.
// Cross-chain address/memo rules need separate adapters before being enabled.
func (r Request) ValidateRoute(tokens []Token) error {
	var origin, destination *Token
	for i := range tokens {
		if tokens[i].AssetID == r.OriginAsset {
			origin = &tokens[i]
		}
		if tokens[i].AssetID == r.DestinationAsset {
			destination = &tokens[i]
		}
	}
	if origin == nil || destination == nil {
		return errors.New("asset not found in the current 1Click token list")
	}
	if origin.Blockchain != "near" || destination.Blockchain != "near" {
		return errors.New("this release supports NEAR-to-NEAR swaps only")
	}
	if !validAccount(r.Recipient) || !validAccount(r.RefundTo) {
		return errors.New("recipient and refundTo must be valid NEAR account IDs")
	}
	return nil
}

func CheckLimit(amount, maximum string) error {
	if !integer.MatchString(amount) || !integer.MatchString(maximum) {
		return errors.New("amount and maximum must be positive integer strings")
	}
	a, _ := new(big.Int).SetString(amount, 10)
	m, _ := new(big.Int).SetString(maximum, 10)
	if a.Cmp(m) > 0 {
		return errors.New("amount exceeds the explicit maximum input")
	}
	return nil
}

func (c Client) Tokens(ctx context.Context) ([]Token, error) {
	data, err := c.call(ctx, http.MethodGet, "/v0/tokens", nil)
	if err != nil {
		return nil, err
	}
	var tokens []Token
	if err = json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("decode tokens: %w", err)
	}
	return tokens, nil
}

// Quote always forces preview mode, even if a caller supplies dry=false.
func (c Client) Quote(ctx context.Context, request Request) (json.RawMessage, error) {
	request.Dry = true
	return c.quote(ctx, request)
}

// Prepare obtains a deposit address. It does not fund it. The caller must obtain
// explicit user confirmation before transferring any funds with their wallet.
func (c Client) Prepare(ctx context.Context, request Request, maxInput string) (json.RawMessage, error) {
	if err := CheckLimit(request.Amount, maxInput); err != nil {
		return nil, err
	}
	request.Dry = false
	return c.quote(ctx, request)
}

func (c Client) quote(ctx context.Context, request Request) (json.RawMessage, error) {
	if err := request.Validate(time.Now()); err != nil {
		return nil, err
	}
	tokens, err := c.Tokens(ctx)
	if err != nil {
		return nil, err
	}
	if err = request.ValidateRoute(tokens); err != nil {
		return nil, err
	}
	data, err := c.call(ctx, http.MethodPost, "/v0/quote", request)
	if err != nil {
		return nil, err
	}
	if err = validateQuote(data, request, time.Now()); err != nil {
		return nil, err
	}
	return data, nil
}

func (c Client) Status(ctx context.Context, address, memo string) (json.RawMessage, error) {
	if address == "" || len(address) > 256 || len(memo) > 256 {
		return nil, errors.New("valid deposit address and optional memo required")
	}
	values := url.Values{"depositAddress": {address}}
	if memo != "" {
		values.Set("depositMemo", memo)
	}
	return c.call(ctx, http.MethodGet, "/v0/status?"+values.Encode(), nil)
}

func (c Client) call(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.JWT != "" {
		req.Header.Set("Authorization", "Bearer "+c.JWT)
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	// Never follow redirects with credentials or quote payloads.
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := copyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	const limit = 4 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, errors.New("Intents response too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Intents HTTP %d; request was not retried", resp.StatusCode)
	}
	if !json.Valid(data) {
		return nil, errors.New("invalid Intents JSON response")
	}
	return data, nil
}

// Validate a quote before displaying funding instructions. The complete response,
// including its service signature, is retained by the CLI for reconciliation.
func validateQuote(data []byte, request Request, now time.Time) error {
	var response struct {
		Signature    string  `json:"signature"`
		QuoteRequest Request `json:"quoteRequest"`
		Quote        struct {
			DepositAddress string `json:"depositAddress"`
			AmountIn       string `json:"amountIn"`
			AmountOut      string `json:"amountOut"`
			MinAmountOut   string `json:"minAmountOut"`
			Deadline       string `json:"deadline"`
		} `json:"quote"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}
	echoed := response.QuoteRequest
	if echoed.Dry != request.Dry || echoed.Amount != request.Amount || echoed.OriginAsset != request.OriginAsset || echoed.DestinationAsset != request.DestinationAsset || echoed.Recipient != request.Recipient || echoed.RefundTo != request.RefundTo || echoed.SwapType != request.SwapType || echoed.DepositType != request.DepositType || echoed.RecipientType != request.RecipientType || echoed.RefundType != request.RefundType || echoed.SlippageTolerance != request.SlippageTolerance {
		return errors.New("quote does not match requested route, addresses, mode or limits")
	}
	requestedDeadline, requestErr := time.Parse(time.RFC3339, request.Deadline)
	echoedDeadline, echoErr := time.Parse(time.RFC3339, echoed.Deadline)
	if requestErr != nil || echoErr != nil || !echoedDeadline.Equal(requestedDeadline) {
		return errors.New("quote changed the requested refund deadline")
	}
	q := response.Quote
	if q.AmountIn != request.Amount || !integer.MatchString(q.AmountOut) || !integer.MatchString(q.MinAmountOut) {
		return errors.New("quote has invalid or mismatched amounts")
	}
	output, _ := new(big.Int).SetString(q.AmountOut, 10)
	minimum, _ := new(big.Int).SetString(q.MinAmountOut, 10)
	floor := new(big.Int).Mul(output, big.NewInt(int64(10000-request.SlippageTolerance)))
	floor.Div(floor, big.NewInt(10000))
	if minimum.Cmp(output) > 0 || minimum.Cmp(floor) < 0 {
		return errors.New("quote minimum output violates slippage limit")
	}
	if !request.Dry {
		if !validAccount(q.DepositAddress) || response.Signature == "" {
			return errors.New("prepared quote lacks deposit address or service signature")
		}
		deadline, err := time.Parse(time.RFC3339, q.Deadline)
		// The request deadline starts refunds; the quote deadline is the
		// provider's deposit-address lifetime. They need not be identical.
		// Funding must occur before BOTH deadlines.
		if err != nil || !deadline.After(now.Add(30*time.Second)) || !requestedDeadline.After(now.Add(30*time.Second)) {
			return errors.New("prepared quote has unsafe expiry")
		}
	}
	return nil
}
