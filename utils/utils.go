package utils

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"io"
	"log"
	"regexp"
	"sync"
)

type ManimbookChapter struct {
	Metadata map[string]interface{} `json:"metadata"`
	Cells    []Cell                 `json:"cells"`
}

type Cell struct {
	CellType string   `json:"cell_type"`
	Source   []string `json:"source"`
	// Additional fields can be added as needed.
}

func AppendRenderedMarkdown(src io.Writer, markdown string) error {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
	)
	return md.Convert([]byte(markdown), src)
}

func ExtractClassName(code string) string {
	re := regexp.MustCompile(`class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:`)
	matches := re.FindStringSubmatch(code)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func SendWsMessage(c *websocket.Conn, message interface{}, mutex *sync.Mutex) {
	data, err := msgpack.Marshal(message)
	if err != nil {
		log.Println("failed at sendMessage due to msgpack", message)
		return
	}
	mutex.Lock()
	defer mutex.Unlock()

	err = c.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Println("failed to send message", message)
	}
}

func CloseWSConnection(c *websocket.Conn, chapter string, mutex *sync.Mutex) {
	defer c.Close()
	endingMsg := map[string]interface{}{
		"type":    "FinalMessage",
		"payload": "bye :)",
		"chapter": chapter,
	}
	SendWsMessage(c, endingMsg, mutex)
}

func DecodeManimbookJSON(manimbook io.Reader, chapter *ManimbookChapter) error {
	decoder := json.NewDecoder(manimbook)
	return decoder.Decode(chapter)
}
