package output

import "io"

// TextRenderer — implemented by domain result types that want to define
// their own plain-text output. Replaces the formatter's generic %v fallback
// with a specialized rendering.
//
// Dependency direction: avoids a domain → output cycle. The domain does not
// import this interface; output.Print() detects implementations via type
// assertion (duck typing).
type TextRenderer interface {
	Text(w io.Writer)
}

// ConsoleRenderer — console-mode rendering (color + tables + emoji).
// When noColor=true, ANSI escapes are omitted (compatible with NO_COLOR / --no-color).
type ConsoleRenderer interface {
	Console(w io.Writer, noColor bool)
}
