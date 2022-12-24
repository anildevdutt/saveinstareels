package insta

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

const ReelsTrayUrl = "https://i.instagram.com/api/v1/feed/reels_tray/"
const UserStoriesURL = "https://i.instagram.com/api/v1/feed/reels_media/"

func checkerr(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

type Insta struct {
	Cookies string
	AppID   string
}

func (instaClient *Insta) Request(requetURL string) string {
	log.Println("Request URl: " + requetURL)
	req, err := http.NewRequest("GET", requetURL, nil)
	checkerr(err)

	req.Header.Add("cookie", instaClient.Cookies)
	req.Header.Add("x-ig-app-id", instaClient.AppID)

	client := &http.Client{}
	res, err := client.Do(req)
	checkerr(err)
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	checkerr(err)

	// log.Println(string(data))

	return string(data)
}

func (instaClient *Insta) Init(cookies, appid string) {
	instaClient.Cookies = cookies
	instaClient.AppID = appid
}

func (instaClient *Insta) GetMyRealsTray() map[string]interface{} {
	reelsTrayData := instaClient.Request(ReelsTrayUrl)

	var reelsTray map[string]interface{}

	err := json.Unmarshal([]byte(reelsTrayData), &reelsTray)
	checkerr(err)

	return reelsTray
}

func (instaClient *Insta) GetStories(userids []string) map[string]interface{} {
	userStoriesURL := UserStoriesURL + "?"
	for _, userid := range userids {
		userStoriesURL += "reel_ids=" + userid + "&"
	}
	userStoriesURL = userStoriesURL[0 : len(userStoriesURL)-1]

	storiesData := instaClient.Request(userStoriesURL)

	var stories map[string]interface{}

	err := json.Unmarshal([]byte(storiesData), &stories)
	checkerr(err)

	return stories
}
