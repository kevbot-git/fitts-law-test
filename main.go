package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

const (
	addr         = ":8080"
	readTimeout  = time.Second * 5
	writeTimeout = time.Second * 10
	bufferSize   = 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  bufferSize,
	WriteBufferSize: bufferSize,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type game struct {
	conn *websocket.Conn
	d    Dimensions
}

// Dimensions of the users browser window.
type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// A Circle placed on the browser window.
type Circle struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	Dimensions `json:"dimensions"`
}

// NewCircle creates a new random circle.
func NewCircle(maxX, maxY int) Circle {
	size := randomInt(30, 120)

	return Circle{
		X: randomInt(0, maxX),
		Y: randomInt(0, maxY),
		Dimensions: Dimensions{
			Width:  size,
			Height: size,
		},
	}
}

func randomInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

func (g game) start() {
	// Every second send a randomly sized circle to the client.
	for range time.Tick(time.Second) {
		c := NewCircle(g.d.Width, g.d.Height)
		if err := g.conn.WriteJSON(&c); err != nil {
			log.Fatal(err)
		}
	}
}

// Game is the main handler for the Fitts' Law game.
func Game(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("error upgrading connection to websocket: %v", err), 500)
		return
	}

	var d Dimensions
	if err := conn.ReadJSON(&d); err != nil {
		http.Error(w, fmt.Sprintf("error getting screen dimensions: %v", err), 500)
		return
	}

	log.Printf("Recieved dimensions: Width: %d, Height: %d\n", d.Width, d.Height)

	g := game{conn: conn, d: d}
	g.start()
}

func main() {
	r := httprouter.New()
	r.GET("/play", Game)

	s := &http.Server{
		Addr:         addr,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		Handler:      r,
	}
	log.Fatal(s.ListenAndServe())
}
