package language

import (
	"fmt"
	"strings"

	textlanguage "golang.org/x/text/language"
)

func Normalize(code string, fallback string) (string, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		trimmed = fallback
	}

	if trimmed == "" {
		return "", fmt.Errorf("language code is required")
	}

	tag, err := textlanguage.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid language code %q: %w", trimmed, err)
	}

	base, _ := tag.Base()
	region, confidence := tag.Region()
	if confidence != textlanguage.No {
		return fmt.Sprintf("%s-%s", strings.ToLower(base.String()), region.String()), nil
	}

	return strings.ToLower(base.String()), nil
}
