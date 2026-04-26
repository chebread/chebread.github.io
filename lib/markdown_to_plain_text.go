package lib

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// MarkdownToPlainText는 Markdown 바이트 슬라이스를 받아 순수 텍스트 문자열로 변환합니다.
func MarkdownToPlainText(markdownContent []byte) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(MarkExtension),
	)

	reader := text.NewReader(markdownContent)
	pc := parser.NewContext()
	doc := md.Parser().Parse(reader, parser.WithContext(pc))

	var plainTextBuf bytes.Buffer
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch node := n.(type) {
			case *ast.Text:
				plainTextBuf.Write(node.Segment.Value(markdownContent))

				if node.SoftLineBreak() || node.HardLineBreak() {
					plainTextBuf.WriteString(" ")
				}
			case *ast.String:
				plainTextBuf.Write(node.Value)
			}
		}

		if !entering && n.Type() == ast.TypeBlock {
			plainTextBuf.WriteString(" ")
		}
		return ast.WalkContinue, nil
	})

	return plainTextBuf.String(), nil
}
