package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	//Load env
	err := godotenv.Load()
	if err != nil {
		log.Println("WARNING: No .env file found...")
	}

	log.Println("starting Server")

	go registerWebhook()

	//Create a new Mux Handler
	m := mux.NewRouter()
	//Listen to the base url and send a response
	m.HandleFunc("/", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(200)
		if _, err := fmt.Fprintf(writer, "Server is up and running"); err != nil {
			log.Printf("couldn't write response to / request: %v \n", err)
			return
		}
	})
	//Listen to crc check and handle
	m.HandleFunc("/webhook/twitter", CrcCheck).Methods("GET")
	//Listen to webhook event and handle
	m.HandleFunc("/webhook/twitter", WebhookHandler).Methods("POST")

	//Start Server
	server := &http.Server{
		Handler: m,
	}
	server.Addr = determineListenAddress()
	log.Printf("Listening on %s...\n", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func determineListenAddress() string {
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("$PORT not set, using :80 as default")
		return ":80"
	}
	return ":" + port
}

func WebhookHandler(_ http.ResponseWriter, request *http.Request) {
	log.Println("webhook Handler called")
	//Read the body of the tweet
	body, _ := ioutil.ReadAll(request.Body)
	log.Printf("request: %s \n", string(body))
	//Initialize a webhook load object for json decoding
	var load WebhookLoad
	err := json.Unmarshal(body, &load)
	if err != nil {
		log.Printf("can't unmarshal webhook request body: %v", err)
		return
	}
	//Check if it was a tweet_create_event and tweet was in the payload and it was not tweeted by the bot
	if len(load.TweetCreateEvent) < 1 || load.UserId == load.TweetCreateEvent[0].User.IdStr {
		return
	}
	//Send Hello world as a reply to the tweet, replies need to begin with the handles
	//of accounts they are replying to
	_, err = SendTweet("@"+load.TweetCreateEvent[0].User.Handle+" Hello World", load.TweetCreateEvent[0].IdStr)
	if err != nil {
		log.Printf("can't send tweet: %v", err)
		return
	}
}

func CrcCheck(writer http.ResponseWriter, request *http.Request) {
	//Set response header to json type
	writer.Header().Set("Content-Type", "application/json")
	//Get crc token in parameter
	token := request.URL.Query()["crc_token"]
	if len(token) < 1 {
		if _, err := fmt.Fprintf(writer, "No crc_token given"); err != nil {
			log.Printf("[No crc_token given] couldn't write response to CrcCheck request: %v \n", err)
			return
		}
		return
	}

	//Encrypt and encode in base 64 then return
	h := hmac.New(sha256.New, []byte(os.Getenv("CONSUMER_SECRET")))
	h.Write([]byte(token[0]))
	encoded := base64.StdEncoding.EncodeToString(h.Sum(nil))
	//Generate response string map
	response := make(map[string]string)
	response["response_token"] = "sha256=" + encoded
	//Turn response map to json and send it to the writer
	responseJson, _ := json.Marshal(response)
	if _, err := fmt.Fprintf(writer, string(responseJson)); err != nil {
		log.Printf("couldn't write token response to CrcCheck request: %v \n", err)
		return
	}
}
