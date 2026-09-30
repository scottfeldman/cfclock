package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	g "maragu.dev/gomponents"
)

//go:embed static/clock.css
var assets embed.FS

// Store is the program the controller will load onto the one Echo timer.
type Store struct {
	mu   sync.Mutex
	sess Session
}

func (s *Store) Get() Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sess.Clone()
}

func (s *Store) Update(fn func(*Session)) Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.sess)
	s.sess.Normalize()
	return s.sess.Clone()
}

func main() {
	sess := defaultSession()
	sess.Normalize()
	store := &Store{sess: sess}

	app := fiber.New()
	routes(app, store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Echo controller at http://localhost%s", addr)
	log.Fatal(app.Listen(addr))
}

func routes(app *fiber.App, store *Store) {
	app.Get("/", pageHandler(store))
	app.Post("/program", programHandler(store))
	app.Get("/face", faceHandler(store))
	app.Get("/static/clock.css", asset("text/css; charset=utf-8", "static/clock.css"))
}

func pageHandler(store *Store) fiber.Handler {
	return func(c fiber.Ctx) error {
		return writeNode(c, Page(store.Get()))
	}
}

func programHandler(store *Store) fiber.Handler {
	return func(c fiber.Ctx) error {
		sess := store.Update(func(s *Session) {
			s.Apply(c.FormValue("op"), c.FormValue("value"), c.FormValue("text"))
			s.Rev++
		})
		if c.Get("HX-Request") == "true" {
			return writeNode(c, Board(sess))
		}
		return writeNode(c, Page(sess))
	}
}

func faceHandler(store *Store) fiber.Handler {
	return func(c fiber.Ctx) error {
		now := time.Now()
		clientRev, _ := strconv.Atoi(c.Query("rev"))
		var ok, becameDone bool
		sess := store.Update(func(s *Session) {
			if clientRev != s.Rev {
				return
			}
			ok = true
			if s.CatchUp(now) {
				becameDone = true
				s.Rev++
			}
		})
		if !ok {
			return c.SendStatus(fiber.StatusNoContent)
		}
		if !becameDone {
			return writeNode(c, Face(sess, now))
		}
		sel := fmt.Sprintf("#transport[data-rev='%d']", clientRev)
		return writeNode(c, g.Group{Face(sess, now), Transport(sess, sel)})
	}
}

func asset(contentType, name string) fiber.Handler {
	return func(c fiber.Ctx) error {
		b, err := assets.ReadFile(name)
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("not found")
		}
		c.Set("Content-Type", contentType)
		c.Set("Cache-Control", "no-store")
		return c.Send(b)
	}
}

func writeNode(c fiber.Ctx, node g.Node) error {
	var buf bytes.Buffer
	if err := node.Render(&buf); err != nil {
		return fmt.Errorf("render: %w", err)
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-store")
	return c.Send(buf.Bytes())
}
