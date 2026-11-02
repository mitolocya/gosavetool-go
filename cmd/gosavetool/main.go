// Command gosavetool converts, shifts and repairs SRT and WebVTT subtitles, counts words and checks social media
// character limits. https://www.gosavetool.com/
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitolocya/gosavetool-go/subtitles"
	"github.com/mitolocya/gosavetool-go/text"
)

const help = `GoSaveTool: subtitle and text tools

Subtitles:
  gosavetool convert <file> --to srt|vtt   Write name.srt or name.vtt next to the file
  gosavetool shift <file> <seconds>        Move every cue (1.5 = later, -2 = earlier), in place
  gosavetool repair <file>                 Sort, fix overlaps and bad times, renumber, in place
  gosavetool info <file>                   Format, number of cues and the end time

Text (reads the file, or standard input without one):
  gosavetool count [file]                  Words, characters, sentences, lines, reading time
  gosavetool limits [file]                 Social media and SEO character limits
  gosavetool case <title|sentence|upper|lower|slug|camel|snake|kebab|clean> [file]

Options:
  -o, --out <file>   Write to this file instead
  --stdout           Print the result instead of writing a file

Online tools: https://www.gosavetool.com/`

var cases = map[string]func(string) string{
	"title": text.Title, "sentence": text.Sentence, "upper": text.Upper, "lower": text.Lower, "slug": text.Slug,
	"camel": text.Camel, "snake": text.Snake, "kebab": text.Kebab, "clean": text.Clean,
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gosavetool: %v\n\nRun gosavetool --help for usage.\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	take := func(names ...string) string {
		for i, a := range args {
			for _, n := range names {
				if a == n && i+1 < len(args) {
					v := args[i+1]
					args = append(args[:i:i], args[i+2:]...)
					return v
				}
			}
		}
		return ""
	}
	to := take("--to")
	out := take("-o", "--out")
	stdout := false
	for i, a := range args {
		if a == "--stdout" {
			stdout = true
			args = append(args[:i:i], args[i+1:]...)
			break
		}
	}
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Println(help)
		return nil
	}

	save := func(content, target string) error {
		if stdout {
			fmt.Print(content)
			return nil
		}
		if out != "" {
			target = out
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return err
		}
		fmt.Println("Wrote", target)
		return nil
	}
	arg := func(i int) string {
		if i < len(args) {
			return args[i]
		}
		return ""
	}

	switch cmd := args[0]; cmd {
	case "count", "limits", "case":
		file, op := arg(1), ""
		if cmd == "case" {
			op, file = arg(1), arg(2)
			if cases[op] == nil {
				return fmt.Errorf("case needs one of: title, sentence, upper, lower, slug, camel, snake, kebab, clean")
			}
		}
		var data []byte
		var err error
		if file == "" || file == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(file)
		}
		if err != nil {
			return err
		}
		s := string(data)
		// A file or a pipe ends with a newline that is not part of the text being counted
		counted := strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r")
		switch cmd {
		case "count":
			st := text.Count(counted)
			fmt.Printf("Words: %d\nCharacters: %d\nWithout spaces: %d\nSentences: %d\nLines: %d\nReading time: %d s\n",
				st.Words, st.Chars, st.CharsNoSpaces, st.Sentences, st.Lines, st.ReadSeconds)
		case "limits":
			for _, l := range text.CheckLimits(counted) {
				state := fmt.Sprintf("%d left", l.Left)
				if !l.Fits {
					state = fmt.Sprintf("%d over", -l.Left)
				}
				fmt.Printf("%-22s%5d  %s\n", l.Field, l.Limit, state)
			}
		default:
			result := cases[op](s)
			if out != "" {
				return save(result, out)
			}
			fmt.Print(result)
		}
		return nil

	case "convert", "shift", "repair", "info":
		file := arg(1)
		if file == "" {
			return errors.New("missing file")
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		src := string(data)
		cues := subtitles.Parse(src)
		if len(cues) == 0 {
			return fmt.Errorf("no subtitle cues found in %s", file)
		}
		own := subtitles.SRT
		if subtitles.IsVTT(src) {
			own = subtitles.VTT
		}
		switch cmd {
		case "convert":
			if to != "srt" && to != "vtt" {
				return errors.New("convert needs --to srt or --to vtt")
			}
			result, _ := subtitles.Write(cues, subtitles.Format(to))
			return save(result, strings.TrimSuffix(file, filepath.Ext(file))+"."+to)
		case "shift":
			sec, err := strconv.ParseFloat(strings.Replace(arg(2), ",", ".", 1), 64)
			if err != nil {
				return errors.New("shift needs seconds, like 1.5 or -2")
			}
			result, _ := subtitles.Write(subtitles.Shift(cues, int64(sec*1000+0.5*sign(sec))), own)
			return save(result, file)
		case "repair":
			fixed, n := subtitles.Repair(cues, 2000)
			result, _ := subtitles.Write(fixed, own)
			if err := save(result, file); err != nil {
				return err
			}
			if !stdout {
				if n > 0 {
					fmt.Println(n, "problems fixed")
				} else {
					fmt.Println("No problems found")
				}
			}
			return nil
		default:
			var end int64
			for _, c := range cues {
				if c.End > end {
					end = c.End
				}
			}
			name, sep := "SRT", byte(',')
			if own == subtitles.VTT {
				name, sep = "WebVTT", '.'
			}
			fmt.Printf("%s, %d cues, ends at %s\n", name, len(cues), subtitles.FromMs(float64(end), sep))
			return nil
		}

	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}
