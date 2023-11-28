package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Starting Server")
	store, err := NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}
	if err = store.Init(); err != nil {
		log.Fatal(err)
	}
	server := NewAPIServer(":3000", store)
	server.Run()
}
