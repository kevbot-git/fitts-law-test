package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"

	"os"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

const (
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
	conn  *websocket.Conn
	d     Dimensions
	stats []ClickStats
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

// ClickStats show the location of the circle and the coordinates of the click that removed it.
type ClickStats struct {
	CircleX    int `json:"circleX"`
	CircleY    int `json:"circleY"`
	ClickX     int `json:"clickX"`
	ClickY     int `json:"clickY"`
	Dimensions `json:"dimensions"`
}

func main() {
	r := httprouter.New()

	r.GET("/", Index)
	r.GET("/ws", Game)

	s := &http.Server{
		Addr:         ":" + os.Getenv("PORT"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		Handler:      r,
	}
	log.Fatal(s.ListenAndServe())
}

// NewCircle creates a new random circle.
func NewCircle(d Dimensions) Circle {
	size := randomInt(5, 120)

	x := randomInt(0, d.Width)
	y := randomInt(0, d.Height)

	// Make sure the circle doesn't go out of bounds.
	if x-size < 0 {
		x += size
	}
	if x+size > d.Width {
		x -= size
	}
	if y+size > d.Height {
		y -= size
	}
	if y-size < 0 {
		y += size
	}

	return Circle{
		X: x,
		Y: y,
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

func (c ClickStats) difference() (int, int) {
	return 0, 0
}

func (c ClickStats) String() string {
	return fmt.Sprintf("Circle located at: X (%d), Y (%d) with dimensions: Width (%d), Height (%d), was clicked at: X (%d), Y (%d)",
		c.CircleX, c.CircleY, c.Width, c.Height, c.ClickX, c.ClickY)
}

func (g game) sendCircle() error {
	c := NewCircle(g.d)
	return g.conn.WriteJSON(&c)
}

func (g game) start() {
	count := 1

	// Send the initial circle.
	if err := g.sendCircle(); err != nil {
		log.Fatal(err)
	}

	for {
		if count == 20 {
			break
		}

		// Receive the click stats for the given circle.
		var cs ClickStats
		if err := g.conn.ReadJSON(&cs); err != nil {
			log.Fatal(err)
		}
		g.stats = append(g.stats, cs)
		fmt.Println(cs)
		count++

		if err := g.sendCircle(); err != nil {
			log.Fatal(err)
		}
	}
}

// Game is the main (Websocket) handler for the Fitts' Law game.
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

// Index page runs the index.html page which contains the game client.
func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	t, err := template.ParseFiles(wd + "/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	t.Execute(w, nil)
}
