package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/johnnylee/goutil/fileutil"
	"github.com/johnnylee/goutil/jsonutil"
	"github.com/johnnylee/goutil/logutil"
)

type Config struct {
	RootName     string
	ThumbWidth   int
	ThumbQuality int
}

var imgExtensions = map[string]bool{
	".jpg": true,
	".png": true,
}

var log = logutil.New("jlstatic")
var tmpl = template.Must(template.ParseFiles("template.html"))

var conf Config
var srcDir string
var buildDir string

type Breadcrumb struct {
	FullPath string
	Name     string
}

type Context struct {
	Content     template.HTML
	Breadcrumbs []Breadcrumb
}

func NewContext(srcPath string) (Context, error) {
	ctx := Context{}

	relPath, err := filepath.Rel(srcDir, srcPath)
	if err != nil {
		return ctx, err
	}

	inputBuf, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return ctx, err
	}

	ctx.Content = template.HTML(Markdown(inputBuf))
	ctx.Breadcrumbs = []Breadcrumb{
		Breadcrumb{"/", conf.RootName},
	}

	parents := strings.Split(filepath.Dir(relPath), "/")
	path := "/"

	for _, p := range parents {
		if p == "." || len(p) == 0 {
			continue
		}

		path = filepath.Join(path, p) + "/"
		ctx.Breadcrumbs = append(ctx.Breadcrumbs, Breadcrumb{path, p})
	}

	return ctx, nil
}

func main() {
	if err := jsonutil.Load(&conf, "config.json"); err != nil {
		log.Err(err, "When reading config.json.")
		return
	}

	srcDir, _ = filepath.Abs("src")
	buildDir, _ = filepath.Abs("build")

	// Walk files, generating content:
	//
	// * For index.md files, create a index.html file.
	// * For images (jpg, png), copy the image and produce a thumbnail.
	// * Any other files are simply coppied.
	walk := func(srcPath string, info os.FileInfo, err error) error {
		srcPath, err = filepath.Abs(srcPath)
		if err != nil {
			log.Err(err, "When getting absolute source path: %v", srcPath)
			return err
		}

		// Skip directories.
		if info.IsDir() {
			log.Msg("Skipping directory: %v", srcPath)
			return nil
		}

		log.Msg("Walking path: %v", srcPath)

		// Get output directory and filename.
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			log.Err(err, "When walking path: %v", srcPath)
			return err
		}

		dstPath := filepath.Join(buildDir, relPath)
		outDir, outFile := filepath.Split(dstPath)

		log.Msg("  Output path: %v", dstPath)

		// Create the destination directory.
		if err := os.MkdirAll(outDir, 0700); err != nil {
			log.Err(err, "When creating output directory %v", outDir)
			return err
		}

		// Is this an index.md file?
		if outFile == "index.md" {
			outFile = "index.html"
			dstPath = filepath.Join(outDir, outFile)

			ctx, err := NewContext(srcPath)
			if err != nil {
				log.Err(err, "When creating context for %v", srcPath)
				return err
			}

			dstFile, err := os.Create(dstPath)
			if err != nil {
				log.Err(err, "When creating file %v", dstPath)
				return err
			}
			defer dstFile.Close()

			if err := tmpl.Execute(dstFile, ctx); err != nil {
				log.Err(err, "When executing template")
				return err
			}

			return nil
		}

		// Skip non-MD files if they exist.
		_, isImage := imgExtensions[filepath.Ext(outFile)]
		if isImage && fileutil.FileExists(dstPath) {
			log.Msg("  Skipping existing image.")
			return nil
		}

		// Copy all other files.
		if err := copyFile(srcPath, dstPath); err != nil {
			log.Err(err, "When copying file: %v", srcPath)
			return err
		}

		// If this isn't an image, we're done.
		if !isImage {
			return nil
		}

		// Create the thumbnail directory.
		thumbDir := filepath.Join(outDir, "t")
		log.Msg("  Creating thumbnail directory: %v", thumbDir)
		if err := os.MkdirAll(thumbDir, 0700); err != nil {
			log.Err(err, "When creating thumbnail directory")
			return err
		}

		thumbFile := filepath.Join(thumbDir, outFile)
		log.Msg("  Creating thumbnail: %v", thumbFile)

		// Use imagemagick to create the thumbnail.
		shellCmd := fmt.Sprintf("convert "+
			"-strip "+
			"-interlace Plane "+
			"-quality %v%% "+
			"-thumbnail %v %v %v",
			conf.ThumbQuality, conf.ThumbWidth, dstPath, thumbFile)

		cmd := exec.Command("bash", "-c", shellCmd)
		if err := cmd.Run(); err != nil {
			log.Err(err, "When creating image thumbnail for %v", dstPath)
			return err
		}

		return nil
	}

	if err := filepath.Walk("src", walk); err != nil {
		log.Err(err, "Walk failed.")
	}
}

func processMarkdown(relPath, srcPath, dstPath string) error {
	return nil
}

func copyFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
