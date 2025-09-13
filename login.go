package amex

import (
	"context"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	http "github.com/saucesteals/fhttp"
)

var (
	loginURL    = "https://www.americanexpress.com/en-us/account/login?DestPage=https%3A%2F%2Fwww.americanexpress.com%2Fen-us%2Fbusiness%2Fvirtual-card%2Fmanage-cards"
	homepageURL = "https://www.americanexpress.com/"
)

func (a *API) initBrowser() (*rod.Browser, error) {
	ctx := context.Background()
	l := launcher.New().
		Context(ctx).
		Headless(false).
		UserDataDir(a.browserUserDataPath).
		Bin(a.browserBinary)

	controlUrl, err := l.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().Context(ctx).ControlURL(controlUrl)

	if err := browser.Connect(); err != nil {
		return nil, err
	}

	cookies := a.GetCookies()
	var browserCookies []*proto.NetworkCookieParam
	for _, cookie := range cookies {
		browserCookies = append(browserCookies, &proto.NetworkCookieParam{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: ".americanexpress.com",
		})
	}

	return browser, browser.SetCookies(browserCookies)
}

func (a *API) syncCookies(browser *rod.Browser) error {
	cookies, err := browser.GetCookies()
	if err != nil {
		return err
	}

	var browserCookies []*http.Cookie
	for _, cookie := range cookies {
		if !strings.HasSuffix(cookie.Domain, cookieURL.Host) {
			continue
		}

		browserCookies = append(browserCookies, &http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	a.SetCookies(browserCookies)
	return nil
}

func (a *API) Login(ctx context.Context) error {
	browser, err := a.initBrowser()
	if browser != nil {
		defer browser.Close()
	}
	if err != nil {
		return err
	}

	page, err := stealth.Page(browser)
	if err != nil {
		return err
	}

	err = page.Navigate(homepageURL)
	if err != nil {
		return err
	}

	err = page.Navigate(loginURL)
	if err != nil {
		return err
	}

	defer page.Close()

	typeElement := func(selector string, text string) error {
		element, err := page.Element(selector)
		if err != nil {
			return err
		}

		currentValue, err := element.Eval(`() => this.value`)
		if err != nil {
			return err
		}

		if value := currentValue.Value.Str(); value == text {
			return nil
		} else if value != "" {
			for i := 0; i < len(value); i++ {
				err := element.Type(input.Backspace)
				if err != nil {
					return err
				}
			}
		}

		for _, char := range text {
			err := element.Input(string(char))
			if err != nil {
				return err
			}

			time.Sleep(100 * time.Millisecond)
		}

		return nil
	}

	clickElement := func(selector string) error {
		_, err := page.Eval(`(selector) => {
			const element = document.querySelector(selector);
			if (element) {
				element.click();
			}
		}`, selector)
		if err != nil {
			return err
		}

		return nil
	}

	err = typeElement("#eliloUserID", a.credentials.Username)
	if err != nil {
		return err
	}

	err = typeElement(`#eliloPassword`, a.credentials.Password)
	if err != nil {
		return err
	}

	if isCheckedValue, err := page.Eval(`() => document.querySelector('#rememberMe').checked`); err != nil {
		return err
	} else if !isCheckedValue.Value.Bool() {
		err = clickElement(`#rememberMe`)
		if err != nil {
			return err
		}
	}

	err = clickElement(`#loginSubmit`)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			info, err := page.Info()
			if err != nil {
				return err
			}

			if info.URL == "https://www.americanexpress.com/en-us/business/virtual-card/manage-cards" {
				err = a.syncCookies(browser)
				if err != nil {
					return err
				}

				return nil
			}
		}
	}
}
