package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"mbook-backend/handlers"
	"mbook-backend/runner"
	"net/http"
	"os"
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func main() {
	godotenv.Load()

	baseDir := getenv("BASE_DIR", "/d/projects/mgobook")
	publicHTTPUrl := getenv("PUBLIC_HTTP_URL", "http://localhost:8080")
	publicWSUrl := getenv("PUBLIC_WS_URL", "ws://localhost:8080")

	router := gin.Default()
	chapters := make(map[string]([]runner.ClassInfo))

	router.Static("/data", fmt.Sprintf("%s/data", baseDir))

	router.GET("/ws", handlers.GenerateWSHandler(chapters, publicHTTPUrl))

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.POST("/render/:chapterName", func(c *gin.Context) {
		chapterName := c.Param("chapterName")
		outputDir := fmt.Sprintf("%s/data/%s", baseDir, chapterName)
		if stat, err := os.Stat(outputDir); err == nil && stat.IsDir() {
			c.JSON(http.StatusConflict, gin.H{"error": "output directory exists"})
			return
		}
		templateDir := fmt.Sprintf("%s/%s", baseDir, "/templates")

		file, _ := c.FormFile("file")

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
			return
		}
		defer f.Close()

		config := runner.Config{
			OutputDir:    outputDir,
			TemplateDir:  templateDir,
			WebsocketURL: fmt.Sprintf("%s/ws", publicWSUrl),
			ChapterName:  chapterName,
		}
		chapters[chapterName], err = runner.PrepareManimbookDir(f, config)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare manimbook for rendering"})
			log.Println(err)
			return
		}

		res := map[string]interface{}{
			"chapter": chapterName,
			"url":     fmt.Sprintf("%s/data/%s", publicHTTPUrl, chapterName),
		}
		c.JSON(http.StatusOK, res)
		return
	})

	router.DELETE("/clear/:chapterName", func(c *gin.Context) {
		chapterName := c.Param("chapterName")
		targetDir := fmt.Sprintf("%s/data/%s", baseDir, chapterName)
		if stat, err := os.Stat(targetDir); err == nil && stat.IsDir() {
			err := os.RemoveAll(targetDir)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not remove directory"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"msg": "directory removed successfully"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "directory not found"})
		return
	})
	router.Run(":8080")
}
