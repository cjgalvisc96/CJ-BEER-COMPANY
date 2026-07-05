package domain

import (
	"strings"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// Style is the brewing style of a beer, restricted to the styles the
// company actually produces.
type Style string

const (
	StyleLager  Style = "lager"
	StyleIPA    Style = "ipa"
	StyleStout  Style = "stout"
	StylePorter Style = "porter"
	StyleAle    Style = "ale"
	StyleWheat  Style = "wheat"
	StyleSour   Style = "sour"
)

var validStyles = map[Style]struct{}{
	StyleLager: {}, StyleIPA: {}, StyleStout: {}, StylePorter: {},
	StyleAle: {}, StyleWheat: {}, StyleSour: {},
}

func ParseStyle(raw string) (Style, error) {
	style := Style(strings.ToLower(strings.TrimSpace(raw)))
	if _, ok := validStyles[style]; !ok {
		return "", shared.NewValidationError("unknown beer style: " + raw)
	}
	return style, nil
}

func (s Style) String() string {
	return string(s)
}
