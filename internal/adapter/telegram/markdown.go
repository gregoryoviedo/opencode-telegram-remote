package telegram

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	codeBlockRE = regexp.MustCompile("(?s)```([\\s\\S]*?)```")
	codeSpanRE  = regexp.MustCompile("`+([^`\\n]+?)`+")
	linkRE      = regexp.MustCompile(`\[([^\]\n]+)\]\(([^\s\)]+)\)`)
	headerRE    = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`)
	boldRE      = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	italicRE    = regexp.MustCompile(`\*([^*\s][^*\n]*?[^*\s]|[^*\s])\*`)
	bulletRE    = regexp.MustCompile(`(?m)^[\-\*]\s+(.+)$`)
)

const (
	codeBlockPlaceholder = "\x00CB"
	codeSpanPlaceholder  = "\x00IC"
	headerPlaceholder    = "\x00H"
	boldPlaceholder      = "\x00B"
	italicPlaceholder    = "\x00I"
)

// markdownToTelegramHTML converts a Markdown string into the subset of HTML
// that the Telegram Bot API accepts (parse_mode=HTML): <b>, <i>, <u>, <s>,
// <del>, <a href="">, <code>, <pre>. Anything else is escaped or stripped so
// the message renders correctly on every Telegram client (iOS included).
func markdownToTelegramHTML(text string) string {
	if text == "" {
		return ""
	}

	var (
		codeBlocks  []string
		inlineCodes []string
		bolds       []string
		italics     []string
	)

	text = codeBlockRE.ReplaceAllStringFunc(text, func(match string) string {
		inner := strings.TrimPrefix(match, "```")
		inner = strings.TrimSuffix(inner, "```")
		inner = strings.TrimRight(inner, "\n")
		if strings.HasPrefix(inner, "\n") {
			inner = inner[1:]
		}
		if idx := strings.Index(inner, "\n"); idx > 0 && idx < 32 && !strings.Contains(inner[:idx], "\n") {
			inner = inner[idx+1:]
		}
		idx := len(codeBlocks)
		codeBlocks = append(codeBlocks, "<pre>"+escapeHTML(inner)+"</pre>")
		return codeBlockPlaceholder + strconv.Itoa(idx)
	})

	text = codeSpanRE.ReplaceAllStringFunc(text, func(match string) string {
		inner := match
		for len(inner) > 0 && inner[0] == '`' {
			inner = inner[1:]
		}
		for len(inner) > 0 && inner[len(inner)-1] == '`' {
			inner = inner[:len(inner)-1]
		}
		idx := len(inlineCodes)
		inlineCodes = append(inlineCodes, "<code>"+escapeHTML(inner)+"</code>")
		return codeSpanPlaceholder + strconv.Itoa(idx)
	})

	text = escapeHTML(text)

	text = linkRE.ReplaceAllStringFunc(text, func(match string) string {
		m := linkRE.FindStringSubmatch(match)
		if len(m) != 3 {
			return match
		}
		label := m[1]
		url := m[2]
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "tg://") {
			return match
		}
		return `<a href="` + url + `">` + label + `</a>`
	})

	text = headerRE.ReplaceAllStringFunc(text, func(match string) string {
		m := headerRE.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		idx := len(bolds)
		bolds = append(bolds, m[1])
		return headerPlaceholder + strconv.Itoa(idx) + "\n"
	})

	text = boldRE.ReplaceAllStringFunc(text, func(match string) string {
		m := boldRE.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		idx := len(bolds)
		bolds = append(bolds, m[1])
		return boldPlaceholder + strconv.Itoa(idx)
	})

	text = italicRE.ReplaceAllStringFunc(text, func(match string) string {
		m := italicRE.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		idx := len(italics)
		italics = append(italics, m[1])
		return italicPlaceholder + strconv.Itoa(idx)
	})

	text = bulletRE.ReplaceAllString(text, "• $1")

	for i, code := range inlineCodes {
		text = strings.ReplaceAll(text, codeSpanPlaceholder+strconv.Itoa(i), code)
	}
	for i, block := range codeBlocks {
		text = strings.ReplaceAll(text, codeBlockPlaceholder+strconv.Itoa(i), block)
	}
	for i, inner := range italics {
		text = strings.ReplaceAll(text, italicPlaceholder+strconv.Itoa(i), "<i>"+inner+"</i>")
	}
	for i, inner := range bolds {
		text = strings.ReplaceAll(text, boldPlaceholder+strconv.Itoa(i), "<b>"+inner+"</b>")
		text = strings.ReplaceAll(text, headerPlaceholder+strconv.Itoa(i), "<b>"+inner+"</b>")
	}

	return text
}

func escapeHTML(s string) string {
	if !strings.ContainsAny(s, "&<>\"") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
