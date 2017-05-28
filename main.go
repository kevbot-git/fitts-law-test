package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"path/filepath"

	"encoding/json"
	"errors"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

const (
	addr = ":8080"
	//readTimeout  = time.Second * 5
	//writeTimeout = time.Second * 10
	bufferSize = 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  bufferSize,
	WriteBufferSize: bufferSize,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type game struct {
	conn     *websocket.Conn
	d        Dimensions
	stats    []ClickStats
	name     string
	testType string
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
	TimeTaken  float64 `json:"timeTaken"`
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("args")
	}

	r := httprouter.New()

	r.GET("/", Index)
	r.GET("/ws", Game)

	s := &http.Server{
		Addr: addr,
		//ReadTimeout:  readTimeout,
		//WriteTimeout: writeTimeout,
		Handler: r,
	}
	log.Fatal(s.ListenAndServe())
}

// NewCircle creates a new random circle.
func NewCircle(d Dimensions) Circle {
	size := randomInt(20, 120)

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

func (g game) sendCircle() error {
	c := NewCircle(g.d)
	return g.conn.WriteJSON(&c)
}

func (g game) save() error {
	f, err := os.Create(filepath.Join("./tests/" + fmt.Sprintf("%s-%s.json", g.name, g.testType)))
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(&g.stats)
	if err != nil {
		return err
	}
	n, err := f.Write(b)
	if err != nil {
		return err
	}
	if len(b) > 0 && n == 0 {
		return errors.New("146????")
	}
	return nil
}

func (g game) start() {
	count := 0

	// Send the initial circle.
	if err := g.sendCircle(); err != nil {
		log.Fatal(err)
	}

	for {
		if count == 20 {
			if err := g.save(); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Saving in ./tests")
			break
		}

		// Receive the click stats for the given circle.
		var cs ClickStats
		if err := g.conn.ReadJSON(&cs); err != nil {
			log.Fatal(err)
		}
		g.stats = append(g.stats, cs)
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

	g := game{conn: conn, d: d, name: os.Args[1], testType: os.Args[2]}
	g.start()
}

// Index ...
func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t, err := template.ParseFiles(filepath.Join(wd, "/index.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}
