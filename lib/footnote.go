package lib

import (
	"strconv"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// FootnoteNumber is an inline AST node that renders a superscript footnote number prefix ("<sup>1</sup> ").
type FootnoteNumber struct {
	gast.BaseInline
	Index int
}

var KindFootnoteNumber = gast.NewNodeKind("FootnoteNumber")

func (n *FootnoteNumber) Kind() gast.NodeKind {
	return KindFootnoteNumber
}

func (n *FootnoteNumber) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// footnotePlaceholder marks the original document position of a parsed Footnote node
// before Goldmark's FootnoteBlockParser moves it into FootnoteList.
type footnotePlaceholder struct {
	gast.BaseBlock
	fn *extast.Footnote
}

var kindFootnotePlaceholder = gast.NewNodeKind("FootnotePlaceholder")

func (n *footnotePlaceholder) Kind() gast.NodeKind {
	return kindFootnotePlaceholder
}

func (n *footnotePlaceholder) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

type inPlaceFootnoteBlockParser struct {
	parser.BlockParser
}

func (b *inPlaceFootnoteBlockParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	if fn, ok := node.(*extast.Footnote); ok && node.Parent() != nil {
		placeholder := &footnotePlaceholder{fn: fn}
		node.Parent().InsertBefore(node.Parent(), node, placeholder)
	}
	b.BlockParser.Close(node, reader, pc)
}

type footnoteParagraphTransformer struct{}

func (t *footnoteParagraphTransformer) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	var placeholders []*footnotePlaceholder
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (astStatus gast.WalkStatus, err error) {
		if entering {
			if ph, ok := n.(*footnotePlaceholder); ok {
				placeholders = append(placeholders, ph)
				return gast.WalkSkipChildren, nil
			}
		}
		return gast.WalkContinue, nil
	})

	for _, ph := range placeholders {
		parent := ph.Parent()
		if parent == nil {
			continue
		}
		fn := ph.fn
		if fn == nil || fn.Index < 0 {
			parent.RemoveChild(parent, ph)
			continue
		}
		if fn.Parent() != nil {
			fn.Parent().RemoveChild(fn.Parent(), fn)
		}
		first := fn.FirstChild()
		if first != nil && first.Kind() == gast.KindParagraph {
			numNode := &FootnoteNumber{Index: fn.Index}
			first.InsertBefore(first, first.FirstChild(), numNode)
			if fn.ChildCount() == 1 {
				first.SetAttributeString("id", []byte("fn:"+strconv.Itoa(fn.Index)))
			}
		}
		parent.ReplaceChild(parent, ph, fn)
	}

	for child := doc.FirstChild(); child != nil; {
		next := child.NextSibling()
		if child.Kind() == extast.KindFootnoteList && child.ChildCount() == 0 {
			doc.RemoveChild(doc, child)
		}
		child = next
	}
}

type footnoteParagraphRenderer struct{}

func (r *footnoteParagraphRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extast.KindFootnoteList, r.renderFootnoteList)
	reg.Register(extast.KindFootnote, r.renderFootnote)
	reg.Register(KindFootnoteNumber, r.renderFootnoteNumber)
}

func (r *footnoteParagraphRenderer) renderFootnoteList(
	w util.BufWriter, source []byte, node gast.Node, entering bool,
) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

func (r *footnoteParagraphRenderer) renderFootnote(
	w util.BufWriter, source []byte, node gast.Node, entering bool,
) (gast.WalkStatus, error) {
	fn := node.(*extast.Footnote)
	isSingleParagraph := fn.ChildCount() == 1 && fn.FirstChild().Kind() == gast.KindParagraph
	if isSingleParagraph {
		return gast.WalkContinue, nil
	}

	if entering {
		_, _ = w.WriteString(`<div id="fn:` + strconv.Itoa(fn.Index) + `">` + "\n")
		if fn.FirstChild() == nil || fn.FirstChild().Kind() != gast.KindParagraph {
			_, _ = w.WriteString("<p><sup>" + strconv.Itoa(fn.Index) + "</sup></p>\n")
		}
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return gast.WalkContinue, nil
}

func (r *footnoteParagraphRenderer) renderFootnoteNumber(
	w util.BufWriter, source []byte, node gast.Node, entering bool,
) (gast.WalkStatus, error) {
	if entering {
		n := node.(*FootnoteNumber)
		_, _ = w.WriteString("<sup>" + strconv.Itoa(n.Index) + "</sup> ")
	}
	return gast.WalkContinue, nil
}

type footnoteExtension struct {
	backlinkHTML string
}

func (e *footnoteExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(&inPlaceFootnoteBlockParser{BlockParser: extension.NewFootnoteBlockParser()}, 999),
		),
		parser.WithInlineParsers(
			util.Prioritized(extension.NewFootnoteParser(), 101),
		),
		parser.WithASTTransformers(
			util.Prioritized(extension.NewFootnoteASTTransformer(), 999),
			util.Prioritized(&footnoteParagraphTransformer{}, 1000),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(extension.NewFootnoteHTMLRenderer(
				extension.WithFootnoteBacklinkHTML(e.backlinkHTML),
			), 500),
			util.Prioritized(&footnoteParagraphRenderer{}, 400),
		),
	)
}

// NewFootnoteExtension returns a Goldmark footnote extension that renders each footnote
// in-place where it is defined as a standard paragraph ("<p id=\"fn:1\"><sup>1</sup> ... [되돌아가기]</p>")
// instead of moving all footnotes into an <ol>/<li> list at the end of the document.
func NewFootnoteExtension(backlinkHTML string) goldmark.Extender {
	return &footnoteExtension{backlinkHTML: backlinkHTML}
}
