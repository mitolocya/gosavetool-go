package text

import (
	"fmt"
	"reflect"
	"testing"
)

// The checks of @gosavetool/text
func TestCount(t *testing.T) {
	want := Stats{Words: 4, Chars: 23, CharsNoSpaces: 20, Lines: 1, Sentences: 2, ReadSeconds: 1}
	if got := Count("Hello world. It’s 2026!"); got != want {
		t.Errorf("Count: %+v", got)
	}
	st := Count("Hello world. It’s 2026!\nSecond line.")
	if st.Words != 6 || st.Lines != 2 || st.Sentences != 3 {
		t.Errorf("Count over two lines: %+v", st)
	}
	if Count("").Lines != 0 {
		t.Error("an empty text has no lines")
	}
	if Count("\U0001F600a").Chars != 2 {
		t.Error("an emoji is one character")
	}
	x := CheckLimits(string(make([]byte, 300)))[0]
	if x.Field != "x-post" || x.Limit != 280 || x.Used != 300 || x.Left != -20 || x.Fits {
		t.Errorf("CheckLimits: %+v", x)
	}
	if len(CheckLimits("a")) != 9 {
		t.Error("CheckLimits returns every limit")
	}
}

func TestCases(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{Title("the lord of the rings: return of the king"), "The Lord of the Rings: Return of the King"},
		{Sentence("HELLO WORLD. how ARE you?"), "Hello world. How are you?"},
		{Slug("Ölçü Birimi & Çeviri!"), "olcu-birimi-ceviri"},
		{Slug("Straße café"), "strasse-cafe"},
		{Slug("café"), "cafe"},
		{Camel("best posting time"), "bestPostingTime"},
		{Snake("bestPostingTime"), "best_posting_time"},
		{Kebab("Best Posting_Time"), "best-posting-time"},
		{Clean("a   b  \n\n\n\nc\t"), "a b\n\nc"},
		{Clean("a  \r\n\r\n\r\n\r\nb"), "a\r\n\r\nb"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	if got := Words("HTMLParser v2Beta"); !reflect.DeepEqual(got, []string{"HTMLParser", "v2", "Beta"}) {
		t.Errorf("Words: %q", got)
	}
}

func ExampleTitle() {
	fmt.Println(Title("the lord of the rings: return of the king"))
	// Output: The Lord of the Rings: Return of the King
}

func ExampleSlug() {
	fmt.Println(Slug("Ölçü Birimi & Çeviri!"))
	// Output: olcu-birimi-ceviri
}

func ExampleCheckLimits() {
	for _, l := range CheckLimits("My new video is live!")[:2] {
		fmt.Println(l.Field, l.Left, l.Fits)
	}
	// Output:
	// x-post 259 true
	// instagram-caption 2179 true
}
