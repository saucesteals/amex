package amex

import (
	"log/slog"
	"net/url"
	"time"

	http "github.com/saucesteals/fhttp"
	"github.com/saucesteals/fhttp/cookiejar"
	"github.com/saucesteals/mimic"
)

type Credentials struct {
	Username string
	Password string
}

type Options struct {
	Credentials Credentials

	BrowserUserDataPath string
	BrowserBinary       string

	Logger *slog.Logger
}

type API struct {
	credentials Credentials
	log         *slog.Logger

	browserUserDataPath string
	browserBinary       string

	jar    *cookiejar.Jar
	client *http.Client
}

func NewAPI(options Options) (*API, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	transport, err := mimic.NewTransport(mimic.TransportOptions{
		Version:  "138.0.0.0",
		Brand:    mimic.BrandChrome,
		Platform: mimic.PlatformMac,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	})
	if err != nil {
		return nil, err
	}

	return &API{
		browserUserDataPath: options.BrowserUserDataPath,
		browserBinary:       options.BrowserBinary,

		credentials: options.Credentials,
		log:         options.Logger,
		jar:         jar,
		client: &http.Client{
			Jar:       jar,
			Transport: transport,
		},
	}, nil
}

var (
	cookieURL = &url.URL{
		Scheme: "https",
		Host:   ".americanexpress.com",
	}
)

func (a *API) GetCookies() []*http.Cookie {
	cookies := a.jar.Cookies(cookieURL)

	for _, cookie := range cookies {
		cookie.Domain = cookieURL.Host
		cookie.Expires = time.Now().Add(time.Hour * 24 * 30)
	}

	return cookies
}

func (a *API) SetCookies(cookies []*http.Cookie) {
	for _, cookie := range cookies {
		cookie.Domain = cookieURL.Host
		cookie.Expires = time.Now().Add(time.Hour * 24 * 30)
	}

	a.jar.SetCookies(
		cookieURL,
		cookies,
	)
}
