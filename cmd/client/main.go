package main

// A simple example demonstrating the use of multiple text input components
// from the Bubbles component library.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/client/db"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/view/tui/root"
	mg "gophkeeper/migrations/client"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
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
	sessionRepo := repository.NewSessionRepo(conn)
	session, err := sessionRepo.Get(context.Background())
	if err != nil {
		switch true {
		case errors.Is(err, sql.ErrNoRows):
			// continue
			break
		case errors.Is(err, repository.ErrNoSession):
			//continue
			break
		default:
			err, ok := errors.AsType[*repository.ErrParseToken](err)
			if !ok {
				log.Fatal(err)
			}
			fmt.Printf("error:%+v\n", err)
		}
	}
	if _, err = tea.NewProgram(root.New()).Run(); err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
	_ = session

}
