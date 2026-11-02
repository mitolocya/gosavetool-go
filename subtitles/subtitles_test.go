package subtitles

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const srt = "1\r\n00:00:01,000 --> 00:00:03,500\r\nHello\r\nworld\r\n\r\n2\r\n00:00:03,000 --> 00:00:02,000\r\nSecond\r\n\r\n" +
	"3\r\n00:00:00,500 --> 00:00:00,900\r\nFirst\r\n"

// The checks of @gosavetool/subtitles
func TestParseAndWrite(t *testing.T) {
	cues := Parse(srt)
	if len(cues) != 3 || cues[0].Text != "Hello\nworld" {
		t.Fatalf("Parse: %#v", cues)
	}
	if !strings.HasPrefix(ToVTT(cues), "WEBVTT\n\n00:00:01.000 --> 00:00:03.500\nHello") {
		t.Errorf("ToVTT: %q", ToVTT(cues))
	}
	got, _ := Convert("WEBVTT\n\nNOTE hi\n\n01:02.5 --> 01:04.000 align:start\nA\n", SRT)
	if got != "1\n00:01:02,500 --> 00:01:04,000\nA\n" {
		t.Errorf("Convert to SRT: %q", got)
	}
	got, _ = Convert("WEBVTT\n\n01:02.5 --> 01:04.000 align:start\nA\n", VTT)
	if got != "WEBVTT\n\n00:01:02.500 --> 00:01:04.000 align:start\nA\n" {
		t.Errorf("Convert keeps settings: %q", got)
	}
	if _, err := Convert(srt, "ass"); err != ErrFormat {
		t.Errorf("Convert to an unknown format: %v", err)
	}
	if len(Parse("no cues here")) != 0 {
		t.Error("Parse of a text without cues")
	}
}

func TestShiftAndRepair(t *testing.T) {
	cues := Parse(srt)
	var moved [][2]int64
	for _, c := range Shift(cues, -1000) {
		moved = append(moved, [2]int64{c.Start, c.End})
	}
	if !reflect.DeepEqual(moved, [][2]int64{{0, 2500}, {2000, 1000}}) {
		t.Errorf("Shift: %v", moved)
	}
	fixed, n := Repair(cues, 2000)
	var texts []string
	for _, c := range fixed {
		texts = append(texts, c.Text)
		if c.End <= c.Start {
			t.Errorf("cue ends before it starts: %+v", c)
		}
	}
	if !reflect.DeepEqual(texts, []string{"First", "Hello\nworld", "Second"}) || n != 3 {
		t.Errorf("Repair: %q, %d", texts, n)
	}
	if cues[1].End != 2000 {
		t.Error("Repair changed its input")
	}
}

func TestTimes(t *testing.T) {
	if got := FromMs(3723004, ','); got != "01:02:03,004" {
		t.Errorf("FromMs: %s", got)
	}
	if ms, ok := ToMs("1:02:03.4"); !ok || ms != 3723400 {
		t.Errorf("ToMs: %d %v", ms, ok)
	}
	if _, ok := ToMs("nope"); ok {
		t.Error("ToMs of something else")
	}
	if !IsVTT("\ufeffWEBVTT\n") {
		t.Error("IsVTT with a byte order mark")
	}
}

func ExampleConvert() {
	out, _ := Convert("1\n00:00:01,000 --> 00:00:02,500\nHello\n", VTT)
	fmt.Print(out)
	// Output:
	// WEBVTT
	//
	// 00:00:01.000 --> 00:00:02.500
	// Hello
}

func ExampleRepair() {
	cues := Parse("1\n00:00:05,000 --> 00:00:04,000\nB\n\n2\n00:00:01,000 --> 00:00:06,000\nA\n")
	fixed, n := Repair(cues, 2000)
	fmt.Print(n, " fixes\n", ToSRT(fixed))
	// Output:
	// 3 fixes
	// 1
	// 00:00:01,000 --> 00:00:04,999
	// A
	//
	// 2
	// 00:00:05,000 --> 00:00:07,000
	// B
}
