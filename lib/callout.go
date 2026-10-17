package lib

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// AlertBlock은 GitHub 스타일 Callout / Alert 블록을 나타내는 AST 노드입니다.
type AlertBlock struct {
	ast.BaseBlock
	AlertType  string // "note", "tip", "important", "warning", "caution"
	AlertTitle string // 표시될 제목 (예: "Note" 등)
}

var KindAlertBlock = ast.NewNodeKind("AlertBlock")

func (a *AlertBlock) Kind() ast.NodeKind {
	return KindAlertBlock
}

func (a *AlertBlock) Dump(source []byte, level int) {
	ast.DumpHelper(a, source, level, nil, nil)
}

// alertTagRegex는 > [!태그] 패턴을 감지합니다.
var alertTagRegex = regexp.MustCompile(`^\[!([^\]]+)\]`)

type alertTransformer struct{}

func (t *alertTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var toReplace [][2]ast.Node

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		bq, ok := n.(*ast.Blockquote)
		if !ok {
			return ast.WalkContinue, nil
		}

		firstBlock := bq.FirstChild()
		if firstBlock == nil || firstBlock.Kind() != ast.KindParagraph {
			return ast.WalkContinue, nil
		}

		para := firstBlock.(*ast.Paragraph)

		// 문단의 첫 번째 줄 텍스트를 수집합니다.
		var firstLineBuf bytes.Buffer
		var firstLineNodes []*ast.Text

		for c := para.FirstChild(); c != nil; c = c.NextSibling() {
			tn, ok := c.(*ast.Text)
			if !ok {
				break
			}
			firstLineNodes = append(firstLineNodes, tn)
			firstLineBuf.Write(tn.Segment.Value(source))
			if tn.HardLineBreak() || tn.SoftLineBreak() {
				break
			}
		}

		fullFirstLine := firstLineBuf.String()
		loc := alertTagRegex.FindStringSubmatchIndex(fullFirstLine)
		if loc == nil {
			return ast.WalkContinue, nil
		}

		rawTag := fullFirstLine[loc[2]:loc[3]]

		var alertClass string
		var alertTitle string

		if strings.Contains(rawTag, "|") {
			parts := strings.SplitN(rawTag, "|", 2)
			styleCandidate := strings.ToLower(strings.TrimSpace(parts[0]))
			titleCandidate := strings.TrimSpace(parts[1])

			switch styleCandidate {
			case "note", "tip", "important", "warning", "caution":
				alertClass = styleCandidate
			default:
				alertClass = "note"
			}

			if titleCandidate != "" {
				alertTitle = titleCandidate
			} else {
				alertTitle = strings.ToUpper(alertClass[:1]) + alertClass[1:]
			}
		} else {
			tagLower := strings.ToLower(rawTag)
			switch tagLower {
			case "note":
				alertClass = "note"
				alertTitle = "Note"
			case "tip":
				alertClass = "tip"
				alertTitle = "Tip"
			case "important":
				alertClass = "important"
				alertTitle = "Important"
			case "warning":
				alertClass = "warning"
				alertTitle = "Warning"
			case "caution":
				alertClass = "caution"
				alertTitle = "Caution"
			default:
				// 커스텀 태그 지원 (스타일 미지정 시 기본 note 스타일)
				alertClass = "note"
				alertTitle = rawTag
			}
		}

		alertNode := &AlertBlock{
			AlertType:  alertClass,
			AlertTitle: alertTitle,
		}

		// [!TAG]에 해당하는 길이만큼 텍스트 노드에서 제거
		tagEnd := loc[1]
		consumed := 0

		for _, tn := range firstLineNodes {
			valLen := tn.Segment.Len()
			if consumed+valLen <= tagEnd {
				consumed += valLen
				para.RemoveChild(para, tn)
			} else if consumed < tagEnd {
				offset := tagEnd - consumed
				consumed = tagEnd
				remVal := tn.Segment.Value(source)[offset:]
				trimmed := bytes.TrimLeft(remVal, " \t")
				diff := len(remVal) - len(trimmed)
				newStart := tn.Segment.Start + offset + diff
				if newStart < tn.Segment.Stop {
					tn.Segment = text.NewSegment(newStart, tn.Segment.Stop)
				} else {
					para.RemoveChild(para, tn)
				}
			} else {
				break
			}
		}

		// 문단이 비어있다면 문단 제거
		if para.FirstChild() == nil {
			bq.RemoveChild(bq, para)
		}

		// 기존 Blockquote의 모든 자식을 AlertBlock 노드로 이전
		for child := bq.FirstChild(); child != nil; {
			next := child.NextSibling()
			bq.RemoveChild(bq, child)
			alertNode.AppendChild(alertNode, child)
			child = next
		}

		toReplace = append(toReplace, [2]ast.Node{bq, alertNode})
		return ast.WalkSkipChildren, nil
	})

	for _, pair := range toReplace {
		oldNode, newNode := pair[0], pair[1]
		parent := oldNode.Parent()
		parent.ReplaceChild(parent, oldNode, newNode)
	}
}

type alertHTMLRenderer struct{}

