# gosavetool-go

Parse, convert, shift and repair **SRT and WebVTT** subtitles, and count **words, characters and reading time** with **social media character limits**, **Title Case**, **slugs** and other case changes. Two Go packages and a command-line tool, standard library only.

Made by [GoSaveTool](https://www.gosavetool.com/), free online tools for creators.

## Command line

```bash
go install github.com/mitolocya/gosavetool-go/cmd/gosavetool@latest

gosavetool convert movie.srt --to vtt      # writes movie.vtt
gosavetool shift movie.srt -1.5            # 1.5 s earlier, in place
gosavetool repair movie.srt                # sort, fix overlaps and bad times
gosavetool info movie.vtt                  # WebVTT, 412 cues, ends at 01:42:10.500

echo "Your caption" | gosavetool count
gosavetool limits post.txt                 # characters left for X, Instagram, TikTok, YouTube, SEO...
echo "the lord of the rings" | gosavetool case title
```

## Packages

```bash
go get github.com/mitolocya/gosavetool-go
```

```go
import (
	"github.com/mitolocya/gosavetool-go/subtitles"
	"github.com/mitolocya/gosavetool-go/text"
)

vtt, err := subtitles.Convert(srtText, subtitles.VTT)
cues := subtitles.Parse(src)                          // []Cue{Start, End (ms), Settings, Text}
later := subtitles.ToSRT(subtitles.Shift(cues, 1500))
fixed, n := subtitles.Repair(cues, 2000)              // sorted, no overlaps, no empty cues

st := text.Count(post)                                // Words, Chars, CharsNoSpaces, Lines, Sentences, ReadSeconds
fitsX := text.CheckLimits(post)[0].Fits               // x-post, instagram-caption, tiktok-caption, ...
text.Title("the lord of the rings")                   // "The Lord of the Rings"
text.Slug("Ölçü Birimi & Çeviri!")                    // "olcu-birimi-ceviri"
text.Camel("best posting time")                       // "bestPostingTime"
```

Characters are Unicode code points, so an emoji counts as one. Slugs handle Turkish and the common accented Latin letters. The rules follow the [`@gosavetool/subtitles`](https://www.npmjs.com/package/@gosavetool/subtitles) and [`@gosavetool/text`](https://www.npmjs.com/package/@gosavetool/text) JavaScript packages.

## Tests

```bash
go test ./...
go vet ./...
```

## Online tools

[VTT Subtitle Generator](https://www.gosavetool.com/vtt-subtitle-generator), [Auto Captions Generator](https://www.gosavetool.com/auto-captions-generator), [Character Counter](https://www.gosavetool.com/character-counter)

## License

MIT
