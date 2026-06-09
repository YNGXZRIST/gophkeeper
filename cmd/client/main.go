package main

// A simple example demonstrating the use of multiple text input components
// from the Bubbles component library.

import (
	"gophkeeper/internal/client/db"
	mg "gophkeeper/migrations/client"
	"log"
)

func main() {
	//TODO: move to app
	conn, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	_ = conn
	err = mg.Migrate()
	if err != nil {
		log.Fatal(err)
	}

}
