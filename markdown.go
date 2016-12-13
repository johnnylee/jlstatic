package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"

	"github.com/russross/blackfriday"
)

const commonHtmlFlags = 0 |
	blackfriday.HTML_USE_SMARTYPANTS |
	blackfriday.HTML_SMARTYPANTS_LATEX_DASHES |
	blackfriday.HTML_FOOTNOTE_RETURN_LINKS

const commonExtensions = 0 |
	blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_HEADER_IDS |
	blackfriday.EXTENSION_FOOTNOTES |
	blackfriday.EXTENSION_DEFINITION_LISTS

type Renderer struct {
	*blackfriday.Html
}

var renderer = &Renderer{
	Html: blackfriday.HtmlRenderer(
		commonHtmlFlags, "", "").(*blackfriday.Html),
}

// BlockCode: Render code blocks using `pygmentize` syntax highlighter.
func (r *Renderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {

	// If the language isn't specified, fallback on the standard processor.
	if len(lang) == 0 {
		r.Html.BlockCode(out, text, lang)
		return
	}

	var stderr bytes.Buffer

	cmd := exec.Command("pygmentize", "-l"+lang, "-fhtml")
	cmd.Stdin = bytes.NewReader(text)
	cmd.Stdout = out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Err(err, "When running pygmentize")
	}
}

// Coppied from BlackFriday for custom Image renderer:
func escapeSingleChar(char byte) (string, bool) {
	if char == '"' {
		return "&quot;", true
	}
	if char == '&' {
		return "&amp;", true
	}
	if char == '<' {
		return "&lt;", true
	}
	if char == '>' {
		return "&gt;", true
	}
	return "", false
}

// Coppied from BlackFriday for custom Image renderer:
func attrEscape(out *bytes.Buffer, src []byte) {
	org := 0
	for i, ch := range src {
		if entity, ok := escapeSingleChar(ch); ok {
			if i > org {
				// copy all the normal characters since the last escape
				out.Write(src[org:i])
			}
			org = i + 1
			out.WriteString(entity)
		}
	}
	if org < len(src) {
		out.Write(src[org:])
	}
}

func (r *Renderer) Image(out *bytes.Buffer, link, title, alt []byte) {
	URL, err := url.Parse(string(link))
	if err != nil {
		r.Html.Image(out, link, title, alt)
		return
	}

	class := URL.Query().Get("c")

	out.WriteString(`<img src="`)
	attrEscape(out, link)
	out.WriteString(`" alt="`)
	if len(alt) > 0 {
		attrEscape(out, alt)
	}
	if len(title) > 0 {
		out.WriteString(`" title="`)
		attrEscape(out, title)
	}

	// Write the image size class.
	out.WriteString(fmt.Sprintf(`" class="%v">`, class))
}

func Markdown(input []byte) []byte {
	return blackfriday.Markdown(input, renderer, commonExtensions)
}
