package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var (
	unwantedCharsRegex = regexp.MustCompile(`[^\p{Hangul}a-zA-Z0-9\s]+`)
	whitespaceRegex    = regexp.MustCompile(`\s+`)
)

func SlugifyPath(filePath string) string {
	filePath = norm.NFC.String(filePath)

	// 파일 경로에서 확장자를 제외한 순수 파일명을 가져옵니다
	base := filepath.Base(filePath)
	title := strings.TrimSuffix(base, filepath.Ext(base))

	// 허용되지 않는 모든 특수문자를 제거합니다
	safeTitle := unwantedCharsRegex.ReplaceAllString(title, "")

	// 하나 이상의 연속된 공백을 하이픈 1개로 변경합니다
	pathWithDashes := whitespaceRegex.ReplaceAllString(safeTitle, "-")

	// URL 일관성을 위해 모두 소문자로 변경합니다
	return norm.NFC.String(strings.ToLower(pathWithDashes))
}

// PathHash4 normalizes filePath to an OS- and Unicode-normalization-independent
// canonical form (cleaned relative path, '/' separators, Unicode NFC) and returns
// the first 2 bytes of its SHA-256 digest encoded as 4 lowercase hex characters.
func PathHash4(filePath string) string {
	normalizedPath := norm.NFC.String(filepath.ToSlash(filepath.Clean(filePath)))
	sum := sha256.Sum256([]byte(normalizedPath))
	return hex.EncodeToString(sum[:2])
}

// GeneratePostSlug builds an immutable "<baseSlug>-<4hex>" post slug derived solely
// from title and sourcePath. If baseSlug is empty (e.g., punctuation-only title),
// it returns the 4-character hash without a leading hyphen.
func GeneratePostSlug(title, sourcePath string) string {
	baseSlug := norm.NFC.String(SlugifyPath(title))
	hashStr := PathHash4(sourcePath)
	if baseSlug == "" {
		return hashStr
	}
	return fmt.Sprintf("%s-%s", baseSlug, hashStr)
}