func (r *alertHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAlertBlock, r.renderAlertBlock)
}

func (r *alertHTMLRenderer) renderAlertBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*AlertBlock)
	if entering {
		class := fmt.Sprintf("markdown-alert markdown-alert-%s", n.AlertType)
		w.WriteString(fmt.Sprintf("<div class=\"%s\">\n", class))
		w.WriteString("<p class=\"markdown-alert-title\">")

		icon := getAlertIconSVG(n.AlertType)
		if icon != "" {
			w.WriteString(icon)
		}
		w.WriteString(fmt.Sprintf("<span>%s</span>", stdhtml.EscapeString(n.AlertTitle)))
		w.WriteString("</p>\n")
	} else {
		w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

func getAlertIconSVG(alertType string) string {
	var path string
	switch alertType {
	case "tip":
		path = "M8 1.5c-2.363 0-4 1.69-4 3.75 0 .761.233 1.574.812 2.389.368.518.79 1.09 1.081 1.778.106.249.207.542.207.833 0 .414.336.75.75.75h2.3a.75.75 0 0 0 .75-.75c0-.291.101-.584.207-.833.291-.688.713-1.26 1.081-1.778.579-.815.812-1.628.812-2.389 0-2.06-1.637-3.75-4-3.75Zm0 1.5c1.464 0 2.5 1.01 2.5 2.25 0 .53-.166 1.108-.574 1.682-.36.507-.82 1.135-1.157 1.932H7.231c-.337-.797-.797-1.425-1.157-1.932-.408-.574-.574-1.152-.574-1.682 0-1.24 1.036-2.25 2.5-2.25ZM6.5 12a1 1 0 0 1 1-1h1a1 1 0 0 1 1 1v.5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1-.5-.5V12Zm.75 2.5a.75.75 0 0 0 0 1.5h1.5a.75.75 0 0 0 0-1.5h-1.5Z"
	case "important":
		path = "M0 1.75C0 .784.784 0 1.75 0h12.5C15.216 0 16 .784 16 1.75v9.5A1.75 1.75 0 0 1 14.25 13H8.06l-2.573 2.573A1.458 1.458 0 0 1 3 14.543V13H1.75A1.75 1.75 0 0 1 0 11.25Zm1.75-.25a.25.25 0 0 0-.25.25v9.5c0 .138.112.25.25.25h2a.75.75 0 0 1 .75.75v2.19l2.72-2.72a.749.749 0 0 1 .53-.22h6.5a.25.25 0 0 0 .25-.25v-9.5a.25.25 0 0 0-.25-.25Zm7 2.25v2.5a.75.75 0 0 1-1.5 0v-2.5a.75.75 0 0 1 1.5 0ZM9 9a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"
	case "warning":
		path = "M6.457 1.047c.659-1.234 2.427-1.234 3.086 0l6.082 11.378A1.75 1.75 0 0 1 14.082 15H1.918a1.75 1.75 0 0 1-1.543-2.575Zm1.763.707a.25.25 0 0 0-.44 0L1.698 13.132a.25.25 0 0 0 .22.368h12.164a.25.25 0 0 0 .22-.368Zm.53 3.996v2.5a.75.75 0 0 1-1.5 0v-2.5a.75.75 0 0 1 1.5 0ZM9 11a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"
	case "caution":
		path = "M4.47.22A.749.749 0 0 1 5 0h6c.199 0 .389.079.53.22l4.25 4.25c.141.14.22.331.22.53v6a.749.749 0 0 1-.22.53l-4.25 4.25A.749.749 0 0 1 11 16H5a.749.749 0 0 1-.53-.22L.22 11.53A.749.749 0 0 1 0 11V5c0-.199.079-.389.22-.53Zm.84 1.28L1.5 5.31v5.38l3.81 3.81h5.38l3.81-3.81V5.31L10.69 1.5ZM8 4a.75.75 0 0 1 .75.75v3.5a.75.75 0 0 1-1.5 0v-3.5A.75.75 0 0 1 8 4Zm0 8a1 1 0 1 1 0-2 1 1 0 0 1 0 2Z"
	default:
		path = "M0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8Zm8-6.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13ZM6.5 7.75A.75.75 0 0 1 7.25 7h1a.75.75 0 0 1 .75.75v2.75h.25a.75.75 0 0 1 0 1.5h-2a.75.75 0 0 1 0-1.5h.25v-2h-.25a.75.75 0 0 1-.75-.75ZM8 6a1 1 0 1 1 0-2 1 1 0 0 1 0 2Z"
	}

	return fmt.Sprintf(`<svg class="octicon" style="margin-right: 8px; vertical-align: text-bottom;" viewBox="0 0 16 16" version="1.1" width="16" height="16" aria-hidden="true" fill="currentColor"><path d="%s"></path></svg>`, path)
}

// CalloutExtension은 마크다운 > [!TAG] 구문을 GitHub 스타일 Alert로 변환해주는 확장입니다.
type calloutExtension struct{}

func (e *calloutExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&alertTransformer{}, 100),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&alertHTMLRenderer{}, 100),
		),
	)
}

var CalloutExtension = &calloutExtension{}
