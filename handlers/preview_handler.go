package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"mbook-backend/runner/preview"
	"net/http"
	"os"
)

func GetPreviewHandler(baseDir, publicWSUrl, publicHTTPUrl string, chapters map[string]([]preview.ClassInfo)) func(c *gin.Context) {
	return func(c *gin.Context) {
		chapterName := c.Param("chapterName")
		outputDir := fmt.Sprintf("%s/data/%s", baseDir, chapterName)
		if stat, err := os.Stat(outputDir); err == nil && stat.IsDir() {
			err := os.RemoveAll(outputDir)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not remove directory"})
				return
			}
		}
		templateDir := fmt.Sprintf("%s/%s", baseDir, "/templates")

		file, _ := c.FormFile("file")

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
			return
		}
		defer f.Close()

		config := preview.Config{
			OutputDir:    outputDir,
			TemplateDir:  templateDir,
			WebsocketURL: fmt.Sprintf("%s/preview/ws", publicWSUrl),
			ChapterName:  chapterName,
		}
		chapters[chapterName], err = preview.PrepareChapterPreviewDir(f, config)

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
	}
}

func GetClearPreviewHandler(baseDir string) func(c *gin.Context) {
	return func(c *gin.Context) {
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
	}
}
