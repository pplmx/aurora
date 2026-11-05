package sqlite

import "strings"

// likeContains turns a reader's search term into a LIKE pattern that matches the
// term anywhere in a column, literally.
//
// It exists because seven repositories had written the same three-step
// expression by hand — trim, lower-case, escape — and the escaping is the part
// that matters: LIKE treats % and _ as wildcards, so a symbol containing one
// would match rows it has nothing to do with. Each copy being correct today is
// not the same as the next one being correct, and a repository that forgot the
// escape would fail only for terms containing those two characters.
//
// The second return value reports whether the term was empty after trimming, so
// a caller can fall back to the plain list instead of building a WHERE clause
// that matches everything.
func likeContains(term string) (pattern string, ok bool) {
	term = strings.TrimSpace(term)
	if term == "" {
		return "", false
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(term))
	return "%" + escaped + "%", true
}
