package lib

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func isSupOrSubOpenTag(tag string) bool {
	for _, prefix := range []string{"<sup", "<sub"} {
		if strings.HasPrefix(tag, prefix) && len(tag) > len(prefix) {
			next := tag[len(prefix)]
			if next == '>' || next == ' ' || next == '\t' || next == '\n' || next == '\r' {
				return !strings.HasSuffix(tag, "/>")
			}
		}
	}
	return false
}

func isSupOrSubCloseTag(tag string) bool {
	for _, prefix := range []string{"</sup", "</sub"} {
		if strings.HasPrefix(tag, prefix) && len(tag) > len(prefix) {
			next := tag[len(prefix)]
			if next == '>' || next == ' ' || next == '\t' || next == '\n' || next == '\r' {
				return true
			}
		}
	}
	return false
}

// MarkdownToPlainText는 Markdown 바이트 슬라이스를 받아 순수 텍스트 문자열로 변환합니다.
func MarkdownToPlainText(markdownContent []byte) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(MarkExtension, CalloutExtension),
	)

	reader := text.NewReader(markdownContent)
	pc := parser.NewContext()
	doc := md.Parser().Parse(reader, parser.WithContext(pc))

	var plainTextBuf bytes.Buffer
	var ignoredTagDepth int

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch node := n.(type) {
			case *ast.RawHTML:
				raw := strings.ToLower(strings.TrimSpace(string(node.Segments.Value(markdownContent))))
				if isSupOrSubOpenTag(raw) {
					ignoredTagDepth++
				} else if isSupOrSubCloseTag(raw) && ignoredTagDepth > 0 {
					ignoredTagDepth--
				}
			case *ast.Text:
				if ignoredTagDepth == 0 {
					plainTextBuf.Write(node.Segment.Value(markdownContent))
				}

				if node.SoftLineBreak() || node.HardLineBreak() {
					plainTextBuf.WriteString(" ")
				}
			case *ast.String:
				if ignoredTagDepth == 0 {
					plainTextBuf.Write(node.Value)
				}
			}
		}

		if !entering && n.Type() == ast.TypeBlock {
			ignoredTagDepth = 0
			plainTextBuf.WriteString(" ")
		}
		return ast.WalkContinue, nil
	})

	return plainTextBuf.String(), nil
}
