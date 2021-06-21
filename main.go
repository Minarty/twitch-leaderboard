package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/jasonlvhit/gocron"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"sort"
)

const ImageFile = "leaderboard.png"

var ctx = context.Background()
var config Config

func main() {
	initialize()

	println("Starting TwitchLeaderboard")

	content, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	jsonError := json.Unmarshal(content, &config)
	if jsonError != nil {
		log.Fatal(jsonError)
	}

	refresh()
	startStream()

	// Schedule to refresh image every 20 seconds, this is blocking
	_ = gocron.Every(20).Seconds().Do(refresh)
	<-gocron.Start()
}

func refresh() {
	fmt.Println("Refreshing...")
	profiles := getSortedLeaderboard()
	render(profiles)
}

// Runs the stream with FFMPEG
func startStream() {
	fmt.Println("Starting stream...")
	cmd := exec.Command("ffmpeg", "-nostdin", "-framerate", "2", "-re", "-loop", "1", "-i", ImageFile,
		"-f", "flv", "-vcodec", "libx264", "-pix_fmt", "yuv420p", "-preset", "veryslow",
		"-r", "2", "-g", "4", "-s", "1280x720", fmt.Sprintf("rtmp://live-cdg.twitch.tv/app/%s", config.Twitch.RtmpToken),
		"-nostats")

	fmt.Println("Running ffmpeg \"" + cmd.String() + "\"")

	cmd.Dir = getWorkingDirectory()
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
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

// Gets the current working directory
func getWorkingDirectory() string {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
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
