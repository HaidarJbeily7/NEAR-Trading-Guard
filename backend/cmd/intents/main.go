// intents is a local, non-custodial mainnet swap preparation tool.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/intents"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: intents tokens | quote --request FILE [--prepare --confirm-mainnet --max-input UNITS --out FILE] | status --address ADDRESS [--memo MEMO]")
	}
	client := intents.Client{JWT: os.Getenv("INTENTS_JWT")}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	switch os.Args[1] {
	case "tokens":
		tokens, err := client.Tokens(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(tokens)
	case "status":
		fs := flag.NewFlagSet("status", flag.ContinueOnError)
		address := fs.String("address", "", "deposit address from saved quote")
		memo := fs.String("memo", "", "deposit memo, if returned by quote")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		result, err := client.Status(ctx, *address, *memo)
		if err != nil {
			return err
		}
		fmt.Println(string(result))
		return nil
	case "quote":
		fs := flag.NewFlagSet("quote", flag.ContinueOnError)
		input := fs.String("request", "", "JSON request file")
		prepare := fs.Bool("prepare", false, "request a mainnet deposit address; does not send funds")
		confirm := fs.Bool("confirm-mainnet", false, "acknowledge mainnet preparation")
		maximum := fs.String("max-input", "", "maximum input in smallest units of the origin asset")
		output := fs.String("out", "", "new private file for prepared quote; required with --prepare")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *prepare && (!*confirm || *output == "" || *maximum == "") {
			return errors.New("preparation requires --confirm-mainnet, --max-input and --out")
		}
		file, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 16384 {
			return errors.New("request must be a regular file no larger than 16 KiB")
		}
		decoder := json.NewDecoder(io.LimitReader(file, 16385))
		decoder.DisallowUnknownFields()
		var request intents.Request
		if err = decoder.Decode(&request); err != nil {
			return err
		}
		var extra any
		if err = decoder.Decode(&extra); err != io.EOF {
			return errors.New("request must contain exactly one JSON object")
		}
		if request.Deadline == "" {
			request.Deadline = time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)
		}
		var result json.RawMessage
		if *prepare {
			// Reserve a private output file before requesting an address; never overwrite
			// an existing quote. An empty file after failure is not a valid quote.
			saved, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			defer saved.Close()
			result, err = client.Prepare(ctx, request, *maximum)
			if err != nil {
				return err
			}
			request.Dry = false
			record := struct {
				CreatedAt time.Time       `json:"createdAt"`
				Request   intents.Request `json:"request"`
				Response  json.RawMessage `json:"response"`
			}{time.Now().UTC(), request, result}
			if err = json.NewEncoder(saved).Encode(record); err != nil {
				return err
			}
			if err = saved.Sync(); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Mainnet quote saved. No funds sent. Review asset, amount, recipient, refund address, expiry and any memo before using your wallet.")
		} else {
			result, err = client.Quote(ctx, request)
			if err != nil {
				return err
			}
		}
		fmt.Println(string(result))
		return nil
	default:
		return errors.New("unknown command")
	}
}
