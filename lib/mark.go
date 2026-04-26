package lib // main에 넣을 경우 package main

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// AST 노드 정의
type Mark struct {
	ast.BaseInline
}

var KindMark = ast.NewNodeKind("Mark")

func (n *Mark) Kind() ast.NodeKind {
	return KindMark
}

func (n *Mark) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type markDelimiterProcessor struct{}

var defaultMarkDelimiterProcessor = &markDelimiterProcessor{}

func (p *markDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '='
}

func (p *markDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *markDelimiterProcessor) OnMatch(consumes int) ast.Node {
	return &Mark{}
}

type markParser struct{}

func (p *markParser) Trigger() []byte {
	return []byte{'='}
}

func (p *markParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, defaultMarkDelimiterProcessor)
	if node == nil {
		return nil
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

// HTML 렌더러 정의
type markHTMLRenderer struct{}

func (r *markHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindMark, r.renderMark)
}

func (r *markHTMLRenderer) renderMark(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString("<mark>")
	} else {
		w.WriteString("</mark>")
	}
	return ast.WalkContinue, nil
}

// Extender 정의
type markExtension struct{}

func (e *markExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(&markParser{}, 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&markHTMLRenderer{}, 500),
		),
	)
}

var MarkExtension = &markExtension{}
