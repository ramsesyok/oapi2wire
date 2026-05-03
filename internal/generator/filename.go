package generator

import "strings"

func safeFileNamePart(s string) string {
	var b strings.Builder
	lastUnderscore := false

	for _, r := range s {
		safe := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' ||
			r == '_'

		if safe {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}

		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}

	part := strings.Trim(b.String(), "_")
	if part == "" {
		return "unnamed"
	}
	return part
}
