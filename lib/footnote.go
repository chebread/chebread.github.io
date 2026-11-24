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

type footnoteParagraphTransformer struct{}

func (t *footnoteParagraphTransformer) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != extast.KindFootnoteList {
			continue
		}
		for item := child.FirstChild(); item != nil; item = item.NextSibling() {
			fn, ok := item.(*extast.Footnote)
			if !ok {
				continue
			}
			first := fn.FirstChild()
			if first != nil && first.Kind() == gast.KindParagraph {
				numNode := &FootnoteNumber{Index: fn.Index}
				first.InsertBefore(first, first.FirstChild(), numNode)
				if fn.ChildCount() == 1 {
					first.SetAttributeString("id", []byte("fn:"+strconv.Itoa(fn.Index)))
				}
			}
		}
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
	if entering {
		_, _ = w.WriteString(`<div role="doc-endnotes">` + "\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
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
	extension.NewFootnote(
		extension.WithFootnoteBacklinkHTML(e.backlinkHTML),
	).Extend(m)

	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&footnoteParagraphTransformer{}, 1000),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&footnoteParagraphRenderer{}, 400),
		),
	)
}

// NewFootnoteExtension returns a Goldmark footnote extension that renders each footnote
// as a standard paragraph ("<p id=\"fn:1\"><sup>1</sup> ... [되돌아가기]</p>") instead of an <ol>/<li> list.
func NewFootnoteExtension(backlinkHTML string) goldmark.Extender {
	return &footnoteExtension{backlinkHTML: backlinkHTML}
}
