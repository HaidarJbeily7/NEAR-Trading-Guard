package intents

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func validRequest() Request {
	return Request{SwapType: "EXACT_INPUT", SlippageTolerance: 50, OriginAsset: "nep141:wrap.near", DepositType: "ORIGIN_CHAIN", DestinationAsset: "nep141:fixture.near", Amount: "1000000000000000000000000", Recipient: "recipient.near", RecipientType: "DESTINATION_CHAIN", RefundTo: "refund.near", RefundType: "ORIGIN_CHAIN", Deadline: time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339)}
}

const tokensJSON = `[{"assetId":"nep141:wrap.near","blockchain":"near","decimals":24},{"assetId":"nep141:fixture.near","blockchain":"near","decimals":6}]`

func TestQuoteModes(t *testing.T) {
	for _, prepare := range []bool{false, true} {
		calls := 0
		client := Client{JWT: "test-token", HTTP: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Host != "1click.chaindefuser.com" || r.Header.Get("Authorization") != "Bearer test-token" {
				t.Fatal("wrong host or auth")
			}
			if r.URL.Path == "/v0/tokens" {
				return response(200, tokensJSON), nil
			}
			if r.Method != "POST" || r.URL.Path != "/v0/quote" {
				t.Fatal("wrong quote endpoint")
			}
			var request Request
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Dry == prepare {
				t.Fatalf("unexpected dry=%v prepare=%v", request.Dry, prepare)
			}
			if request.Amount != validRequest().Amount {
				t.Fatal("integer amount modified")
			}
			body, _ := json.Marshal(map[string]any{"signature": "fixture-signature", "quoteRequest": request, "quote": map[string]string{"depositAddress": "deposit.near", "amountIn": request.Amount, "amountOut": "1000", "minAmountOut": "995", "deadline": request.Deadline}})
			return response(200, string(body)), nil
		})}}
		request := validRequest()
		request.Dry = prepare // deliberately opposite to requested mode
		var err error
		if prepare {
			_, err = client.Prepare(context.Background(), request, request.Amount)
		} else {
			_, err = client.Quote(context.Background(), request)
		}
		if err != nil || calls != 2 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}

func TestRejectUnsafeRequestsBeforeNetwork(t *testing.T) {
	client := Client{HTTP: &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network call"); return nil, nil })}}
	cases := []struct {
		name   string
		change func(*Request)
	}{
		{"fractional amount", func(r *Request) { r.Amount = "1.5" }},
		{"zero amount", func(r *Request) { r.Amount = "0" }},
		{"expired", func(r *Request) { r.Deadline = time.Now().Add(-time.Minute).Format(time.RFC3339) }},
		{"unbounded slippage", func(r *Request) { r.SlippageTolerance = 101 }},
		{"unsupported funding", func(r *Request) { r.DepositType = "INTENTS" }},
		{"same asset", func(r *Request) { r.DestinationAsset = r.OriginAsset }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validRequest()
			tc.change(&r)
			if _, err := client.Quote(context.Background(), r); err == nil {
				t.Fatal("accepted invalid request")
			}
		})
	}
	if _, err := client.Prepare(context.Background(), validRequest(), "1"); err == nil {
		t.Fatal("accepted oversized swap")
	}
}

func TestRouteValidation(t *testing.T) {
	var tokens []Token
	_ = json.Unmarshal([]byte(tokensJSON), &tokens)
	r := validRequest()
	if err := r.ValidateRoute(tokens); err != nil {
		t.Fatal(err)
	}
	r.Recipient = "wrong..near"
	if r.ValidateRoute(tokens) == nil {
		t.Fatal("accepted bad account")
	}
	r = validRequest()
	tokens[1].Blockchain = "eth"
	if r.ValidateRoute(tokens) == nil {
		t.Fatal("accepted unsupported chain")
	}
	if r.ValidateRoute(nil) == nil {
		t.Fatal("accepted missing token")
	}
}

func TestStatusPreservesMemo(t *testing.T) {
	client := Client{HTTP: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("depositMemo") != "a+b & c" || r.URL.Query().Get("depositAddress") != "deposit.near" {
			t.Fatal("query corrupted")
		}
		return response(200, `{"status":"REFUNDED"}`), nil
	})}}
	data, err := client.Status(context.Background(), "deposit.near", "a+b & c")
	if err != nil || !strings.Contains(string(data), "REFUNDED") {
		t.Fatalf("%s %v", data, err)
	}
}

func TestFailuresAreNotRetried(t *testing.T) {
	for _, tc := range []struct {
		code int
		body string
	}{{429, `{"error":"limited"}`}, {500, `secret upstream error`}, {200, `not JSON`}, {200, strings.Repeat("x", (4<<20)+1)}} {
		calls := 0
		client := Client{HTTP: &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { calls++; return response(tc.code, tc.body), nil })}}
		if _, err := client.Status(context.Background(), "deposit.near", ""); err == nil {
			t.Fatal("accepted failed response")
		}
		if calls != 1 {
			t.Fatal("retried request")
		}
	}
}

func TestRejectMismatchedQuote(t *testing.T) {
	request := validRequest()
	base := func() map[string]any {
		return map[string]any{
			"signature": "fixture-signature", "quoteRequest": request,
			"quote": map[string]string{"depositAddress": "deposit.near", "amountIn": request.Amount, "amountOut": "1000", "minAmountOut": "995", "deadline": request.Deadline},
		}
	}
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"recipient", func(v map[string]any) { r := request; r.Recipient = "other.near"; v["quoteRequest"] = r }},
		{"amount", func(v map[string]any) { v["quote"].(map[string]string)["amountIn"] = "1" }},
		{"slippage", func(v map[string]any) { v["quote"].(map[string]string)["minAmountOut"] = "990" }},
		{"expired", func(v map[string]any) {
			v["quote"].(map[string]string)["deadline"] = time.Now().Add(-time.Minute).Format(time.RFC3339)
		}},
		{"missing signature", func(v map[string]any) { delete(v, "signature") }},
		{"invalid deposit", func(v map[string]any) { v["quote"].(map[string]string)["depositAddress"] = "https://wrong.example" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := base()
			tc.change(v)
			data, _ := json.Marshal(v)
			if validateQuote(data, request, time.Now()) == nil {
				t.Fatal("unsafe quote accepted")
			}
		})
	}
}
