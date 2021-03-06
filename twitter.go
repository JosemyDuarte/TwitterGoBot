package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/dghubble/oauth1"
)

type WebhookLoad struct {
	UserId           string  `json:"for_user_id"`
	TweetCreateEvent []Tweet `json:"tweet_create_events"`
}

type Tweet struct {
	Id    int64
	IdStr string `json:"id_str"`
	User  User
	Text  string
}

type User struct {
	Id     int64
	IdStr  string `json:"id_str"`
	Name   string
	Handle string `json:"screen_name"`
}

func createClient() *http.Client {
	config := oauth1.NewConfig(os.Getenv("CONSUMER_KEY"), os.Getenv("CONSUMER_SECRET"))
	token := oauth1.NewToken(os.Getenv("ACCESS_TOKEN_KEY"), os.Getenv("ACCESS_TOKEN_SECRET"))

	return config.Client(oauth1.NoContext, token)
}

func registerWebhook() {
	log.Println("registering webhook...")
	httpClient := createClient()

	path := "https://api.twitter.com/1.1/account_activity/all/" + os.Getenv("WEBHOOK_ENV") + "/webhooks.json"
	values := url.Values{}
	values.Set("url", os.Getenv("APP_URL")+"/webhook/twitter")

	_, _ = httpClient.PostForm(path, values)
	log.Println("webhook has been registered")
	subscribeWebhook()
}

func subscribeWebhook() {
	log.Println("subscribing webapp...")
	client := createClient()
	path := "https://api.twitter.com/1.1/account_activity/all/" + os.Getenv("WEBHOOK_ENV") + "/subscriptions.json"
	resp, _ := client.PostForm(path, nil)
	body, _ := ioutil.ReadAll(resp.Body)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("something went wrong closing body response")
			return
		}
	}()

	if resp.StatusCode != 204 {
		log.Println("could not subscribe the webhook. Response below:")
		log.Println(string(body))
	}

	log.Println("subscribed successfully")
}

func SendTweet(tweet string, replyId string) (*Tweet, error) {
	log.Println("sending tweet as reply to " + replyId)

	params := url.Values{}
	params.Set("status", tweet)
	params.Set("in_reply_to_status_id", replyId)

	client := createClient()
	resp, err := client.PostForm("https://api.twitter.com/1.1/statuses/update.json", params)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("something went wrong closing body response")
			return
		}
	}()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("twitter response of tweet sent: %s \n", string(body))

	var responseTweet Tweet
	err = json.Unmarshal(body, &responseTweet)
	if err != nil {
		return nil, err
	}

	log.Println("tweet sent successfully")
	return &responseTweet, nil
}
