package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gregdel/pushover"
	"github.com/spf13/viper"
	"github.com/tebeka/selenium"
)

// PO is the poushover wrapper type
type PO struct {
	app  *pushover.Pushover
	user *pushover.Recipient
}

// NewPO creates a new PO
func NewPO(app *pushover.Pushover, user *pushover.Recipient) *PO {
	return &PO{app, user}
}

// Notify push the message
func (po *PO) Notify(msg, title, url string) {
	poMsg := pushover.NewMessageWithTitle(msg, title)
	poMsg.URL = url
	res, err := po.app.SendMessage(poMsg, po.user)
	if err != nil {
		log.Panic(err)
	}
	log.Println(res)
}

func main() {
	// Set the config location
	cwd, _ := os.Getwd()
	conf := flag.String("c", "gpw.toml", "The config.[toml|yml] to use.")
	version := flag.Bool("v", false, "Print version.")
	flag.Parse()

	if *version {
		bi, _ := debug.ReadBuildInfo()
		fmt.Printf("%v\n", bi.Main.Version)
		os.Exit(0)
	}

	log.Println("--- Starting galaxus-price-watcher")

	// load config
	if *conf != "gpw.toml" {
		configPath, err := filepath.Abs(*conf)
		if err != nil {
			panic(err)
		}
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("gpw")
		viper.AddConfigPath(cwd)
	}

	// read the config file
	if err := viper.ReadInConfig(); err != nil { // handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	var (
		// Debug enable
		debugEnabled = viper.GetBool("general.debug")
		// Notification level - 0: nothing, 1: errors, 2: shukkin, and 3: everything to notify with PushOver
		notificationLevel = viper.GetInt("general.notification-level")
		// Preflight sleep - if true sleep for random minutes
		preflightSleep = viper.GetBool("general.preflight-sleep")
		// Preflight sleep Max minutes for the preflight sleep
		preflightSleepMaxMinutes = viper.GetInt("general.preflight-sleep-max")
		// Path to the sqlite3 directory
		dbDirPath         = viper.GetString("general.sqlite3dir")
		poAPIToken        = viper.GetString("pushover.api-token")
		poUserKey         = viper.GetString("pushover.user-key")
		webdriverPath     = viper.GetString("webdriver.path")
		seleniumPath      = viper.GetString("selenium.path")
		seleniumPort      = viper.GetInt("selenium.port")
		seleniumRemoteURL = viper.GetString("selenium.remote-url")
	)
	seleniumPath, err := filepath.Abs(seleniumPath)
	if err != nil {
		panic(err)
	}
	webdriverPath, err = filepath.Abs(webdriverPath)
	if err != nil {
		panic(err)
	}

	// pre-flight sleep config
	if preflightSleep && preflightSleepMaxMinutes > 0 {
		rand.Seed(time.Now().UTC().UnixNano())
		minute := rand.Intn(preflightSleepMaxMinutes) // [0-59]
		log.Printf("Sleeping for %v minutes...", minute)
		time.Sleep(time.Minute * time.Duration(minute))
	}

	// Create a new pushover app and user
	var po *PO
	if notificationLevel > 0 {
		poApp := pushover.New(poAPIToken)
		poUser := pushover.NewRecipient(poUserKey)
		po = NewPO(poApp, poUser)
	}

	if _, err := os.Stat(webdriverPath); os.IsNotExist(err) {
		panic(err)
	}
	opts := []selenium.ServiceOption{
		selenium.GeckoDriver(webdriverPath), // Specify the path to GeckoDriver in order to use Firefox.
	}
	if runtime.GOOS != "darwin" {
		opts = append(opts, selenium.StartFrameBuffer()) // Start an X frame buffer for the browser to run in.
	}
	if debugEnabled {
		opts = append(opts, selenium.Output(os.Stderr)) // Output debug information to STDERR.
	}

	//selenium.SetDebug(true)
	if _, err := os.Stat(seleniumPath); os.IsNotExist(err) {
		panic(err)
	}
	log.Printf("Launching the selenium: %s", seleniumPath)
	service, err := selenium.NewSeleniumService(seleniumPath, seleniumPort, opts...)
	if err != nil {
		panic(err) // panic is used only as an example and is not otherwise recommended.
	}
	defer service.Stop()

	// connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{
		"browserName": "firefox",
	}
	log.Printf("Launching a new remote: %s:%v", seleniumRemoteURL, seleniumPort)
	wd, err := selenium.NewRemote(caps, fmt.Sprintf(seleniumRemoteURL, seleniumPort))
	if err != nil {
		panic(err)
	}
	wd.SetAsyncScriptTimeout(time.Second * 5)
	wd.SetImplicitWaitTimeout(time.Second * 1)
	//wd.SetPageLoadTimeout(time.Second * 30)
	defer wd.Quit()

	// check the items
	items := viper.GetStringMap("items")
	for itemid := range items {
		url := viper.GetString(fmt.Sprintf("items.%s.url", itemid))
		name := viper.GetString(fmt.Sprintf("items.%s.name", itemid))
		lastPrice := viper.GetString(fmt.Sprintf("items.%s.price", itemid))
		lastAvailability := viper.GetString(fmt.Sprintf("items.%s.availability", itemid))
		log.Printf("Checking \"%s\": %s", name, url)

		// get the item page
		if err = wd.Get(url); err != nil {
			if notificationLevel > 0 {
				po.Notify(
					"Something went wrong",
					fmt.Sprintf("[gpw] %s", name),
					url,
				)
			}
			panic(err)
		}

		// check the item price
		itemPriceElem, err := wd.FindElement(selenium.ByCSSSelector, "#pageContent > div > div > div > div > div > div > span > strong")
		//angelNameElem, err := wd.FindElement(selenium.ByCSSSelector, "#girlprofile > h4 > span > table > tbody > tr > td:nth-child(1)")
		if err != nil {
			// process the main body of the diary
			if notificationLevel > 0 {
				po.Notify(
					"Something went wrong",
					fmt.Sprintf("[dpw] %s", name),
					url,
				)
			}
			panic(err)
		}
		itemPrice, _ := itemPriceElem.Text()

		// check the item availability
		itemAvailabilityElem, err := wd.FindElements(selenium.ByCSSSelector, ".availabilityText > div > div")
		if err != nil {
			panic(err)
		}
		itemAvailability, _ := itemAvailabilityElem.Text()

		// check notification condition
		updated := false

		if lastPrice != itemPrice {
			viper.Set(fmt.Sprintf("items.%s.price", itemid), itemPrice)
			updated = true
		}

		if lastAvailability != itemAvailability {
			viper.Set(fmt.Sprintf("items.%s.availability", itemid), itemAvailability)
			updated = true
		}

		if updated && notificationLevel > 2 {
			po.Notify(
				itemAvailability,
				fmt.Sprintf("[dpw] %s: CHF %s", name, itemPrice),
				url,
			)
		} // body end
	} // items end

	// Wrap-up
	log.Println("All the items checked - exit saving the config")
	viper.WriteConfig()
}
