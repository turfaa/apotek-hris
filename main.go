package main

import (
	"log"
	"time"

	"github.com/turfaa/apotek-hris/cmd/hris"

	"github.com/klauspost/lctime"
)

func main() {
	setupTime()

	if err := hris.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %s", err)
	}
}

func setupTime() {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Fatalf("LoadLocation: %s", err)
	}

	time.Local = loc
	lctime.SetLocale("id_ID")
}
