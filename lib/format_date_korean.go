package lib

import (
	"fmt"
	"time"
)

// FormatDateKorean은 다음 형식을 모두 처리합니다:
//   - "2026-08-22"
//   - "2026-08-22T14:32:23"
//   - "2026-08-22T14:32:23+09:00" (RFC3339)
func FormatDateKorean(dateString string) (string, error) {
	if len(dateString) < 10 {
		return "", fmt.Errorf("날짜 형식 오류: %q", dateString)
	}

	t, err := time.Parse("2006-01-02", dateString[:10])
	if err != nil {
		return "", fmt.Errorf("날짜 파싱 실패: %q: %w", dateString, err)
	}

	return t.Format("2006년 01월 02일"), nil
}
