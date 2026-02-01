package main

import (
	"log"

	"igcmailimap/ui"
)

func main() {
	app, err := ui.New()
	if err != nil {
		log.Fatal(err)
	}
	app.Run()
}
