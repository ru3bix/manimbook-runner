package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"mbook-backend/handlers"
	"mbook-backend/runner/preview"
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
	chapters := make(map[string]([]preview.ClassInfo))

	router.Static("/data", fmt.Sprintf("%s/data", baseDir))

	router.GET("/preview/ws", handlers.GetWSHandler(chapters, publicHTTPUrl))

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.POST("/preview/:chapterName", handlers.GetPreviewHandler(baseDir, publicWSUrl, publicHTTPUrl, chapters))

	router.DELETE("/clear/:chapterName", handlers.GetClearPreviewHandler(baseDir))
	router.Run(":8080")
}
