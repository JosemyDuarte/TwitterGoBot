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
)

func main() {
	log.Println("starting server...")
	defer log.Println("shutting down server...")
	go registerWebhook()

	m := mux.NewRouter()
	m.HandleFunc("/", ServerUpHandler)
	m.HandleFunc("/webhook/twitter", WebhookHandler).Methods("POST")
	m.HandleFunc("/webhook/twitter", CrcCheckHandler).Methods("GET")

	server := &http.Server{
		Handler: m,
	}
	server.Addr = determineListenAddress()
	log.Printf("Listening on %s...\n", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func ServerUpHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(200)
	if _, err := fmt.Fprintf(writer, "Server is up and running"); err != nil {
		log.Printf("couldn't write response to / request: %v \n", err)
		return
	}
}

func WebhookHandler(_ http.ResponseWriter, request *http.Request) {
	log.Println("webhook handler called")

	body, _ := ioutil.ReadAll(request.Body)
	log.Printf("request: %s \n", string(body))

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

	_, err = SendTweet("@"+load.TweetCreateEvent[0].User.Handle+" Hello!", load.TweetCreateEvent[0].IdStr)
	if err != nil {
		log.Printf("can't send tweet: %v", err)
		return
	}
}

func CrcCheckHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	token := request.URL.Query()["crc_token"]
	if len(token) < 1 {
		if _, err := fmt.Fprintf(writer, "No crc_token given"); err != nil {
			log.Printf("[No crc_token given] couldn't write response to CrcCheck request: %v \n", err)
			return
		}
		return
	}

	h := hmac.New(sha256.New, []byte(os.Getenv("CONSUMER_SECRET")))
	h.Write([]byte(token[0]))
	encoded := base64.StdEncoding.EncodeToString(h.Sum(nil))
	response := make(map[string]string)
	response["response_token"] = "sha256=" + encoded
	responseJson, _ := json.Marshal(response)
	if _, err := fmt.Fprintf(writer, string(responseJson)); err != nil {
		log.Printf("couldn't write token response to CrcCheck request: %v \n", err)
		return
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
