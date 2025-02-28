package preview

import (
	"bufio"
	"fmt"
	"io"
	"mbook-backend/utils"
	"os/exec"
	"sync"

	"context"
	"log"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/semaphore"
)

func execCell(c *websocket.Conn, info ClassInfo, mutex *sync.Mutex, publicHTTPUrl string) {
	log.Println("Executing class ", info.ClassName)
	initiateMsg := map[string]interface{}{
		"type":    "InitiateMessage",
		"payload": info.Index,
		"chapter": info.Chapter,
	}
	utils.SendWsMessage(c, initiateMsg, mutex)

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
		utils.CloseWSConnection(c, info.Chapter, mutex)
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Println("failed at stderr pipe")
		utils.CloseWSConnection(c, info.Chapter, mutex)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Println("failed at command start")
		utils.CloseWSConnection(c, info.Chapter, mutex)
		return
	}

	s := bufio.NewScanner(io.MultiReader(stdoutPipe, stderrPipe))
	for s.Scan() {
		logMsg := map[string]interface{}{
			"type":    "LogMessage",
			"payload": []interface{}{info.Index, s.Text()},
			"chapter": info.Chapter,
		}
		utils.SendWsMessage(c, logMsg, mutex)
	}

	if err := cmd.Wait(); err != nil {
		log.Println("failed at command wait")
		utils.CloseWSConnection(c, info.Chapter, mutex)
		return
	}

	videoUrl := fmt.Sprintf("%s/data/%s/%s/media/videos/output/720p30/%s.mp4", publicHTTPUrl, info.Chapter, info.ClassName, info.ClassName)
	completionMsg := map[string]interface{}{
		"type":    "CompletionMessage",
		"payload": []interface{}{info.Index, videoUrl},
		"chapter": info.Chapter,
	}
	utils.SendWsMessage(c, completionMsg, mutex)
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
	utils.SendWsMessage(c, startMsg, &wsMutex)

	for _, class := range classes {
		if err := sem.Acquire(ctx, 1); err != nil {
			return nil
		}
		go func() {
			defer sem.Release(1)
			execCell(c, class, &wsMutex, publicHTTPUrl)
		}()
	}
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
		utils.CloseWSConnection(c, classes[0].Chapter, &wsMutex)
		return err
	}

	endingMsg := map[string]interface{}{
		"type":    "FinalMessage",
		"payload": "bye :)",
		"chapter": classes[0].Chapter,
	}
	utils.SendWsMessage(c, endingMsg, &wsMutex)

	return nil
}
