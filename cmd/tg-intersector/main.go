package main

import (
	"log"
	"os"
	"strconv"

	"github.com/idkroff/tg-intersector/internal/telegram"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err == nil {
		log.Print("Loaded .env file")
	}
}

func main() {
	TG_API_ID_STR, exists := os.LookupEnv("TG_API_ID")
	if !exists {
		log.Fatal("TG_API_ID not provided")
	}

	TG_API_ID, err := strconv.Atoi(TG_API_ID_STR)
	if err != nil {
		log.Fatalf("Unable to parse TG_API_ID \"%s\": %s", TG_API_ID_STR, err)
	}

	TG_API_HASH, exists := os.LookupEnv("TG_API_HASH")
	if !exists {
		log.Fatal("TG_API_HASH not provided")
	}

	TG_PHONE := ""
	TG_PHONE_ENV, exists := os.LookupEnv("TG_PHONE")
	if exists {
		TG_PHONE = TG_PHONE_ENV
	}

	TG_PASSWORD := ""
	TG_PASSWORD_ENV, exists := os.LookupEnv("TG_PASSWORD")
	if exists {
		TG_PASSWORD = TG_PASSWORD_ENV
	}

	AUTH_FLOW := "code"
	AUTH_FLOW_ENV, exists := os.LookupEnv("AUTH_FLOW")
	if exists {
		AUTH_FLOW = AUTH_FLOW_ENV
	}

	client := telegram.IntersectorClient{
		API_ID:   TG_API_ID,
		API_HASH: TG_API_HASH,
		Phone:    TG_PHONE,
		Password: TG_PASSWORD,
		AuthFlow: AUTH_FLOW,
	}

	client.Run()
}
