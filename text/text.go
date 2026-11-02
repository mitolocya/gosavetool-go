// Package text counts words, characters and reading time, checks social media character limits, and changes
// case: Title Case, Sentence case, URL slugs, camelCase, snake_case and kebab-case. No dependencies.
//
// It is the Go version of @gosavetool/text. Online counter: https://www.gosavetool.com/character-counter
package text

import (
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limit is how a text meets one character limit.
type Limit struct {
	Field string
	Limit int
	Used  int
	Left  int // negative when the text is too long
	Fits  bool
}

// Limits are the character limits of common fields, in display order. SEO values are the length search results
// show before cutting.
var Limits = []struct {
	Field string
	Limit int
}{
	{"x-post", 280},
	{"instagram-caption", 2200},
	{"tiktok-caption", 4000},
	{"youtube-title", 100},
	{"youtube-description", 5000},
	{"linkedin-post", 3000},
	{"pinterest-description", 500},
	{"seo-title", 60},
	{"meta-description", 160},
}

var small = map[string]bool{}

func init() {
	for _, w := range strings.Fields("a an and as at but by for in nor of on or so the to up yet via vs") {
		small[w] = true
	}
}

// Whitespace as JavaScript's \s sees it
const ws = `[\pZ\s\x{0B}\x{FEFF}]`

var (
	wordRe          = regexp.MustCompile(`[\pL\pN][\pL\pN'\x{2019}_-]*`)
	titleWordRe     = regexp.MustCompile(`[\pL\pN][\pL\pN'\x{2019}]*`)
	sentenceRe      = regexp.MustCompile(`[^.!?\x{2026}]+[.!?\x{2026}]+`)
	sentenceStartRe = regexp.MustCompile(`(^` + ws + `*|[.!?\x{2026}]` + ws + `+)(\pL)`)
	afterColonRe    = regexp.MustCompile(`[:.!?]` + ws + `*$`)
	lineBreakRe     = regexp.MustCompile(`\r\n|\r|\n`)
	camelBoundaryRe = regexp.MustCompile(`([\p{Ll}\pN])(\p{Lu})`)
	nonWordRe       = regexp.MustCompile(`[^\pL\pN]+`)
	runOfSpacesRe   = regexp.MustCompile(`([^\pZ\s\x{0B}\x{FEFF}])[ \t]{2,}`)
	trailingRe      = regexp.MustCompile(`[ \t]+$`)
	nonSlugRe       = regexp.MustCompile(`[^a-z0-9]+`)
)

// Stats are the counts of a text. Characters are Unicode code points, so an emoji counts as one.
type Stats struct {
	Words         int
	Chars         int
	CharsNoSpaces int
	Lines         int
	Sentences     int
	ReadSeconds   int // at 238 words a minute, the average silent reading speed for English non-fiction
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r) || r == '﻿'
}

// Count returns the words, characters, lines, sentences and reading time of a text.
func Count(s string) Stats {
	words := len(wordRe.FindAllStringIndex(s, -1))
	noSpaces := 0
	for _, r := range s {
		if !isSpace(r) {
			noSpaces++
		}
	}
	lines := 0
	if s != "" {
		lines = len(lineBreakRe.FindAllStringIndex(s, -1)) + 1
	}
	return Stats{
		Words:         words,
		Chars:         utf8.RuneCountInString(s),
		CharsNoSpaces: noSpaces,
		Lines:         lines,
		Sentences:     len(sentenceRe.FindAllStringIndex(s, -1)),
		ReadSeconds:   int(math.Floor(float64(words)/238*60 + 0.5)),
	}
}

// CheckLimits returns how the text meets every limit in Limits, in that order.
func CheckLimits(s string) []Limit {
	used := utf8.RuneCountInString(s)
	out := make([]Limit, 0, len(Limits))
	for _, l := range Limits {
		out = append(out, Limit{Field: l.Field, Limit: l.Limit, Used: used, Left: l.Limit - used, Fits: used <= l.Limit})
	}
	return out
}

