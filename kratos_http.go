package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	pgs "github.com/lyft/protoc-gen-star"
	"google.golang.org/genproto/googleapis/api/annotations"
)

var methodSets = make(map[string]int)

func buildHTTPRule(m pgs.Method, rule *annotations.HttpRule) *methodDesc {
	var (
		path         string
		method       string
		body         string
		responseBody string
	)

	switch pattern := rule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		path = pattern.Get
		method = "GET"
	case *annotations.HttpRule_Put:
		path = pattern.Put
		method = "PUT"
	case *annotations.HttpRule_Post:
		path = pattern.Post
		method = "POST"
	case *annotations.HttpRule_Delete:
		path = pattern.Delete
		method = "DELETE"
	case *annotations.HttpRule_Patch:
		path = pattern.Patch
		method = "PATCH"
	case *annotations.HttpRule_Custom:
		path = pattern.Custom.Path
		method = pattern.Custom.Kind
	}
	body = rule.Body
	responseBody = rule.ResponseBody
	md := buildMethodDesc(m, method, path)
	if method == "GET" || method == "DELETE" {
		if body != "" {
			_, _ = fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: %s %s body should not be declared.\n", method, path)
		}
	} else {
		if body == "" {
			_, _ = fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: %s %s does not declare a body.\n", method, path)
		}
	}
	if body == "*" {
		md.HasBody = true
		md.Body = ""
	} else if body != "" {
		md.HasBody = true
		md.Body = "." + camelCaseVars(body)
	} else {
		md.HasBody = false
	}
	if responseBody == "*" {
		md.ResponseBody = ""
	} else if responseBody != "" {
		md.ResponseBody = "." + camelCaseVars(responseBody)
	}
	return md
}

func buildPathVars(path string) (res map[string]*string) {
	if strings.HasSuffix(path, "/") {
		fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: Path %s should not end with \"/\" \n", path)
	}
	res = make(map[string]*string)
	pattern := regexp.MustCompile(`(?i){([a-z\.0-9_\s]*)=?([^{}]*)}`)
	matches := pattern.FindAllStringSubmatch(path, -1)
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		if len(name) > 1 && len(m[2]) > 0 {
			res[name] = &m[2]
		} else {
			res[name] = nil
		}
	}
	return
}

func buildMethodDesc(m pgs.Method, method, path string) *methodDesc {
	defer func() { methodSets[string(m.Name())]++ }()

	vars := buildPathVars(path)

	for v, s := range vars {
		fields := m.Input().Fields()
		_ = fields

		if s != nil {
			path = replacePath(v, *s, path)
		}
		for _, field := range strings.Split(v, ".") {
			if strings.TrimSpace(field) == "" {
				continue
			}
			if strings.Contains(field, ":") {
				field = strings.Split(field, ":")[0]
			}

			// fd := fields.ByName(protoreflect.Name(field))
			// if fd == nil {
			// 	fmt.Fprintf(os.Stderr, "\u001B[31mERROR\u001B[m: The corresponding field '%s' declaration in message could not be found in '%s'\n", v, path)
			// 	os.Exit(2)
			// }
			// if fd.IsMap() {
			// 	fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: The field in path:'%s' shouldn't be a map.\n", v)
			// } else if fd.IsList() {
			// 	fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: The field in path:'%s' shouldn't be a list.\n", v)
			// } else if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
			// 	fields = fd.Message().Fields()
			// }
		}
	}
	return &methodDesc{
		Name:         string(m.Name()),
		OriginalName: string(m.Name()),
		Num:          methodSets[string(m.Name())],
		Request:      m.Input().Name().String(),
		Reply:        m.Output().Name().String(),
		Path:         path,
		Method:       method,
		HasVars:      len(vars) > 0,
	}
}

func camelCaseVars(s string) string {
	subs := strings.Split(s, ".")
	vars := make([]string, 0, len(subs))
	for _, sub := range subs {
		vars = append(vars, camelCase(sub))
	}
	return strings.Join(vars, ".")
}

func replacePath(name string, value string, path string) string {
	pattern := regexp.MustCompile(fmt.Sprintf(`(?i){([\s]*%s[\s]*)=?([^{}]*)}`, name))
	idx := pattern.FindStringIndex(path)
	if len(idx) > 0 {
		path = fmt.Sprintf("%s{%s:%s}%s",
			path[:idx[0]], // The start of the match
			name,
			strings.ReplaceAll(value, "*", ".*"),
			path[idx[1]:],
		)
	}
	return path
}

// camelCase returns the CamelCased name.
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
// There is a remote possibility of this rewrite causing a name collision,
// but it's so remote we're prepared to pretend it's nonexistent - since the
// C++ generator lowercases names, it's extremely unlikely to have two fields
// with different capitalizations.
// In short, _my_field_name_2 becomes XMyFieldName_2.
func camelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
