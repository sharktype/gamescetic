package bugtracker

import (
	"os"
	"strings"
	"sync"

	"github.com/getsentry/sentry-go"
)

const sourceContextRadius = 5

type sourceFile struct {
	lines []string
	ok    bool
}

var sourceFiles sync.Map

func addSourceContext(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	for i := range event.Exception {
		stacktrace := event.Exception[i].Stacktrace
		if stacktrace == nil {
			continue
		}

		for j := range stacktrace.Frames {
			addFrameContext(&stacktrace.Frames[j])
		}
	}

	return event
}

func addFrameContext(frame *sentry.Frame) {
	if frame.AbsPath == "" || frame.Lineno <= 0 {
		return
	}

	source, ok := readSourceFile(frame.AbsPath)
	if !ok || frame.Lineno > len(source.lines) {
		return
	}

	line := frame.Lineno - 1
	start := max(0, line-sourceContextRadius)
	end := min(len(source.lines), line+sourceContextRadius+1)

	frame.ContextLine = source.lines[line]
	frame.PreContext = source.lines[start:line]
	frame.PostContext = source.lines[line+1 : end]
}

func readSourceFile(path string) (sourceFile, bool) {
	if cached, ok := sourceFiles.Load(path); ok {
		file := cached.(sourceFile)

		return file, file.ok
	}

	data, err := os.ReadFile(path)
	if err != nil {
		sourceFiles.Store(path, sourceFile{})

		return sourceFile{}, false
	}

	file := sourceFile{
		lines: strings.Split(strings.TrimSuffix(string(data), "\n"), "\n"),
		ok:    true,
	}
	sourceFiles.Store(path, file)

	return file, true
}
