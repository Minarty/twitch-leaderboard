package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/jasonlvhit/gocron"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"html/template"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"sort"
)

const ServiceAddr = "localhost:32812"
const ImageFile = "leaderboard.png"

var ctx = context.Background()
var currentBg int
var backgrounds []fs.FileInfo
var config Config

func main() {
	println("Starting TwitchLeaderboard")

	// Add all file names in the "bg" directory to our backgrounds array
	backgrounds, _ = ioutil.ReadDir("bg")

	// Read and parse the config JSON file into the config variable
	content, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	jsonError := json.Unmarshal(content, &config)
	if jsonError != nil {
		log.Fatal(jsonError)
	}

	println("Starting HTTP server")
	fileServer := http.FileServer(http.Dir("./site/static"))
	http.Handle("/", fileServer)
	http.HandleFunc("/bg", bgHandler)
	http.HandleFunc("/lb", handler)

	// The HTTP server is sync, we need to run it in a goroutine
	go func() {
		log.Fatal(http.ListenAndServe(ServiceAddr, nil))
	}()

	startWebDriver()
	startStream()

	// Schedule to refresh image every 20 seconds, this is blocking
	_ = gocron.Every(20).Seconds().Do(refresh)
	<-gocron.Start()
}

// Gets the leaderboard sorted by XP in descending order
// This is limited to 3 entries, which is what we need
func getSortedLeaderboard() [3]LeaderboardEntry {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: config.Redis.Password,
		DB:       0,
	})

	val, _ := rdb.ZRangeWithScores(ctx, config.Redis.Key, 0, -1).Result()

	// Sort the values by score
	sort.Slice(val, func(i, j int) bool {
		return val[i].Score > val[j].Score
	})

	// Limit to 3 entries
	val = val[0:int(math.Min(float64(len(val)), float64(3)))]

	var profiles [3]LeaderboardEntry

	// Create LeaderboardEntry of the redis scores
	for i := 0; i < 3; i++ {
		var profile LeaderboardEntry
		if i < len(val) {
			s := val[i]
			uuid := fmt.Sprintf("%v", s.Member)
			name, _ := rdb.Get(ctx, uuid).Result()

			profile = LeaderboardEntry{
				Place: i,
				UUID:  uuid,
				Name:  fmt.Sprintf("%v", name),
				XP:    s.Score,
			}
		} else {
			profile = LeaderboardEntry{
				Place: i,
				UUID:  "Steve",
				Name:  "Ingen",
				XP:    0,
			}
		}

		profiles[i] = profile
	}

	return profiles
}

var webDriver selenium.WebDriver

func startWebDriver() {
	const (
		seleniumPath = "selenium/selenium.jar"
		chromeDriver = "selenium/chromedriver"
		port         = 4444
	)

	opts := []selenium.ServiceOption{
		selenium.Output(nil),
		selenium.StartFrameBufferWithOptions(selenium.FrameBufferOptions{ScreenSize: "1920x1080x24"}),
		selenium.ChromeDriver(chromeDriver),
	}

	_, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		log.Fatal(err)
		return
	}

	selenium.SetDebug(false)

	seleniumConfig := selenium.Capabilities{"browserName": "chrome"}
	chromeConfig := chrome.Capabilities{
		Path: "",
		Args: []string{
			"--headless",
			"--start-maximized",
			"--window-size=1920x1080",
		},
	}

	seleniumConfig.AddChrome(chromeConfig)
	webDriver, _ = selenium.NewRemote(seleniumConfig, fmt.Sprintf("http://localhost:%d/wd/hub", port))

	if err := webDriver.Get(fmt.Sprintf("http://%s/lb", ServiceAddr)); err != nil {
		panic(err)
	}
}

func refresh() {
	fmt.Println("Refreshing...")
	err := webDriver.Refresh()
	if err != nil {
		log.Fatal(err)
		return
	}

	b, _ := webDriver.Screenshot()
	err = ioutil.WriteFile(ImageFile, b, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// Handles /lb HTTP requests
func handler(w http.ResponseWriter, _ *http.Request) {
	disableCache(w)
	t, _ := template.ParseFiles("site/index.html")

	profiles := getSortedLeaderboard()
	_ = t.Execute(w, profiles)
}

// Handles /bg HTTP requests
func bgHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers to prevent browser from caching previous image
	disableCache(w)
	http.ServeFile(w, r, fmt.Sprintf("bg/%s", getNextBackground()))
}

// Runs the stream with FFMPEG
func startStream() {
	cmd := exec.Command("ffmpeg", "-nostdin", "-framerate", "2", "-re", "-loop", "1", "-i", ImageFile,
		"-f", "flv", "-vcodec", "libx264", "-pix_fmt", "yuv420p", "-preset", "veryslow",
		"-r", "2", "-g", "4", "-s", "1280x720", fmt.Sprintf("rtmp://live-cdg.twitch.tv/app/%s", config.Twitch.RtmpToken),
		"-nostats")

	cmd.Dir = getWorkingDirectory()
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}

// Disables caching for the request, this is needed since the file name
// for the background is always the same, causing the browser to cache it
func disableCache(w http.ResponseWriter) {
	header := w.Header()
	header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	header.Set("Pragma", "no-cache")
	header.Set("Expires", "0")
}

// Gets the next background in the file name array or 0 if reached the end
func getNextBackground() string {
	if currentBg == len(backgrounds) { // Loop back
		currentBg = 0
	}

	name := backgrounds[currentBg].Name()
	currentBg++

	return name
}

// Gets the current working directory
func getWorkingDirectory() string {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
		return ""
	}

	return path
}

// Config Main config struct for parsing config.json
type Config struct {
	Redis  Redis
	Twitch Twitch
}

// Redis Config struct for "redis" settings json object
type Redis struct {
	Address  string
	Password string
	Key      string
}

// Twitch Config struct for "twitch" settings json object
type Twitch struct {
	RtmpToken string `json:"rtmp_token"`
}

// LeaderboardEntry Struct for an entry found in the leaderboard
type LeaderboardEntry struct {
	Place int
	UUID  string
	Name  string
	XP    float64
}
