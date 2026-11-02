// Package slug turns a Jellyfin item into the readable part of a share URL.
//
// A slug is never the whole token: the caller appends a random suffix, so a
// readable link stays as unguessable as the old random one. The slug is fixed
// when the share is created; a later title change must not break the link.
package slug

import (
	"fmt"
	"strings"
)

// maxSlug keeps the finished token inside the 64-character public_token column
// once the hyphen and the 12-character suffix are added.
const maxSlug = 40

// Item is the part of a Jellyfin item a slug needs.
type Item struct {
	Type              string
	Name              string
	SeriesName        string
	IndexNumber       int
	ParentIndexNumber int
}

// Build returns a lowercase ASCII slug for the item, or "share" when the title
// has no usable Latin characters (a Thai-only title, for example).
func Build(item Item) string {
	s := normalize(rawName(item))
	if s == "" {
		return "share"
	}
	return s
}

// rawName picks the words that identify the item. A season or episode is named
// after its series plus its numbers, because Jellyfin calls them "Season 1" and
// "Pilot" on their own.
func rawName(item Item) string {
	switch item.Type {
	case "Season":
		series := item.SeriesName
		if series == "" {
			series = item.Name
		}
		return fmt.Sprintf("%s s%02d", series, item.IndexNumber)
	case "Episode":
		series := item.SeriesName
		if series == "" {
			series = item.Name
		}
		return fmt.Sprintf("%s s%02d ep %02d", series, item.ParentIndexNumber, item.IndexNumber)
	default:
		return item.Name
	}
}

// normalize lowercases, folds common accented Latin letters to ASCII, and
// collapses every other run of characters to a single hyphen.
func normalize(s string) string {
	var b strings.Builder
	lastHyphen := false

	for _, r := range strings.ToLower(s) {
		if r < 128 {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
				lastHyphen = false
			} else if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
			continue
		}

		if folded, ok := fold[r]; ok {
			b.WriteString(folded)
			lastHyphen = false
			continue
		}

		// Non-Latin scripts are dropped rather than percent-encoded. A separator
		// keeps the words on either side apart.
		if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}

	out := strings.Trim(b.String(), "-")
	if len(out) > maxSlug {
		out = strings.TrimRight(out[:maxSlug], "-")
	}
	return out
}

// fold maps the accented Latin letters that turn up in film and show titles to
// their ASCII form. Anything not listed is treated as non-Latin.
var fold = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'ç': "c", 'ć': "c", 'č': "c",
	'ď': "d", 'đ': "d",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'ğ': "g",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'į': "i", 'ı': "i",
	'ł': "l",
	'ñ': "n", 'ń': "n", 'ň': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ő': "o",
	'ř': "r",
	'ś': "s", 'š': "s", 'ş': "s",
	'ť': "t",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u", 'ű': "u",
	'ý': "y", 'ÿ': "y",
	'ž': "z", 'ź': "z", 'ż': "z",
	'æ': "ae", 'œ': "oe", 'ß': "ss", 'þ': "th",
}
