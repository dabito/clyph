package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type cssBlock struct {
	selector string
	body     string
}

func loadSource(source string) (string, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source) // #nosec G107 - intentional fetch for update
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeCSSEscape(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); {
		if value[i] != '\\' {
			out.WriteByte(value[i])
			i++
			continue
		}
		i++
		if i >= len(value) {
			break
		}
		hexStart := i
		for i < len(value) && i-hexStart < 6 {
			c := value[i]
			if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
				i++
				continue
			}
			break
		}
		if i > hexStart {
			v, err := strconv.ParseInt(value[hexStart:i], 16, 32)
			if err == nil {
				out.WriteRune(rune(v))
				if i < len(value) && isSpace(value[i]) {
					i++
				}
				continue
			}
		}
		out.WriteByte(value[i])
		i++
	}
	return out.String()
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// parseCSSCatalog parses Nerd Fonts webfont.css into glyph records.
// When a CSS content value contains multiple runes, only the first rune's
// codepoint is used (multi-rune content collapses to the first rune).
func parseCSSCatalog(text string) ([]Record, error) {
	text = stripComments(text)
	records := map[string]Record{}
	for _, block := range findBlocks(text) {
		content, ok := extractContent(block.body)
		if !ok {
			continue
		}
		decoded := decodeCSSEscape(content)
		if decoded == "" {
			continue
		}
		codepointHex := fmt.Sprintf("%x", []rune(decoded)[0])
		aliases := extractClasses(block.selector)
		if len(aliases) == 0 {
			continue
		}
		for _, name := range aliases {
			rec, err := recordFromParts(name, codepointHex, nil)
			if err != nil {
				return nil, err
			}
			records[name] = rec
		}
	}
	out := make([]Record, 0, len(records))
	for _, rec := range records {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func stripComments(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		if i+1 < len(text) && text[i] == '/' && text[i+1] == '*' {
			i += 2
			for i+1 < len(text) && !(text[i] == '*' && text[i+1] == '/') {
				i++
			}
			if i+1 < len(text) {
				i += 2
			}
			continue
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

func findBlocks(text string) []cssBlock {
	var blocks []cssBlock
	for {
		open := strings.IndexByte(text, '{')
		if open < 0 {
			break
		}
		close := strings.IndexByte(text[open+1:], '}')
		if close < 0 {
			break
		}
		close += open + 1
		selector := strings.TrimSpace(text[:open])
		body := text[open+1 : close]
		blocks = append(blocks, cssBlock{selector: selector, body: body})
		text = text[close+1:]
	}
	return blocks
}

func extractContent(body string) (string, bool) {
	idx := strings.Index(body, "content")
	if idx < 0 {
		return "", false
	}
	rest := body[idx+len("content"):]
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return "", false
	}
	rest = rest[colon+1:]
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 {
		return "", false
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return "", false
	}
	rest = rest[1:]
	end := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == quote && (i == 0 || rest[i-1] != '\\') {
			end = i
			break
		}
	}
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

func extractClasses(selector string) []string {
	parts := strings.Split(selector, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, ".") {
			part = part[1:]
		}
		for i := 0; i < len(part); i++ {
			if part[i] == ':' {
				part = part[:i]
				break
			}
		}
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}