// Words splits a text for camelCase and the like: "bestPostingTime 2" gives ["best" "Posting" "Time" "2"].
func Words(s string) []string {
	spaced := camelBoundaryRe.ReplaceAllString(s, "$1 $2")
	out := []string{}
	for _, w := range nonWordRe.Split(spaced, -1) {
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

// Upper returns the text in UPPERCASE.
func Upper(s string) string { return strings.ToUpper(s) }

// Lower returns the text in lowercase.
func Lower(s string) string { return strings.ToLower(s) }

func capitalize(w string) string {
	r, size := utf8.DecodeRuneInString(w)
	if size == 0 {
		return w
	}
	return strings.ToUpper(string(r)) + w[size:]
}

// Title returns the text in Title Case: "the lord of the rings" gives "The Lord of the Rings". Short words
// (a, of, the, and...) stay in lower case unless they start the text or follow a colon.
func Title(s string) string {
	low := strings.ToLower(s)
	var b strings.Builder
	last := 0
	for i, m := range titleWordRe.FindAllStringIndex(low, -1) {
		w := low[m[0]:m[1]]
		afterColon := afterColonRe.MatchString(low[:m[0]])
		b.WriteString(low[last:m[0]])
		if i == 0 || afterColon || !small[w] {
			b.WriteString(capitalize(w))
		} else {
			b.WriteString(w)
		}
		last = m[1]
	}
	b.WriteString(low[last:])
	return b.String()
}

// Sentence returns the text in Sentence case: "HELLO WORLD. how ARE you?" gives "Hello world. How are you?".
func Sentence(s string) string {
	low := strings.ToLower(s)
	var b strings.Builder
	last := 0
	for _, m := range sentenceStartRe.FindAllStringSubmatchIndex(low, -1) {
		// m[4]:m[5] is the letter to capitalize
		b.WriteString(low[last:m[4]])
		b.WriteString(strings.ToUpper(low[m[4]:m[5]]))
		last = m[5]
	}
	b.WriteString(low[last:])
	return b.String()
}

// Latin letters with accents and their plain forms, for slugs
var folds = map[string]string{
	"àáâãäåāăąǎÀÁÂÃÄÅĀĂĄǍ": "a", "çćĉċčÇĆĈĊČ": "c", "ďđðĎĐÐ": "d", "èéêëēĕėęěÈÉÊËĒĔĖĘĚ": "e", "ĝğġģĜĞĠĢ": "g",
	"ĥħĤĦ": "h", "ìíîïĩīĭįıÌÍÎÏĨĪĬĮİ": "i", "ĵĴ": "j", "ķĶ": "k", "ĺļľŀłĹĻĽĿŁ": "l", "ñńņňŉÑŃŅŇ": "n",
	"òóôõöøōŏőǒÒÓÔÕÖØŌŎŐǑ": "o", "ŕŗřŔŖŘ": "r", "śŝşšșŚŜŞŠȘ": "s", "ţťŧțŢŤŦȚ": "t", "ùúûüũūŭůűųǔÙÚÛÜŨŪŬŮŰŲǓ": "u",
	"ŵŴ": "w", "ýÿŷÝŸŶ": "y", "źżžŹŻŽ": "z", "ß": "ss", "æÆ": "ae", "œŒ": "oe", "þÞ": "th",
}

var foldRune = map[rune]string{}

func init() {
	for group, plain := range folds {
		for _, r := range group {
			foldRune[r] = plain
		}
	}
}

// Slug returns a URL slug: "Ölçü Birimi & Çeviri!" gives "olcu-birimi-ceviri". Turkish and the common accented
// Latin letters become plain letters.
func Slug(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0x300 && r <= 0x36f {
			continue // combining accents of decomposed letters
		}
		if p, ok := foldRune[r]; ok {
			b.WriteString(p)
		} else {
			b.WriteRune(r)
		}
	}
	return strings.Trim(nonSlugRe.ReplaceAllString(strings.ToLower(b.String()), "-"), "-")
}

// Camel returns the text in camelCase: "best posting time" gives "bestPostingTime".
func Camel(s string) string {
	var b strings.Builder
	for i, w := range Words(s) {
		if i == 0 {
			b.WriteString(strings.ToLower(w))
		} else {
			b.WriteString(capitalize(strings.ToLower(w)))
		}
	}
	return b.String()
}

func joinLower(s, sep string) string {
	ws := Words(s)
	for i, w := range ws {
		ws[i] = strings.ToLower(w)
	}
	return strings.Join(ws, sep)
}

// Snake returns the text in snake_case.
func Snake(s string) string { return joinLower(s, "_") }

// Kebab returns the text in kebab-case.
func Kebab(s string) string { return joinLower(s, "-") }

// Clean removes trailing spaces, runs of spaces and more than one blank line in a row.
func Clean(s string) string {
	nl := "\n"
	if strings.Contains(s, "\r\n") {
		nl = "\r\n"
	}
	parts := strings.Split(s, "\n")
	for i, p := range parts {
		if i < len(parts)-1 {
			p = strings.TrimSuffix(p, "\r")
		}
		parts[i] = runOfSpacesRe.ReplaceAllString(trailingRe.ReplaceAllString(p, ""), "$1 ")
	}
	out := strings.Join(parts, nl)
	three, two := strings.Repeat(nl, 3), strings.Repeat(nl, 2)
	for strings.Contains(out, three) {
		out = strings.ReplaceAll(out, three, two)
	}
	return out
}
