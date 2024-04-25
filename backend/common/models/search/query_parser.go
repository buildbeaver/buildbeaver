package search

import (
	"fmt"
	"strings"

	"github.com/buildbeaver/buildbeaver/common/models"
)

// ParseQuery parses a raw query string into a structured Query.
func ParseQuery(query string) Query {
	var (
		builder   = NewQueryBuilder()
		tokenizer = &queryTokenStream{input: &queryInputStream{query: query}}
	)
	for token := tokenizer.Next(); token != nil; token = tokenizer.Next() {
		switch t := token.(type) {
		case Term:
			builder = builder.Term(t)
		case models.ResourceKind:
			builder = builder.Kind(t)
		case FieldName:
			builder = builder.In(t)
		case *FieldFilter:
			builder = builder.Where(t.Field, t.Operator, t.Value)
		case *SortField:
			builder = builder.Sort(t.Field, t.Direction)
		default:
			panic(fmt.Sprintf("unknown token type: %T", t))
		}
	}
	return builder.Compile()
}

// queryInputStream provides a peekable stream over a raw query string
type queryInputStream struct {
	pos   int
	query string
}

// Next pops the next character off the stream and moves the cursor forward.
// Check EOF() first or this may panic.
func (s *queryInputStream) Next() rune {
	r := rune(s.query[s.pos])
	s.pos++
	return r
}

// Peek at the next character without popping it off the stream.
// Check EOF() first or this may panic.
func (s *queryInputStream) Peek() rune {
	return rune(s.query[s.pos])
}

// EOF returns true if the stream has been consumed.
func (s *queryInputStream) EOF() bool {
	return len(s.query) == s.pos
}

// queryTokenStream provides a stream of whole tokens over a raw input stream.
type queryTokenStream struct {
	input   *queryInputStream
	current interface{}
}

// Next pops the next token off the stream and moves the cursor forward.
// Returns nil on EOF.
func (s *queryTokenStream) Next() interface{} {
	next := s.current
	s.current = nil
	if next != nil {
		return next
	}
	return s.readNext()
}

// Peek at the next token without popping it off the stream.
// Returns nil on EOF.
func (s *queryTokenStream) Peek() interface{} {
	if s.current != nil {
		return s.current
	}
	s.current = s.readNext()
	return s.current
}

// EOF returns true if the stream has been consumed.
func (s *queryTokenStream) EOF() bool {
	return s.Peek() == nil
}

// readNext reads and pops the next token off the stream.
// Returns nil on EOF.
func (s *queryTokenStream) readNext() interface{} {
	var token string
	if s.input.EOF() {
		return nil
	}
	s.readWhile(func(ch rune) bool { // discard leading whitespace
		return ch == ' '
	})
	ch := s.input.Peek()
	if ch == '"' {
		token = s.readEscaped('"')
	} else {
		token = s.readWhile(func(ch rune) bool {
			return ch != ':' && ch != ' '
		})
		if !s.input.EOF() && s.input.Peek() == ':' {
			constraint, escaped := s.readConstraint()
			return s.parseCommand(token, constraint, escaped)
		}
	}
	return Term(token)
}

// readWhile pops the next rune off the stream whilst the callback function returns true.
// Returns the contiguous set of runes that were read as a string.
func (s *queryTokenStream) readWhile(fn func(ch rune) bool) string {
	var out string
	for !s.input.EOF() && fn(s.input.Peek()) {
		out += string(s.input.Next())
	}
	return out
}

// readEscaped pops a series of runes off the stream until the terminator rune is encountered.
// Runes within the bounds of the terminators that are escaped with \ are unescaped.
// Expects the stream to be positioned at the beginning of the opening terminator.
// The returned string contains the unescaped runes within the terminators e.g. does not return the terminators.
func (s *queryTokenStream) readEscaped(terminator rune) string {
	ch := s.input.Next()
	if ch != terminator {
		panic("expected escaped sequence to begin with terminator")
	}
	var (
		out     string
		escaped bool
	)
	for !s.input.EOF() {
		ch := s.input.Next()
		if escaped {
			out += string(ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else if ch == terminator {
			break
		} else {
			out += string(ch)
		}
	}
	return out
}

// readConstraint reads a constraint off of the stream.
// Expects the stream to be positioned at the ':' constraint marker.
// Returns the constraint and a bool indicating whether the constraint is escaped.
func (s *queryTokenStream) readConstraint() (string, bool) {
	ch := s.input.Next()
	if ch != ':' {
		panic("expected command sequence to begin with :")
	}
	var (
		constraint string
		escaped    bool
	)
	if !s.input.EOF() && s.input.Peek() == '"' {
		constraint = s.readEscaped('"')
		escaped = true
	} else { // TODO could support foo:>="hello world" if we check for operator here
		constraint = s.readWhile(func(ch rune) bool {
			return ch != ' '
		})
	}
	return constraint, escaped
}

// parseCommand parses a raw command:constraint combination into a typed command.
func (s *queryTokenStream) parseCommand(command string, constraint string, escaped bool) interface{} {
	switch command {
	case "in":
		return FieldName(constraint)
	case "kind":
		return models.ResourceKind(constraint)
	case "sort":
		return s.parseSortCommand(command, constraint)
	default:
		return s.parseFieldCommand(command, constraint, escaped)
	}
}

// parseSortCommand parses a raw sort: command into a typed sort command.
func (s *queryTokenStream) parseSortCommand(command string, constraint string) interface{} {
	if strings.HasSuffix(constraint, "-asc") {
		return NewSortField(strings.TrimSuffix(constraint, "-asc"), Ascending)
	} else if strings.HasSuffix(constraint, "-desc") {
		return NewSortField(strings.TrimSuffix(constraint, "-desc"), Descending)
	} else {
		return NewSortField(constraint, Ascending)
	}
}

// parseSortCommand parses a raw field: command into a typed field filter.
func (s *queryTokenStream) parseFieldCommand(command string, constraint string, escaped bool) interface{} {
	operator := Equal
	if !escaped {
		for _, op := range operatorSet {
			if strings.HasPrefix(constraint, op.String()) {
				constraint = strings.TrimPrefix(constraint, op.String())
				operator = op
				break
			}
		}
	}
	return NewFieldFilter(FieldName(command), operator, constraint)
}
