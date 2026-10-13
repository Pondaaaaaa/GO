package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Pondaaaaaa/GO/practices/pz14-notes-db/internal/notes"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	schema := flag.String("schema", "", "owned schema")
	optimized := flag.Bool("optimized", false, "keyset/batch mode")
	max := flag.Int("pool", 8, "max open and idle")
	flag.Parse()
	if *max < 1 || *max > 32 {
		log.Fatal("invalid pool")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := notes.Open(ctx, *max)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	repo, err := notes.New(ctx, db, *schema, *optimized)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("listen failed")
	}
	fmt.Printf("listening=%s\n", listener.Addr())
	srv := &http.Server{Handler: notes.Router(repo), ReadHeaderTimeout: 5 * time.Second}
	if err = srv.Serve(listener); err != nil {
		log.Fatal("server stopped")
	}
}
