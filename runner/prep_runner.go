package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

type Manimbook struct {
	Version  string                 `json:"version"`
	Metadata map[string]interface{} `json:"metadata"`
	Cells    []Cell                 `json:"cells"`
}

type Cell struct {
	CellType string   `json:"cell_type"`
	Source   []string `json:"source"`
	// Additional fields can be added as needed.
}

// ClassInfo holds a found python class name and the index of the cell where it was found.
type ClassInfo struct {
	Chapter   string
	FilePath  string
	ClassName string
	Index     int
	OutputDir string
}

type Config struct {
	OutputDir    string
	TemplateDir  string
	WebsocketURL string
	ChapterName  string
}

func PrepareManimbookDir(manimbookReader io.Reader, config Config) ([]ClassInfo, error) {
	var manimbook Manimbook
	var HTMLBuf bytes.Buffer
	var PythonBuf bytes.Buffer
	var PythonClasses []ClassInfo

	err := os.Mkdir(config.OutputDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(manimbookReader)
	err = decoder.Decode(&manimbook)
	if err != nil {
		return nil, err
	}

	// Create or open the Python file to save all code snippets
	pythonOutputFilePath := fmt.Sprintf("%s/output.py", config.OutputDir)
	pythonOutputFile, err := os.Create(pythonOutputFilePath)
	if err != nil {
		return nil, err
	}
	defer pythonOutputFile.Close()

	htmlOutputFilePath := fmt.Sprintf("%s/index.html", config.OutputDir)
	htmlOutputFile, err := os.Create(htmlOutputFilePath)
	if err != nil {
		return nil, err
	}
	defer htmlOutputFile.Close()

	for index, cell := range manimbook.Cells {
		if cell.CellType == "code" {
			code := strings.Join(cell.Source, "\n")
			_, err := PythonBuf.WriteString(code + "\n\n")
			if err != nil {
				return nil, err
			}
			// Extract the first class name (if any)
			className := extractClassName(code)
			if className != "" {
				PythonClasses = append(PythonClasses, ClassInfo{ClassName: className, Index: index, Chapter: config.ChapterName, FilePath: pythonOutputFilePath, OutputDir: config.OutputDir})
			}
			video_tag := fmt.Sprintf("<video autoplay loop muted poster class=\"hidden\" id=\"%d\"></video>\n", index)
			//			placeholder_tag := `
			//				<div id="placeholder-%d" class="placeholder shimmer">
			//				<div class="faux-image-wrapper">
			//				<div class="faux-image"></div>
			//				</div>
			//				</div>
			//			`
			//			HTMLBuf.WriteString(fmt.Sprintf(placeholder_tag, index))
			HTMLBuf.WriteString(video_tag)
		} else if cell.CellType == "markdown" {
			markdown := strings.Join(cell.Source, "\n")
			err := appendRenderedMarkdown(&HTMLBuf, markdown)
			if err != nil {
				return nil, err
			}
		}

	}
	htmlTemplateFile := fmt.Sprintf("%s/html.tmpl", config.TemplateDir)
	pythonTemplateFile := fmt.Sprintf("%s/py.tmpl", config.TemplateDir)
	HTMLData := struct {
		WebsocketURL string
		HTMLContent  string
		ChapterName  string
	}{
		WebsocketURL: config.WebsocketURL,
		HTMLContent:  HTMLBuf.String(),
		ChapterName:  config.ChapterName,
	}
	PythonData := struct {
		Code string
	}{
		Code: PythonBuf.String(),
	}
	htmlTemplate, err := template.ParseFiles(htmlTemplateFile)
	if err != nil {
		return nil, err
	}
	pythonTemplate, err := template.ParseFiles(pythonTemplateFile)
	if err != nil {
		return nil, err
	}
	err = htmlTemplate.Execute(htmlOutputFile, HTMLData)
	if err != nil {
		return nil, err
	}
	err = pythonTemplate.Execute(pythonOutputFile, PythonData)
	if err != nil {
		return nil, err
	}
	return PythonClasses, nil
}

func appendRenderedMarkdown(src io.Writer, markdown string) error {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
	)
	return md.Convert([]byte(markdown), src)
}

func extractClassName(code string) string {
	re := regexp.MustCompile(`class\s+(\w+)\s*\([^)]*Scene[^)]*\)\s*:`)
	matches := re.FindStringSubmatch(code)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
