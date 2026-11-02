// Package subtitles reads SRT and WebVTT cues, writes them in either format, and shifts and repairs their
// timings. No dependencies.
//
// It is the Go version of @gosavetool/subtitles. Online editor: https://www.gosavetool.com/vtt-subtitle-generator
package subtitles

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Cue is one subtitle cue. Times are in milliseconds.
type Cue struct {
	Start    int64
	End      int64
	Settings string // WebVTT cue settings, such as "align:start"; empty for SRT
	Text     string
}

// Format is a subtitle format.
type Format string

const (
	SRT Format = "srt"
	VTT Format = "vtt"
)

// ErrFormat is returned for a target format other than SRT or VTT.
var ErrFormat = errors.New(`subtitles: the target format must be "srt" or "vtt"`)

const stamp = `(?:\d{1,2}:)?\d{1,2}:\d{1,2}[,.]\d{1,3}`

var (
	timeRe    = regexp.MustCompile(`(\d{1,2}:)?(\d{1,2}):(\d{1,2})[,.](\d{1,3})`)
	arrowRe   = regexp.MustCompile(`^\s*(` + stamp + `)\s*-->\s*(` + stamp + `)(.*)$`)
	newlineRe = regexp.MustCompile(`\r*\n|\r`)
	blockRe   = regexp.MustCompile(`\n{2,}`)
)

// ToMs converts "00:01:02,500" or "01:02.5" to milliseconds. ok is false when it is not a timestamp.
func ToMs(s string) (ms int64, ok bool) {
	m := timeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	var h int64
	if m[1] != "" {
		h, _ = strconv.ParseInt(strings.TrimSuffix(m[1], ":"), 10, 64)
	}
	mins, _ := strconv.ParseInt(m[2], 10, 64)
	sec, _ := strconv.ParseInt(m[3], 10, 64)
	frac, _ := strconv.ParseInt((m[4] + "00")[:3], 10, 64)
	return ((h*60+mins)*60+sec)*1000 + frac, true
}

// FromMs formats milliseconds as "00:01:02,500" (sep ',', SRT) or "00:01:02.500" (sep '.', WebVTT).
func FromMs(ms float64, sep byte) string {
	// Half up, as JavaScript's Math.round
	t := int64(math.Max(0, math.Floor(ms+0.5)))
	return fmt.Sprintf("%02d:%02d:%02d%c%03d", t/3600000, t/60000%60, t/1000%60, sep, t%1000)
}

// Parse returns the cues of an SRT or WebVTT text. Headers, NOTE and STYLE blocks are skipped.
func Parse(src string) []Cue {
	// "\r\r\n" too: a file saved twice with Windows line endings
	text := newlineRe.ReplaceAllString(strings.TrimPrefix(src, "\ufeff"), "\n")
	cues := []Cue{}
	for _, block := range blockRe.Split(text, -1) {
		lines := strings.Split(block, "\n")
		for i, line := range lines {
			m := arrowRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			start, _ := ToMs(m[1])
			end, _ := ToMs(m[2])
			cues = append(cues, Cue{Start: start, End: end, Settings: strings.TrimSpace(m[3]), Text: strings.TrimSpace(strings.Join(lines[i+1:], "\n"))})
			break
		}
	}
	return cues
}

// IsVTT reports whether the text starts with the WEBVTT signature.
func IsVTT(src string) bool {
	return strings.HasPrefix(strings.TrimPrefix(src, "\ufeff"), "WEBVTT")
}

// ToSRT writes cues as SRT, numbered from 1.
func ToSRT(cues []Cue) string {
	parts := make([]string, len(cues))
	for i, c := range cues {
		parts[i] = fmt.Sprintf("%d\n%s --> %s\n%s", i+1, FromMs(float64(c.Start), ','), FromMs(float64(c.End), ','), c.Text)
	}
	return strings.Join(parts, "\n\n") + "\n"
}

// ToVTT writes cues as WebVTT; cue settings are kept.
func ToVTT(cues []Cue) string {
	parts := make([]string, len(cues))
	for i, c := range cues {
		settings := ""
		if c.Settings != "" {
			settings = " " + c.Settings
		}
		parts[i] = fmt.Sprintf("%s --> %s%s\n%s", FromMs(float64(c.Start), '.'), FromMs(float64(c.End), '.'), settings, c.Text)
	}
	return "WEBVTT\n\n" + strings.Join(parts, "\n\n") + "\n"
}

// Write writes cues in a format.
func Write(cues []Cue, to Format) (string, error) {
	switch to {
	case SRT:
		return ToSRT(cues), nil
	case VTT:
		return ToVTT(cues), nil
	}
	return "", ErrFormat
}

// Convert converts a subtitle text to SRT or VTT.
func Convert(src string, to Format) (string, error) {
	return Write(Parse(src), to)
}

// Shift moves every cue by ms (negative = earlier). Cues pushed before zero are clipped or dropped.
func Shift(cues []Cue, ms int64) []Cue {
	out := []Cue{}
	for _, c := range cues {
		end := max64(0, c.End+ms)
		if end > 0 {
			c.Start, c.End = max64(0, c.Start+ms), end
			out = append(out, c)
		}
	}
	return out
}

// Repair fixes common faults: cues out of order, an end before its start, overlapping cues and empty cues.
// Cues that end before they start get minDuration milliseconds (2000 is a good value). It returns the repaired
// cues and the number of changes; the input is not changed.
func Repair(cues []Cue, minDuration int64) ([]Cue, int) {
	fixed := 0
	out := []Cue{}
	for _, c := range cues {
		if c.Text == "" {
			fixed++
		} else {
			out = append(out, c)
		}
	}
	before := make([]int64, len(out))
	for i, c := range out {
		before[i] = c.Start
	}
	// SliceStable: cues with the same start keep their order
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	for i, c := range out {
		if c.Start != before[i] {
			fixed++
			break
		}
	}
	for i := range out {
		c := &out[i]
		if !(c.End > c.Start) {
			c.End = c.Start + minDuration
			fixed++
		}
		if i+1 < len(out) && c.End > out[i+1].Start {
			c.End = max64(c.Start+1, out[i+1].Start-1)
			fixed++
		}
	}
	return out, fixed
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
