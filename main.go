package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"insta/insta"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var Cookies string
var AppID string

var firestoreClient *firestore.Client
var storageClient *storage.Client

var whiteListedUsers []string

var lastseen map[string]float64

func filterWhiteListed(reelsTrayData map[string]interface{}) []string {
	var okausers []string
	for _, user := range reelsTrayData["tray"].([]interface{}) {
		userid := user.(map[string]interface{})["id"].(string)
		for _, wuser := range whiteListedUsers {
			if userid == wuser {
				okausers = append(okausers, userid)
				break
			}
		}
	}
	return okausers
}

func initfire() {
	ctx := context.Background()
	opt := option.WithCredentialsFile(`D:\projects\instagram\instastories\instatrackercreds.json`)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	firestoreClient, err = app.Firestore(ctx)
	if err != nil {
		log.Fatal(err)
	}

	storageClient, err = storage.NewClient(ctx, opt)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	iter := firestoreClient.Collection("whitelist").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		whiteListedUsers = append(whiteListedUsers, doc.Data()["userid"].(string))
	}

	lastseen = make(map[string]float64)
	iter = firestoreClient.Collection("lastseen").Documents(context.Background())
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		uid := doc.Data()["userid"].(string)
		lst := doc.Data()["time"].(float64)

		lastseen[uid] = lst
	}

	instacreds, err := os.ReadFile("instacreds.txt")
	if err != nil {
		log.Fatal(err)
	}

	instacreds2 := strings.Split(string(instacreds), "\n")
	Cookies = instacreds2[0]
	AppID = instacreds2[1]

}

func storeNewReels(reels map[string]interface{}) {
	for _, reel := range reels["reels_media"].([]interface{}) {
		userid := reel.(map[string]interface{})["user"].(map[string]interface{})["pk"].(string)
		username := reel.(map[string]interface{})["user"].(map[string]interface{})["username"].(string)
		latestreel := reel.(map[string]interface{})["latest_reel_media"].(float64)

		for _, story := range reel.(map[string]interface{})["items"].([]interface{}) {
			takenat := story.(map[string]interface{})["taken_at"].(float64)
			mediatype := story.(map[string]interface{})["media_type"].(float64)
			mediaurl := ""
			mediaid := story.(map[string]interface{})["id"].(string)
			switch mediatype {
			case 2:
				mediaurl = story.(map[string]interface{})["video_versions"].([]interface{})[0].(map[string]interface{})["url"].(string)
				log.Println(userid, username, latestreel, takenat, mediatype, mediaid, mediaurl)
				if isNewStory(userid, takenat) {
					log.Println("uploading the file")
					ts := time.Unix(int64(takenat), 0)
					uploadFiles(mediaurl, username+"_"+mediaid+"_"+ts.Format("2006_01_02_15_04_05")+".mp4", username)
					lastseen[userid] = takenat
				}
			case 1:
				mediaurl = story.(map[string]interface{})["image_versions2"].(map[string]interface{})["candidates"].([]interface{})[0].(map[string]interface{})["url"].(string)
				log.Println(userid, username, latestreel, takenat, mediatype, mediaid, mediaurl)
				if isNewStory(userid, takenat) {
					log.Println("uploading the file")
					ts := time.Unix(int64(takenat), 0)
					uploadFiles(mediaurl, username+"_"+mediaid+"_"+ts.Format("2006_01_02_15_04_05")+".jpg", username)
					lastseen[userid] = takenat
				}
			}
			// return
		}
	}
}

func uploadLastSeen() {

	for k, v := range lastseen {

		_, err := firestoreClient.Collection("lastseen").Doc(k).Set(context.Background(), map[string]interface{}{
			"userid": k,
			"time":   v,
		}, firestore.MergeAll)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func isNewStory(userid string, takenat float64) bool {
	if lst, ok := lastseen[userid]; ok {
		if lst >= takenat {
			return false
		}
	}
	return true
}

func uploadFiles(mediaurl, filename, username string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	res, err := http.Get(mediaurl)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	wc := storageClient.Bucket("instatracker-2aaaf.appspot.com").Object(username + "/" + filename).NewWriter(ctx)
	wc.ChunkSize = 0

	_, err = io.Copy(wc, res.Body)
	if err != nil {
		log.Fatal(err)
	}

	if err := wc.Close(); err != nil {
		log.Fatal(err)
	}

}

func main() {
	log.Println("started")

	initfire()
	defer firestoreClient.Close()
	defer storageClient.Close()

	log.Println(whiteListedUsers)

	myInstagram := insta.Insta{}
	myInstagram.Init(Cookies, AppID)

	reelsTray := myInstagram.GetMyRealsTray()

	filteredUsers := filterWhiteListed(reelsTray)

	userStories := myInstagram.GetStories(filteredUsers)

	// log.Println(userStories)

	storeNewReels(userStories)

	uploadLastSeen()

}
