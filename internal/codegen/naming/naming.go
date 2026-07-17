// Package naming holds the DDF-to-Go identifier rules shared by every
// emitter. All decisions about how a CSP, node or enum value becomes a Go
// name live here.
package naming

import (
	"strings"
	"unicode"
)

// PackageName derives the Go package (and directory) name for a snapshot
// base name, e.g. "Camera_AreaDDF" -> "camera", "ADMX_AppCompat_AreaDDF" ->
// "admx_appcompat", "LAPS" -> "laps".
func PackageName(snapshotBase string) string {
	s := strings.TrimSuffix(snapshotBase, "_AreaDDF")
	var b strings.Builder
	lastUnderscore := true // suppress leading underscores
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "_")
	if out == "" {
		return "csp"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "csp" + out
	}
	return out
}

// ImportAlias derives a package alias for the registry file, unique across
// both families: "csp"/"policy" prefix plus the package name without
// underscores.
func ImportAlias(family, pkg string) string {
	return family + strings.ReplaceAll(pkg, "_", "")
}

// ExportName converts an arbitrary DDF name into an exported Go identifier.
// Word boundaries are non-alphanumeric runs; existing capitalization inside
// a word is preserved ("ADMX_AppCompat" -> "ADMXAppCompat").
func ExportName(s string) string {
	var b strings.Builder
	newWord := true
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9':
			if newWord && r >= 'a' && r <= 'z' {
				r = unicode.ToUpper(r)
			}
			b.WriteRune(r)
			newWord = false
		default:
			newWord = true
		}
	}
	out := b.String()
	if out == "" {
		return "X"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "N" + out
	}
	return out
}

// JoinExport concatenates path segments into one exported identifier.
func JoinExport(segs []string) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(ExportName(s))
	}
	if b.Len() == 0 {
		return "Root"
	}
	return b.String()
}

// reserved lists Go keywords and predeclared identifiers that cannot be
// parameter names.
var reserved = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true,
	"for": true, "func": true, "go": true, "goto": true, "if": true,
	"import": true, "interface": true, "map": true, "package": true,
	"range": true, "return": true, "select": true, "struct": true,
	"switch": true, "type": true, "var": true,
	"string": true, "int": true, "bool": true, "byte": true, "error": true,
	"nil": true, "true": true, "false": true, "len": true, "cap": true,
	"ctx": true, "value": true, "s": true, // taken by method scaffolding
}

// ParamName converts a DDF title into an unexported Go parameter name.
func ParamName(s string) string {
	exp := ExportName(s)
	// Lowercase the leading upper-case run, keeping the last capital of a
	// multi-capital run ("ProfileName" -> "profileName", "LocURI" -> "locURI").
	runes := []rune(exp)
	i := 0
	for i < len(runes) && unicode.IsUpper(runes[i]) {
		i++
	}
	if i > 1 && i < len(runes) {
		i-- // keep the capital that starts the next word
	}
	for j := 0; j < i; j++ {
		runes[j] = unicode.ToLower(runes[j])
	}
	out := string(runes)
	if reserved[out] {
		out += "Name"
	}
	return out
}

// ConstName derives the member part of an enum constant from its value
// description, falling back to the raw value.
func ConstName(description, value string) string {
	d := strings.TrimSpace(description)
	// Keep it short: first sentence, capped word count.
	if i := strings.IndexAny(d, ".;:("); i > 0 {
		d = d[:i]
	}
	words := strings.Fields(d)
	if len(words) > 6 {
		words = words[:6]
	}
	name := ExportName(strings.Join(words, " "))
	if name == "X" || name == "" {
		name = ExportName("Value " + value)
	}
	return name
}

// FirstLine returns the first line of a description, trimmed.
func FirstLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// WrapComment wraps s into comment-friendly lines of at most width runes.
func WrapComment(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
			continue
		}
		line += " " + w
	}
	return append(lines, line)
}
