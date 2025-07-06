package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/saucesteals/amex"
	http "github.com/saucesteals/fhttp"
)

var (
	log = slog.New(tint.NewHandler(colorable.NewColorable(os.Stdout), &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.TimeOnly,
	}))
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Unexpected error", "error", r)
		}

		log.Info("Press ENTER to exit...")
		fmt.Scanln()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	profile, err := loadProfile()
	if err != nil {
		log.Error("Load profile", "error", err)
		return
	}

	credentials, err := profile.Credentials.Get()
	if err != nil {
		log.Error("Get credentials", "error", err)
		return
	}

	browserBin := os.Getenv("AMEX_BROWSER_BINARY")
	if browserBin == "" {
		browserBin = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if runtime.GOOS == "windows" {
			browserBin = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
		}
	}

	if _, err := os.Stat(browserBin); err != nil {
		if os.IsNotExist(err) {
			log.Error("Please install Google Chrome or set a custom binary with the AMEX_BROWSER_BINARY environment variable")
			return
		}

		log.Error("Check browser binary", "error", err)
		return
	}

	userDataDir, err := profile.GetDirectory("user_data")
	if err != nil {
		log.Error("Get user data directory", "error", err)
		return
	}

	api, err := amex.NewAPI(amex.Options{
		Credentials:         credentials,
		Logger:              log,
		BrowserUserDataPath: userDataDir,
		BrowserBinary:       browserBin,
	})
	if err != nil {
		log.Error("failed to create api", "error", err)
		return
	}

	cookies, err := profile.Cookies.Get()
	if err != nil {
		if !errors.Is(err, ErrResourceMissing) {
			log.Error("Get cookies", "error", err)
			return
		}

		cookies = []*http.Cookie{}
		err = profile.Cookies.Set(cookies)
		if err != nil {
			log.Error("Set cookies", "error", err)
			return
		}
	}

	api.SetCookies(cookies)

	cards, err := api.ReadEligibleCards(ctx)
	if err != nil {
		log.Info("Login required")

		err = api.Login(ctx)
		if err != nil {
			log.Error("failed to login", "error", err)
			return
		}

		log.Info("Login successful")

		cookies = api.GetCookies()
		err = profile.Cookies.Set(cookies)
		if err != nil {
			log.Error("Set cookies", "error", err)
			return
		}

		cards, err = api.ReadEligibleCards(ctx)
		if err != nil {
			log.Error("failed to read eligible cards", "error", err)
			return
		}
	} else {
		log.Info("Loaded saved session")
	}

	commands := []string{
		"create",
		"list",
		"delete",
		"exit",
	}

	for {
		var command string
		for {
			command = ask(fmt.Sprintf("Enter command (%s)", strings.Join(commands, ", ")))
			if slices.Contains(commands, command) {
				break
			}

			log.Error("Invalid command", "command", command)
		}

		if command == "exit" {
			log.Info("Exiting...")
			os.Exit(0)
			return
		}

		var card amex.EligibleCard
		if len(cards) == 0 {
			log.Error("No cards found")
			continue
		} else if len(cards) == 1 {
			card = cards[0]
		} else {
			fmt.Printf("Cards:\n")
			for i, card := range cards {
				fmt.Printf("%d. %s (%s)\n", i+1, card.Product.ProductName, card.AccountToken)
			}

			cardIndex, err := strconv.Atoi(ask("Select a card"))
			if err != nil {
				log.Error("Invalid card number")
				continue
			}

			cardIndex--

			if cardIndex < 0 || cardIndex > len(cards) {
				log.Error("Unknown card")
				continue
			}

			card = cards[cardIndex]
		}

		fmt.Printf("Selected card: %s (%s)\n", card.Product.ProductName, card.AccountToken)

		var err error
		switch command {
		case "list":
			err = list(ctx, api, card)
		case "delete":
			err = delete(ctx, api, card)
		case "create":
			err = create(ctx, profile, api, card)
		}

		if err != nil {
			log.Error("Error", "error", err)
		}
	}

}

func ask(prompt string) string {
	fmt.Printf("[?] %s: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func loadProfile() (*Profile, error) {
	username := ask("Enter username")

	profile, err := ImportProfile(username)
	if err != nil {
		return nil, fmt.Errorf("import profile: %w", err)
	}

	credentials, err := profile.Credentials.Get()
	if err != nil {
		if !errors.Is(err, ErrResourceMissing) {
			return nil, fmt.Errorf("get credentials: %w", err)
		}

		credentials = amex.Credentials{
			Username: username,
			Password: ask("Enter password"),
		}

		err = profile.Credentials.Set(credentials)
		if err != nil {
			return nil, fmt.Errorf("set credentials: %w", err)
		}
	}

	return profile, nil
}
