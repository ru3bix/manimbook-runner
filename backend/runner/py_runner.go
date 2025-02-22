package runner

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"context"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/sync/semaphore"
	"log"
)

func sendMessage(c *websocket.Conn, message interface{}, mutex *sync.Mutex) {
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

func closeConnection(c *websocket.Conn, chapter string, mutex *sync.Mutex) {
	defer c.Close()
	endingMsg := map[string]interface{}{
		"type":    "FinalMessage",
		"payload": "bye :)",
		"chapter": chapter,
	}
	sendMessage(c, endingMsg, mutex)
}

func ExecCell(c *websocket.Conn, info ClassInfo, mutex *sync.Mutex, publicHTTPUrl string) {
	log.Println("Executing class ", info.ClassName)
	initiateMsg := map[string]interface{}{
		"type":    "InitiateMessage",
		"payload": info.Index,
		"chapter": info.Chapter,
	}
	sendMessage(c, initiateMsg, mutex)

	mediaDir := fmt.Sprintf("%s/%s/media", info.OutputDir, info.ClassName)
	cmd := exec.Command("manim",
		"--media_dir",
		mediaDir,
		"--disable_caching",
		"--flush_cache",
		"-v", "INFO",
		"--progress_bar", "none",
		"-qm",
		info.FilePath,
		info.ClassName,
	)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("failed at stdout pipe")
		closeConnection(c, info.Chapter, mutex)
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Println("failed at stderr pipe")
		closeConnection(c, info.Chapter, mutex)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Println("failed at command start")
		closeConnection(c, info.Chapter, mutex)
		return
	}

	s := bufio.NewScanner(io.MultiReader(stdoutPipe, stderrPipe))
	for s.Scan() {
		logMsg := map[string]interface{}{
			"type":    "LogMessage",
			"payload": []interface{}{info.Index, s.Text()},
			"chapter": info.Chapter,
		}
		sendMessage(c, logMsg, mutex)
	}

	if err := cmd.Wait(); err != nil {
		log.Println("failed at command wait")
		closeConnection(c, info.Chapter, mutex)
		return
	}

	videoUrl := fmt.Sprintf("%s/data/%s/%s/media/videos/output/720p30/%s.mp4", publicHTTPUrl, info.Chapter, info.ClassName, info.ClassName)
	completionMsg := map[string]interface{}{
		"type":    "CompletionMessage",
		"payload": []interface{}{info.Index, videoUrl},
		"chapter": info.Chapter,
	}
	sendMessage(c, completionMsg, mutex)
}

func ExecChapter(c *websocket.Conn, classes []ClassInfo, ctx context.Context, publicHTTPUrl string) error {
	var (
		wsMutex    sync.Mutex
		maxWorkers = 5
		sem        = semaphore.NewWeighted(int64(maxWorkers))
	)
	log.Println("Received classes:", classes)

	startMsg := map[string]interface{}{
		"type":    "BeginningMessage",
		"payload": len(classes),
		"chapter": classes[0].Chapter,
	}
	sendMessage(c, startMsg, &wsMutex)

	for _, class := range classes {
		if err := sem.Acquire(ctx, 1); err != nil {
			return nil
		}
		go func() {
			defer sem.Release(1)
			ExecCell(c, class, &wsMutex, publicHTTPUrl)
		}()
	}
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
		closeConnection(c, classes[0].Chapter, &wsMutex)
		return err
	}

	endingMsg := map[string]interface{}{
		"type":    "FinalMessage",
		"payload": "bye :)",
		"chapter": classes[0].Chapter,
	}
	sendMessage(c, endingMsg, &wsMutex)

	return nil
}
