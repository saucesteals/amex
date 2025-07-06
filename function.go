package amex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"maps"

	"github.com/google/uuid"
	http "github.com/saucesteals/fhttp"
)

type FunctionError struct {
	Type    string `json:"-"`
	Code    string `json:"ErrorCode"`
	Message string `json:"ErrorMessage"`

	// For step up
	RiskDecision    string `json:"RiskDecision"`
	JourneyId       string `json:"JourneyId"`
	AssessmentToken string `json:"AssessmentToken"`
}

func (e *FunctionError) Error() string {
	return fmt.Sprintf("function error: %s", e.Code)
}

type FunctionResponse struct {
	error *FunctionError
	data  json.RawMessage
}

func (r FunctionResponse) Bind(v any) error {
	if r.error != nil {
		return fmt.Errorf("function error: %s", r.error.Code)
	}

	return json.Unmarshal(r.data, v)
}

func (r FunctionResponse) Error() *FunctionError {
	return r.error
}

func (a *API) callFunction(ctx context.Context, function string, args any, headers ...http.Header) (FunctionResponse, error) {
	var bodyReader io.Reader
	if args != nil {
		payload, err := json.Marshal(args)
		if err != nil {
			return FunctionResponse{}, err
		}

		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://functions.americanexpress.com/"+function, bodyReader)
	if err != nil {
		return FunctionResponse{}, err
	}

	if bodyReader != nil {
		req.Header.Set("content-type", "application/json")
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip, deflate, br, zstd")
	req.Header.Set("accept-language", "en-US,en;q=0.7")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("ce-source", "blueprint")
	req.Header.Set("one-data-correlation-id", uuid.NewString())
	req.Header.Set("origin", "https://www.americanexpress.com")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("sec-gpc", "1")

	for _, header := range headers {
		maps.Copy(req.Header, header)
	}

	res, err := a.client.Do(req)
	if err != nil {
		return FunctionResponse{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return FunctionResponse{}, err
	}

	if bytes.HasPrefix(body, []byte(`{"error":`)) {
		var errorResponse struct {
			Error string `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResponse); err != nil {
			return FunctionResponse{}, err
		}

		parts := strings.SplitN(errorResponse.Error, " ", 2)
		if len(parts) != 2 {
			return FunctionResponse{}, fmt.Errorf("invalid error response: %s", errorResponse.Error)
		}

		errorTypeParts := strings.SplitN(parts[0], ",", 2)
		if len(errorTypeParts) != 2 {
			return FunctionResponse{}, fmt.Errorf("invalid error response: %s", errorResponse.Error)
		}

		errorType := strings.TrimPrefix(errorTypeParts[0], "(")

		var functionError FunctionError
		if err := json.Unmarshal([]byte(parts[1]), &functionError); err != nil {
			return FunctionResponse{}, err
		}

		functionError.Type = errorType

		return FunctionResponse{
			data:  body,
			error: &functionError,
		}, nil
	}

	return FunctionResponse{
		data:  body,
		error: nil,
	}, nil
}
