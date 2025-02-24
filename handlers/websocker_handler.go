package handlers

import (
	"log"
	"mbook-backend/runner"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

// Configure the WebSocket upgrader with an origin checker.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GenerateWSHandler(chapters map[string]([]runner.ClassInfo), publicHTTPUrl string) func(c *gin.Context) {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot upgrade connection"})
			return
		}
		defer conn.Close()

		log.Println("Client connected.")

		_, handshakeData, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading handshake message:", err)
			return
		}

		// Decode the handshake message.
		var handshake map[string]interface{}
		if err := msgpack.Unmarshal(handshakeData, &handshake); err != nil {
			log.Println("Error decoding handshake:", err)
			return
		}

		chapterValue, ok := handshake["chapter"]
		if !ok {
			log.Println("No chapter value provided by client. Closing connection.")
			return
		}
		log.Println("Received chapter value:", chapterValue)
		classes := chapters[chapterValue.(string)]

		if len(classes) == 0 {
			log.Println("No such chapter exists yet. Closing connecton.")
			return
		}

		runner.ExecChapter(conn, classes, c, publicHTTPUrl)
	}
}
