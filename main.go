package main

// TODO: 코드 이해하기
// TODO: 포스트 썸네일 기능 추가하기

import (
	"blog/lib"
	"hash/fnv"

	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v2"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"

	"github.com/gohugoio/hugo-goldmark-extensions/passthrough"
)

// TODO: post에 Thumnail 추가 -> header hero 처럼

// TODO: date를 지정하지 않으면 post list에 어떻게 뜨는가? -> Compile Error

// TODO: date가 같다면 post list에서 이름순(숫자->한글->영어)로 정렬되는가? -> Ok

// TODO: Category fixed through site.yml

type CustomSlice []string

func (c *CustomSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var array []string
	if err := unmarshal(&array); err == nil {
		*c = array
		return nil
	}
	var single string
	if err := unmarshal(&single); err != nil {
		return err
	}
	*c = []string{single}
	return nil
}

func main() {
	// Get env
	var appEnv string = os.Getenv("APP_ENV")
	_ = appEnv

	// Set goldmark
	var md = goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // markdown에서 html tag 사용할 수 있게 활성화함
		),
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(
					// chromahtml.WithLineNumbers(true),
					chromahtml.WithClasses(true),
				),
			),
			lib.MarkExtension,
			passthrough.New(
				passthrough.Config{
					InlineDelimiters: []passthrough.Delimiters{
						{Open: "$", Close: "$"},
						{Open: "\\(", Close: "\\)"},
					},
					BlockDelimiters: []passthrough.Delimiters{
						{Open: "$$", Close: "$$"},
						{Open: "\\[", Close: "\\]"},
					},
				},
			),
		),
		goldmark.WithParserOptions(
			parser.WithAttribute(), // 수동 id 지정 기능
		),
	)
	_ = md

	// PostsData
	fmt.Println()
	fmt.Println("-- PostsData 처리 --")
	var contentDirPath string = "./content"
	var contentFilePaths []string = lib.GetFilePaths(contentDirPath)
	_ = contentFilePaths

	var postsData []map[string]any
	var postsDataByCategory = make(map[string][]map[string]any)
	_ = postsData
	_ = postsDataByCategory

	slugTracker := make(map[string]bool)

	type PostFrontMatter struct {
		Date      string      `yaml:"date"`
		Desc      string      `yaml:"desc"`
		Category  CustomSlice `yaml:"category"`
		Published bool        `yaml:"published"`
		Fixed     bool        `yaml:"fixed"`
		Thumbnail string      `yaml:"thumbnail"`
	}

	for _, path := range contentFilePaths {
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("error: %s - 파일 열기 실패\n", path)
			continue
		}
		defer file.Close()
		buffer := make([]byte, 4096)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Printf("error: %s - 파일 읽기 실패\n", path)
			continue
		}
		content := buffer[:n]
		if !bytes.HasPrefix(content, []byte("---")) {
			fmt.Printf("error: %s - 프론트 매터를 찾을 수 없습니다\n", path)
			continue
		}
		parts := bytes.SplitN(content, []byte("---"), 3)
		if len(parts) < 3 {
			fmt.Printf("error: %s - 프론트 매터의 끝을 찾을 수 없습니다\n", path)
			continue
		}
		frontMatterBytes := parts[1]
		var fm PostFrontMatter
		if err := yaml.Unmarshal(frontMatterBytes, &fm); err != nil {
			fmt.Printf("error: %s - YAML 파싱 실패\n", path)
			continue
		}

		// 파일 경로에 /unpublished/ 디렉토리가 포함되어 있는지 확인하고 있으면 published: false와 동일하게 취급한다
		isUnpublishedDir := strings.Contains(filepath.ToSlash(path), "/unpublished/")

		// 프론트매터 published가 true이면서, unpublished 폴더에 있지 않은 파일만 PostsData에 추가한다
		if fm.Published && !isUnpublishedDir {
			// Date 누락 검사 및 경고 출력
			if fm.Date == "" {
				fmt.Printf("warning: %s - date 필드 누락\n", path)
				// FormatDateKorean 함수에서 에러가 나지 않도록 임시 기본값 할당
				fm.Date = "1970-01-01"
			}

			// Category 누락 검사 및 '미분류' 지정
			if len(fm.Category) == 0 {
				fm.Category = CustomSlice{"미분류"}
			}

			// title
			var title string = path[strings.LastIndex(path, "/")+1 : strings.LastIndex(path, ".")]

			// URL
			var slug string = lib.SlugifyPath(title)

			// 슬러그가 중복되는 경우, 파일 경로를 기반으로 한 6자리 해시값을 뒤에 붙임
			if slugTracker[slug] {
				h := fnv.New32a()
				h.Write([]byte(path))                         // 원본 파일 경로를 해시 재료로 사용
				hashStr := fmt.Sprintf("%08x", h.Sum32())[:4] // 4자리 추출

				oldSlug := slug
				slug = fmt.Sprintf("%s-%s", slug, hashStr)

				fmt.Printf("파일명 중복으로 인한 slug 변경: %s -> %s (%s)\n", oldSlug, slug, path)
			}
			slugTracker[slug] = true // 현재 슬러그를 사용 처리

			// description
			var description string
			if fm.Desc != "" {
				description = fm.Desc
			} else {
				bodyContent := bytes.TrimSpace(parts[2])
				plainText, err := lib.MarkdownToPlainText(bodyContent)
				if err != nil {
					fmt.Printf("error: %s - Markdown 변환 실패\n", path)
					description = ""
				} else {
					normalizedText := strings.Join(strings.Fields(plainText), " ")
					runes := []rune(normalizedText)
					if len(runes) > 100 {
						description = string(runes[:100]) + "..."
					} else {
						description = string(runes)
					}
				}
			}

			// PostsData 처리
			var data = map[string]any{
				"title":          title,
				"date":           fm.Date,
				"description":    description,
				"category":       []string(fm.Category),
				"fixed":          fm.Fixed,
				"thumbnail":      fm.Thumbnail,
				"sourceFilePath": path,
				"slug":           slug,
			}

			postsData = append(postsData, data)

			for _, category := range fm.Category {
				postsDataByCategory[category] = append(postsDataByCategory[category], data)
			}
		}
	}

	// 고정 카테고리 파일 처리 (content/pinned_categories.yml)
	fmt.Println()
	fmt.Println("-- 고정 카테고리 처리 --")

	var pinnedCategories []string
	const pinnedCategoriesFilePath = "content/pinned_categories.yml"

	pinnedFileBytes, pinnedFileErr := os.ReadFile(pinnedCategoriesFilePath)
	if pinnedFileErr != nil {
		if os.IsNotExist(pinnedFileErr) {
			fmt.Printf("info: %s 파일이 없습니다. 고정 카테고리 기능을 사용하지 않습니다.\n", pinnedCategoriesFilePath)
		} else {
			fmt.Printf("warning: %s 파일 읽기 실패: %v\n", pinnedCategoriesFilePath, pinnedFileErr)
		}
	} else {
		if err := yaml.Unmarshal(pinnedFileBytes, &pinnedCategories); err != nil {
			fmt.Printf("warning: %s YAML 파싱 실패: %v\n", pinnedCategoriesFilePath, err)
			pinnedCategories = nil
		} else {
			// 존재하지 않는 카테고리 경고 구축
			var nonExistentCategories []string
			for _, pc := range pinnedCategories {
				if _, exists := postsDataByCategory[pc]; !exists {
					nonExistentCategories = append(nonExistentCategories, pc)
				}
			}
			if len(nonExistentCategories) > 0 {
				fmt.Printf("warning: 고정된 카테고리 파일에 존재하지 않는 카테고리가 있습니다: %s\n", strings.Join(nonExistentCategories, ", "))
			}
			if len(pinnedCategories) > 0 {
				fmt.Printf("info: 고정 카테고리 %d개 로드 완료: %s\n", len(pinnedCategories), strings.Join(pinnedCategories, ", "))
			}
		}
	}

	// Post
	fmt.Println()
	fmt.Println("-- Post 처리 --")

	var postDirPath string = "public/post"
	if err := os.MkdirAll(postDirPath, 0755); err != nil {
		fmt.Printf("error: public/post - 디렉토리 생성 실패\n")
	}

	layoutFile := "layout/post.html"
	postTemplate, err := template.ParseFiles(layoutFile)
	if err != nil {
		fmt.Printf("error: layout/post.html - 템플릿 파싱 실패\n")
	}

	type CategoryInfo struct {
		Name string
		URL  string
	}

	type PostTmplData struct {
		IsProduction  bool
		Title         string
		Date          string
		FormattedDate string
		// Category      []string
		Description string
		Permalink   string
		Content     template.HTML
		CurrentURL  string // for nav tag
		Categories  []CategoryInfo
		Thumbnail   string
	}

	for _, data := range postsData {
		// Post 데이터 처리
		title, _ := data["title"].(string)
		date, _ := data["date"].(string)
		category, _ := data["category"].([]string)
		description, _ := data["description"].(string)
		sourceFilePath, _ := data["sourceFilePath"].(string) // content에 저장된 파일명
		slug, _ := data["slug"].(string)
		publicPath := filepath.Join("public", "post", fmt.Sprintf("%s.html", slug)) // public에 저장된 파일명
		var permalink string                                                        // 블로그의 고유 링크임
		if appEnv == "production" {
			permalink = filepath.ToSlash(filepath.Join("post", slug))
		} else {
			permalink = filepath.ToSlash(filepath.Join("post", fmt.Sprintf("%s.html", slug)))
		}
		formattedDate, err := lib.FormatDateKorean(date) // yyyy년 mm월 dd일
		if err != nil {
			fmt.Printf("error: %s - 날짜 변환 실패\n", sourceFilePath)
			continue // return 대신 continue를 사용하여 이 포스트만 건너뛰고 나머지 블로그는 정상 빌드되도록 수정
		}

		var categoriesData []CategoryInfo
		for _, c := range category {
			var linkURL string
			slugifiedPath := lib.SlugifyPath(c)

			if appEnv == "production" {
				linkURL = filepath.ToSlash(filepath.Join("/", "posts", slugifiedPath))
			} else {
				linkURL = filepath.ToSlash(filepath.Join("/", "posts", fmt.Sprintf("%s.html", slugifiedPath)))
			}

			categoriesData = append(categoriesData, CategoryInfo{
				Name: c,       // "Go"
				URL:  linkURL, // "/posts/go.html"
			})
		}

		// Post 생성
		sourceBytes, err := os.ReadFile(sourceFilePath)
		if err != nil {
			fmt.Printf("error: %s - 파일 읽기 실패\n", sourceFilePath)
			continue
		}
		var bodyBytes []byte
		if bytes.HasPrefix(sourceBytes, []byte("---")) {
			parts := bytes.SplitN(sourceBytes, []byte("---"), 3)
			if len(parts) >= 3 {
				bodyBytes = parts[2]
			} else {
				bodyBytes = sourceBytes
			}
		} else {
			bodyBytes = sourceBytes
		}
		var contentBuf bytes.Buffer
		context := parser.NewContext()
		if err := md.Convert(bodyBytes, &contentBuf, parser.WithContext(context)); err != nil {
			fmt.Printf("error: %s - Markdown 변환 실패\n", sourceFilePath)
			continue
		}

		outputFile, err := os.Create(publicPath)
		if err != nil {
			fmt.Printf("error: %s - 출력 파일 생성 실패\n", sourceFilePath)
			continue
		}
		defer outputFile.Close()

		thumbnail, _ := data["thumbnail"].(string)
		postTmplData := PostTmplData{
			IsProduction:  appEnv == "production",
			Title:         title,
			Date:          date,
			FormattedDate: formattedDate,
			// Category:      category,
			Description: description,
			Permalink:   permalink,
			Content:     template.HTML(contentBuf.String()),
			CurrentURL:  "/posts",
			Categories:  categoriesData,
			Thumbnail:   thumbnail,
		}

		if err := postTemplate.Execute(outputFile, postTmplData); err != nil {
			fmt.Printf("error: layoutpost.html - 템플릿 실행 실패\n")
			continue
		}

		fmt.Printf("sucess: %s 파일 생성\n", publicPath)
	}

	// Post list 처리
	fmt.Println()
	fmt.Println("-- Post List 처리 --")

	var postList []string // postListHtml strings.Builder 방식으로 바꾸기.

	var categories []string // map은 순서가 보장되지 않으니까 이런 방식으로 순서화한다.
	for category := range postsDataByCategory {
		categories = append(categories, category)
	}

	// 목록 Sort: 고정 카테고리를 먼저, 그 다음 일반 카테고리를 이름순으로 정렬
	// pinnedCategories 순서를 유지하여 인덱스 맵 생성
	pinnedIndexMap := make(map[string]int)
	for idx, pc := range pinnedCategories {
		pinnedIndexMap[pc] = idx
	}
	sort.Slice(categories, func(i, j int) bool {
		iPinnedIdx, iIsPinned := pinnedIndexMap[categories[i]]
		jPinnedIdx, jIsPinned := pinnedIndexMap[categories[j]]

		if iIsPinned && jIsPinned {
			// 둘 다 고정: pinnedCategories 파일에 적힌 순서대로
			return iPinnedIdx < jPinnedIdx
		}
		if iIsPinned {
			return true // i가 고정, j가 일반 -> i 먼저
		}
		if jIsPinned {
			return false // j가 고정, i가 일반 -> j 먼저
		}
		// 둘 다 일반: 이름순
		return lib.CompareStrings(categories[i], categories[j])
	})

	for _, category := range categories {
		var categoryLink string
		if appEnv == "production" {
			categoryLink = filepath.ToSlash(filepath.Join("posts", lib.SlugifyPath(category)))
		} else {
			categoryLink = filepath.ToSlash(filepath.Join("posts", fmt.Sprintf("%s.html", lib.SlugifyPath(category))))
		}
		_ = categoryLink

		posts := postsDataByCategory[category]
		postList = append(postList, "<section class=\"category-group\">")
		postList = append(postList, fmt.Sprintf("<h2 class=\"category-group-title\"><a href=\"%s\">[%s]</a></h2>", categoryLink, category))
		postList = append(postList, "<ul class=\"category-group-list\">")

		// fixed 정렬 ("전체" 카테고리에서는 fixed 무시)
		sort.Slice(posts, func(i, j int) bool {
			postI := posts[i]
			postJ := posts[j]

			fixedI, _ := postI["fixed"].(bool)
			fixedJ, _ := postJ["fixed"].(bool)

			// "전체" 카테고리에서는 fixed를 무시하고 날짜순으로만 정렬
			if category == "전체" {
				dateI, _ := postI["date"].(string)
				dateJ, _ := postJ["date"].(string)
				return dateI > dateJ
			}

			if fixedI != fixedJ {
				return fixedI
			}

			if fixedI {
				titleI, _ := postI["title"].(string)
				titleJ, _ := postJ["title"].(string)
				return lib.CompareStrings(titleI, titleJ)
			}

			dateI, _ := postI["date"].(string)
			dateJ, _ := postJ["date"].(string)
			return dateI > dateJ
		})

		const maxPostsToShow = 3
		postsToDisplay := posts
		needsMoreLink := false

		if len(posts) > maxPostsToShow {
			postsToDisplay = posts[:maxPostsToShow]
			needsMoreLink = true
		}

		for _, data := range postsToDisplay {
			isFixed, _ := data["fixed"].(bool)
			// "전체" 카테고리에서는 fixed 아이콘(☞)을 표시하지 않음
			if category == "전체" {
				isFixed = false
			}
			title, _ := data["title"].(string)
			date, _ := data["date"].(string) // yyyy-mm-dd
			description, _ := data["description"].(string)
			// categorySlice, _ := data["category"].([]string)
			slug, _ := data["slug"].(string)
			encodedSlug := url.PathEscape(slug)

			var permalink string // 블로그 링크는 Production에서는 .html가 없음
			if appEnv == "production" {
				permalink = filepath.ToSlash(filepath.Join("post", encodedSlug))
			} else {
				permalink = filepath.ToSlash(filepath.Join("post", fmt.Sprintf("%s.html", encodedSlug)))
			}

			formattedDate, err := lib.FormatDateKorean(date) // yyyy년 mm월 dd일
			if err != nil {
				fmt.Printf("날짜 변환 실패: %v\n", err)
				return
			}

			if isFixed {
				fixedTemplate := `<li>
                    <article class="post-item">
                        <h3 class="post-item-title"><a href="%s">☞ %s</a></h3>
                        <p class="post-item-date"><time datetime="%s">%s</time></p>
                        <p class="post-item-description">%s</p>
                    </article>
                </li>`
				postList = append(postList, fmt.Sprintf(fixedTemplate, permalink, title, date, formattedDate, description))
			} else {
				template := `<li>
                    <article class="post-item">
                        <h3 class="post-item-title"><a href="%s">%s</a></h3>
                        <p class="post-item-date"><time datetime="%s">%s</time></p>
                        <p class="post-item-description">%s</p>
                    </article>
                </li>`
				postList = append(postList, fmt.Sprintf(template, permalink, title, date, formattedDate, description))
			}
		}

		if needsMoreLink {
			var moreLinkURL string // 블로그 링크는 Production에서는 .html가 없음
			if appEnv == "production" {
				moreLinkURL = filepath.ToSlash(filepath.Join("posts", lib.SlugifyPath((category))))
			} else {
				moreLinkURL = filepath.ToSlash(filepath.Join("posts", fmt.Sprintf("%s.html", lib.SlugifyPath(category))))
			}
			moreLinkHTML := fmt.Sprintf(`<li><article class="post-more-link"><a href="%s">더보기...</a></article></li>`, moreLinkURL)
			postList = append(postList, moreLinkHTML)
		}

		postList = append(postList, "</ul>")
		postList = append(postList, "</section>")
	}
	htmlPostList := strings.Join(postList, "")

	type PostsPageTemplateData struct {
		IsProduction bool
		PostList     template.HTML
		CurrentURL   string
	}
	postsPageTemplateData := PostsPageTemplateData{
		IsProduction: appEnv == "production",
		PostList:     template.HTML(htmlPostList),
		CurrentURL:   "/posts",
	}

	tmpl, err := template.ParseFiles("./layout/posts.html")
	if err != nil {
		fmt.Printf("layout/posts.html 템플릿 파일 파싱 실패\n")
	}

	outputFile, err := os.Create("public/posts.html") // 파일이 없으면 새로 생성. 파일이 이미 있으면 초기화.
	if err != nil {
		fmt.Printf("출력 파일 생성 실패\n")
	}
	defer outputFile.Close()

	if err := tmpl.Execute(outputFile, postsPageTemplateData); err != nil {
		fmt.Printf("템플릿 실행 실패\n")
	}
	fmt.Printf("성공: public/posts.html 파일 생성\n")

	// Category 처리
	fmt.Println()
	fmt.Println("-- Category별 Post page 생성 --")

	categoryPageTmpl, err := template.ParseFiles("./layout/category.html")
	if err != nil {
		fmt.Printf("layout/category_page.html 템플릿 파일 파싱 실패: %v\n", err)
		return
	}

	categoryPostDir := "public/posts"
	if err := os.MkdirAll(categoryPostDir, 0755); err != nil {
		fmt.Printf("디렉토리 생성 실패: %v\n", err)
		return
	}

	// 정렬
	for category, posts := range postsDataByCategory {
		sort.Slice(posts, func(i, j int) bool {
			postI := posts[i]
			postJ := posts[j]
			fixedI, _ := postI["fixed"].(bool)
			fixedJ, _ := postJ["fixed"].(bool)

			// "전체" 카테고리에서는 fixed를 무시하고 날짜순으로만 정렬
			if category == "전체" {
				dateI, _ := postI["date"].(string)
				dateJ, _ := postJ["date"].(string)
				return dateI > dateJ
			}

			if fixedI != fixedJ {
				return fixedI
			}

			if fixedI {
				titleI, _ := postI["title"].(string)
				titleJ, _ := postJ["title"].(string)
				return lib.CompareStrings(titleI, titleJ)
			}

			dateI, _ := postI["date"].(string)
			dateJ, _ := postJ["date"].(string)
			return dateI > dateJ
		})

		var postListHtml strings.Builder
		postListHtml.WriteString("<section class=\"category-group\">")
		postListHtml.WriteString(fmt.Sprintf("<h2 class=\"category-group-title\">[%s]</h2>", category))
		postListHtml.WriteString("<ul class=\"category-group-list\">")

		for _, data := range posts {
			isFixed, _ := data["fixed"].(bool)
			// "전체" 카테고리 페이지에서는 fixed 아이콘(☞)을 표시하지 않음
			if category == "전체" {
				isFixed = false
			}
			title, _ := data["title"].(string)
			date, _ := data["date"].(string)
			description, _ := data["description"].(string)
			slug, _ := data["slug"].(string)
			encodedSlug := url.PathEscape(slug)

			var permalink string
			if appEnv == "production" {
				permalink = filepath.ToSlash(filepath.Join("/", "post", encodedSlug))
			} else {
				permalink = filepath.ToSlash(filepath.Join("/", "post", fmt.Sprintf("%s.html", encodedSlug)))
			}

			formattedDate, _ := lib.FormatDateKorean(date)

			var templateString string
			if isFixed {
				templateString = `<li>
					<article class="category-item">
						<h3 class="category-item-title"><a href="%s">☞ %s</a></h3>
						<p class="category-item-date"><time datetime="%s">%s</time></p>
						<p class="category-item-description">%s</p>
					</article>
				</li>`
				postListHtml.WriteString(fmt.Sprintf(templateString, permalink, title, date, formattedDate, description))
			} else {
				templateString = `<li>
					<article class="category-item">
						<h3 class="category-item-title"><a href="%s">%s</a></h3>
						<p class="category-item-date"><time datetime="%s">%s</time></p>
						<p class="category-item-description">%s</p>
					</article>
				</li>`
				postListHtml.WriteString(fmt.Sprintf(templateString, permalink, title, date, formattedDate, description))
			}
		}

		backButtonHTML :=
			`<li>
				<article class="back-link">
					<a href="#" id="back">돌아가기...</a>
				</article>
			</li>`
			// onclick="history.back(); return false;"
		postListHtml.WriteString(backButtonHTML)

		postListHtml.WriteString("</ul>")
		postListHtml.WriteString("</section>")

		type CategoryPageTemplateData struct {
			IsProduction bool
			CategoryName string
			PostList     template.HTML
			CurrentURL   string
		}

		templateData := CategoryPageTemplateData{
			IsProduction: appEnv == "production",
			CategoryName: category,
			PostList:     template.HTML(postListHtml.String()),
			CurrentURL:   "/posts",
		}

		var outputFilePath = filepath.Join(categoryPostDir, fmt.Sprintf("%s.html", lib.SlugifyPath(category)))

		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			fmt.Printf("출력 파일 생성 실패 (%s): %v\n", outputFilePath, err)
			continue
		}
		defer outputFile.Close()

		if err := categoryPageTmpl.Execute(outputFile, templateData); err != nil {
			fmt.Printf("템플릿 실행 실패 (%s): %v\n", outputFilePath, err)
		}

		fmt.Printf("성공: %s 파일 생성\n", outputFilePath)
	}

	// index.html 처리
	fmt.Println()
	fmt.Println("-- index.html 처리 --")

	homeMdBytes, err := os.ReadFile("content/home/home.md")
	if err != nil {
		fmt.Printf("home.md 파일 읽기 실패: %v\n", err)
		return
	}

	var homeBodyBytes []byte
	if bytes.HasPrefix(homeMdBytes, []byte("---")) {
		parts := bytes.SplitN(homeMdBytes, []byte("---"), 3)
		if len(parts) >= 3 {
			homeBodyBytes = parts[2]
		} else {
			homeBodyBytes = homeMdBytes
		}
	} else {
		homeBodyBytes = homeMdBytes
	}

	var homeContentBuf bytes.Buffer
	if err := md.Convert(homeBodyBytes, &homeContentBuf); err != nil {
		fmt.Printf("home.md 변환 실패: %v\n", err)
		return
	}

	sourceHomeFile := "layout/index.html"
	tmplHome, err := template.ParseFiles(sourceHomeFile)
	if err != nil {
		fmt.Printf("템플릿 파일 파싱 실패: %v\n", err)
		return
	}

	destHomeFile := "public/index.html"
	outputHomeFile, err := os.Create(destHomeFile)
	if err != nil {
		fmt.Printf("출력 파일 생성 실패: %v\n", err)
		return
	}
	defer outputHomeFile.Close()

	type HomePageData struct {
		IsProduction bool
		CurrentURL   string
		Content      template.HTML
	}
	homePageData := HomePageData{
		IsProduction: appEnv == "production",
		CurrentURL:   "/",
		Content:      template.HTML(homeContentBuf.String()),
	}

	if err := tmplHome.Execute(outputHomeFile, homePageData); err != nil {
		fmt.Printf("템플릿 실행 실패: %v\n", err)
		return
	}

	fmt.Printf("성공: %s 파일 생성\n", destHomeFile)

	// about.html 처리
	fmt.Println()
	fmt.Println("-- abou.html 처리 --")

	aboutMdBytes, err := os.ReadFile("content/about/about.md")
	if err != nil {
		fmt.Printf("about.md 파일 읽기 실패: %v\n", err)
		return
	}

	var aboutBodyBytes []byte
	if bytes.HasPrefix(aboutMdBytes, []byte("---")) {
		parts := bytes.SplitN(aboutMdBytes, []byte("---"), 3)
		if len(parts) >= 3 {
			aboutBodyBytes = parts[2]
		} else {
			aboutBodyBytes = aboutMdBytes
		}
	} else {
		aboutBodyBytes = aboutMdBytes
	}

	var aboutContentBuf bytes.Buffer
	if err := md.Convert(aboutBodyBytes, &aboutContentBuf); err != nil {
		fmt.Printf("about.md 변환 실패: %v\n", err)
		return
	}

	sourceAboutFile := "layout/about.html"
	tmplabout, err := template.ParseFiles(sourceAboutFile)
	if err != nil {
		fmt.Printf("템플릿 파일 파싱 실패: %v\n", err)
		return
	}

	destAboutFile := "public/about.html"
	outputAboutFile, err := os.Create(destAboutFile)
	if err != nil {
		fmt.Printf("출력 파일 생성 실패: %v\n", err)
		return
	}
	defer outputAboutFile.Close()

	type AboutPageData struct {
		IsProduction bool
		CurrentURL   string
		Content      template.HTML
	}
	aboutPageData := AboutPageData{
		IsProduction: appEnv == "production",
		CurrentURL:   "/about",
		Content:      template.HTML(aboutContentBuf.String()),
	}

	if err := tmplabout.Execute(outputAboutFile, aboutPageData); err != nil {
		fmt.Printf("템플릿 실행 실패: %v\n", err)
		return
	}

	fmt.Printf("성공: %s 파일 생성\n", destAboutFile)

	// sitemap.xml 처리
	type SitemapURL struct {
		XMLName    xml.Name `xml:"url"`
		Loc        string   `xml:"loc"`
		LastMod    string   `xml:"lastmod"`
		ChangeFreq string   `xml:"changefreq"`
	}

	type URLSet struct {
		XMLName     xml.Name     `xml:"urlset"`
		Xmlns       string       `xml:"xmlns,attr"`
		XmlnsXSI    string       `xml:"xmlns:xsi,attr"`
		XSILocation string       `xml:"xsi:schemaLocation,attr"`
		URLs        []SitemapURL `xml:"url"`
	}

	fmt.Println()
	fmt.Println("-- Sitemap 생성 --")

	const baseURL = "https://chebread.github.io"

	var urlset = &URLSet{
		Xmlns:       "http://www.sitemaps.org/schemas/sitemap/0.9",
		XmlnsXSI:    "http://www.w3.1/XMLSchema-instance",
		XSILocation: "http://www.sitemaps.org/schemas/sitemap/0.9 http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd",
	}

	var today = time.Now().Format("2006-01-02")

	var staticPages = []string{"", "about.html"} // index.html은 경로가 "" 임.
	for _, page := range staticPages {
		urlEntry := SitemapURL{
			Loc:        fmt.Sprintf("%s/%s", baseURL, page),
			LastMod:    today,
			ChangeFreq: "monthly",
		}
		urlset.URLs = append(urlset.URLs, urlEntry)
	}

	for _, data := range postsData {
		var slug, okSlug = data["slug"].(string)
		var date, okDate = data["date"].(string)

		if !okSlug || !okDate {
			continue
		}

		var postURL string
		if appEnv == "production" {
			postURL = fmt.Sprintf("%s/post/%s", baseURL, slug)
		} else {
			postURL = fmt.Sprintf("%s/post/%s.html", baseURL, slug)
		}

		var urlEntry = SitemapURL{
			Loc:        postURL,
			LastMod:    date,     // 포스트의 실제 최종 수정일
			ChangeFreq: "weekly", // 포스트는 비교적 자주 변경될 수 있으므로 weekly로 설정함
		}
		urlset.URLs = append(urlset.URLs, urlEntry)
	}

	xmlBytes, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		fmt.Printf("error: Sitemap XML 변환 실패\n")
		return
	}

	var sitemapContent = []byte(xml.Header + string(xmlBytes))

	var sitemapPath = "public/sitemap.xml"
	err = os.WriteFile(sitemapPath, sitemapContent, 0644)
	if err != nil {
		fmt.Printf("error: %s 파일 생성 실패\n", sitemapPath)
		return
	}

	fmt.Printf("성공: %s 파일 생성\n", sitemapPath)

	// RSS Feed 생성
	type RSSItem struct {
		Title       string   `xml:"title"`
		Link        string   `xml:"link"`
		Description string   `xml:"description"`
		PubDate     string   `xml:"pubDate"`
		GUID        string   `xml:"guid"`
		Categories  []string `xml:"category"`
	}

	type RSSChannel struct {
		Title         string    `xml:"title"`
		Link          string    `xml:"link"`
		Description   string    `xml:"description"`
		Language      string    `xml:"language"`
		LastBuildDate string    `xml:"lastBuildDate"`
		Generator     string    `xml:"generator"`
		Items         []RSSItem `xml:"item"`
	}

	type RSSFeed struct {
		XMLName xml.Name   `xml:"rss"`
		Version string     `xml:"version,attr"`
		Channel RSSChannel `xml:"channel"`
	}

	fmt.Println()
	fmt.Println("-- RSS Feed 생성 --")

	var rssItems []RSSItem
	rssPosts := make([]map[string]any, len(postsData))
	copy(rssPosts, postsData)
	sort.Slice(rssPosts, func(i, j int) bool {
		dateI, _ := rssPosts[i]["date"].(string)
		dateJ, _ := rssPosts[j]["date"].(string)
		return dateI > dateJ
	})

	for _, data := range rssPosts {
		var slug, okSlug = data["slug"].(string)
		var date, okDate = data["date"].(string)
		var title, _ = data["title"].(string)
		var description, _ = data["description"].(string)
		var category, _ = data["category"].([]string)

		if !okSlug || !okDate {
			continue
		}

		var postURL string
		if appEnv == "production" {
			postURL = fmt.Sprintf("%s/post/%s", baseURL, slug)
		} else {
			postURL = fmt.Sprintf("%s/post/%s.html", baseURL, slug)
		}

		kst := time.FixedZone("KST", 9*60*60)
		t, err := time.ParseInLocation("2006-01-02", date, kst)
		var pubDate string
		if err == nil {
			pubDate = t.Format(time.RFC1123Z)
		} else {
			pubDate = date
		}

		rssItems = append(rssItems, RSSItem{
			Title:       title,
			Link:        postURL,
			Description: description,
			PubDate:     pubDate,
			GUID:        postURL,
			Categories:  category,
		})
	}

	var rssFeed = &RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:         "차한음 블로그",
			Link:          baseURL,
			Description:   "개발자 차한음 블로그",
			Language:      "ko-KR",
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Generator:     "Go Custom SSG",
			Items:         rssItems,
		},
	}

	rssBytes, err := xml.MarshalIndent(rssFeed, "", "  ")
	if err != nil {
		fmt.Printf("error: RSS XML 변환 실패\n")
		return
	}

	var rssContent = []byte(xml.Header + string(rssBytes))
	var rssPath = "public/rss.xml"
	err = os.WriteFile(rssPath, rssContent, 0644)
	if err != nil {
		fmt.Printf("error: %s 파일 생성 실패\n", rssPath)
		return
	}

	fmt.Printf("성공: %s 파일 생성\n", rssPath)
}
