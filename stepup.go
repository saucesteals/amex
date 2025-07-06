package amex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"unsafe"

	http "github.com/saucesteals/fhttp"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type StepUpRequiredError struct {
	AssessmentToken string
}

func (e StepUpRequiredError) Error() string {
	return "step up required"
}

func (a *API) StepUp(ctx context.Context, accountKey, assessmentToken string) error {
	stepUpUrl := url.URL{
		Scheme: "https",
		Host:   "www.americanexpress.com",
		Path:   "/en-us/account/authenticate",
	}

	query := stepUpUrl.Query()
	query.Set("userJourneyIdentifier", "aexp.commercial:create:virtual_card")
	query.Set("state", assessmentToken)
	query.Set("actionId", "DCND01")
	query.Set("applicationId", "EDC01")
	query.Set("accountKey", accountKey)
	query.Set("errors", "on")
	query.Set("loader", "placeholder")

	stepUpUrl.RawQuery = query.Encode()

	browser, err := a.initBrowser()
	if err != nil {
		return err
	}
	defer browser.Close()

	router := browser.HijackRequests()

	done := make(chan error)
	err = router.Add("*", "", func(h *rod.Hijack) {
		u := h.Request.URL()
		if u.Host != "functions.americanexpress.com" {
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		// This should never happen, but just in case
		if u.Path == "/CreateVirtualCard.v1" {
			h.Response.Fail(proto.NetworkErrorReasonAborted)
			return
		}

		stdRequest := h.Request.Req()

		req, err := http.NewRequest(stdRequest.Method, stdRequest.URL.String(), stdRequest.Body)
		if err != nil {
			h.Response.Fail(proto.NetworkErrorReasonFailed)
			a.log.Error("new hijacked request", "url", stdRequest.URL.String(), "error", err)
			return
		}
		req.Header = http.Header(stdRequest.Header)

		res, err := a.client.Do(req)
		if err != nil {
			h.Response.Fail(proto.NetworkErrorReasonFailed)
			a.log.Error("do hijacked request", "url", stdRequest.URL.String(), "error", err)
			return
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			h.Response.Fail(proto.NetworkErrorReasonFailed)
			a.log.Error("failed to read response body", "url", stdRequest.URL.String(), "error", err)
			return
		}

		if u.Path == "/UpdateAuthenticationTokenWithChallenge.v3" && req.Method == http.MethodPost {
			if res.StatusCode != http.StatusOK {
				done <- fmt.Errorf("status code: %d", res.StatusCode)
				return
			}

			var challengeResponse struct {
				Challenge string `json:"challenge"`
			}
			if err := json.Unmarshal(body, &challengeResponse); err != nil {
				done <- fmt.Errorf("failed to unmarshal challenge response: %w", err)
				return
			}

			if challengeResponse.Challenge == "" {
				done <- nil
				h.Response.Fail(proto.NetworkErrorReasonAborted)
				return
			}
		}

		for k, vs := range res.Header {
			for _, v := range vs {
				h.Response.SetHeader(k, v)
			}
		}

		h.Response.SetBody(body)

		// Set the response code on the private :( payload field
		responseValue := reflect.ValueOf(h.Response).Elem()
		payloadField := responseValue.FieldByName("payload")
		if payloadField.IsValid() {
			payloadField = reflect.NewAt(payloadField.Type(), unsafe.Pointer(payloadField.UnsafeAddr())).Elem()
			if payloadField.CanInterface() {
				payload := payloadField.Interface().(*proto.FetchFulfillRequest)
				payload.ResponseCode = res.StatusCode
			} else {
				a.log.Error("cannot interface with internal payload field")
			}
		} else {
			a.log.Error("internal payload field is invalid")
		}
	})
	if err != nil {
		return err
	}

	go router.Run()
	defer router.Stop()

	page, err := stealth.Page(browser)
	if err != nil {
		return err
	}

	err = page.Navigate(stepUpUrl.String())
	if err != nil {
		return err
	}

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
