package lucene

import (
	"strings"
)

// Parser turns query strings into Query trees. The accepted grammar supports:
//
//   - term queries:            word
//   - field qualifiers:        title:word
//   - phrase queries:          "a b c"  (optionally field-qualified)
//   - required / prohibited:   +word  -word   (AND / NOT)
//   - prefix / wildcard:       net*    title:net*
//   - range queries:           [a TO z]  {1 TO 9}  (bounds, inclusive [] or exclusive {})
//   - grouping:                (a b) +c
//
// Bare, space-separated clauses combine as Should (OR) by default. A clause
// prefixed with '+' becomes Must (AND); a clause prefixed with '-' becomes
// MustNot (NOT). Field qualifiers apply to the clause they precede.
type Parser struct {
	analyzer     *Analyzer
	defaultField string
}

// NewParser builds a parser that analyzes terms with the given analyzer and
// resolves unqualified clauses against defaultField. If analyzer is nil,
// NewStandardAnalyzer is used; if defaultField is empty, "text" is used.
func NewParser(analyzer *Analyzer, defaultField string) *Parser {
	if analyzer == nil {
		analyzer = NewStandardAnalyzer()
	}
	if defaultField == "" {
		defaultField = "text"
	}
	return &Parser{analyzer: analyzer, defaultField: defaultField}
}

// Parse converts a query string into a Query. An empty or whitespace-only input
// yields a MatchAllQuery. Structural errors (unterminated phrase, malformed
// range, unbalanced parentheses) return an *Error.
func (p *Parser) Parse(input string) (Query, error) {
	toks, err := lex(input)
	if err != nil {
		return nil, err
	}
	ps := &parseState{toks: toks}
	q, err := p.parseClauses(ps, false)
	if err != nil {
		return nil, err
	}
	if ps.pos != len(ps.toks) {
		return nil, &Error{Op: "parse", Msg: "unexpected trailing input"}
	}
	if q == nil {
		return &MatchAllQuery{}, nil
	}
	return q, nil
}

// --- lexer ---

type tokKind int

const (
	tokTerm tokKind = iota
	tokPhrase
	tokPlus
	tokMinus
	tokColon
	tokLParen
	tokRParen
	tokLBrack // [
	tokRBrack // ]
	tokLBrace // {
	tokRBrace // }
	tokStar   // *
)

type lexToken struct {
	kind tokKind
	text string
}

func lex(input string) ([]lexToken, error) {
	var toks []lexToken
	runes := []rune(input)
	i := 0
	for i < len(runes) {
		r := runes[i]
		switch r {
		case ' ', '\t', '\n', '\r':
			i++
		case '"':
			j := i + 1
			for j < len(runes) && runes[j] != '"' {
				j++
			}
			if j >= len(runes) {
				return nil, &Error{Op: "parse", Msg: "unterminated phrase quote"}
			}
			toks = append(toks, lexToken{kind: tokPhrase, text: string(runes[i+1 : j])})
			i = j + 1
		case '+':
			toks = append(toks, lexToken{kind: tokPlus})
			i++
		case '-':
			toks = append(toks, lexToken{kind: tokMinus})
			i++
		case ':':
			toks = append(toks, lexToken{kind: tokColon})
			i++
		case '(':
			toks = append(toks, lexToken{kind: tokLParen})
			i++
		case ')':
			toks = append(toks, lexToken{kind: tokRParen})
			i++
		case '[':
			toks = append(toks, lexToken{kind: tokLBrack})
			i++
		case ']':
			toks = append(toks, lexToken{kind: tokRBrack})
			i++
		case '{':
			toks = append(toks, lexToken{kind: tokLBrace})
			i++
		case '}':
			toks = append(toks, lexToken{kind: tokRBrace})
			i++
		default:
			// Read a run of term characters. '*' is captured separately so a
			// trailing star can signal a prefix query.
			j := i
			var sb strings.Builder
			for j < len(runes) && isTermRune(runes[j]) {
				sb.WriteRune(runes[j])
				j++
			}
			if sb.Len() == 0 {
				// Unknown standalone character (e.g. '*'): emit as star or skip.
				if r == '*' {
					toks = append(toks, lexToken{kind: tokStar})
					i++
					continue
				}
				i++
				continue
			}
			toks = append(toks, lexToken{kind: tokTerm, text: sb.String()})
			i = j
			if j < len(runes) && runes[j] == '*' {
				toks = append(toks, lexToken{kind: tokStar})
				i++
			}
		}
	}
	return toks, nil
}

func isTermRune(r rune) bool {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return false
	}
	switch r {
	case '"', '+', '-', ':', '(', ')', '[', ']', '{', '}', '*':
		return false
	default:
		return true
	}
}

// --- parser ---

type parseState struct {
	toks []lexToken
	pos  int
}

func (ps *parseState) peek() (lexToken, bool) {
	if ps.pos >= len(ps.toks) {
		return lexToken{}, false
	}
	return ps.toks[ps.pos], true
}

func (ps *parseState) next() (lexToken, bool) {
	if ps.pos >= len(ps.toks) {
		return lexToken{}, false
	}
	t := ps.toks[ps.pos]
	ps.pos++
	return t, true
}

// parseClauses parses a sequence of clauses until end of input or, when
// inGroup is true, a closing parenthesis.
func (p *Parser) parseClauses(ps *parseState, inGroup bool) (Query, error) {
	bq := NewBooleanQuery()
	for {
		t, ok := ps.peek()
		if !ok {
			break
		}
		if t.kind == tokRParen {
			if inGroup {
				break
			}
			return nil, &Error{Op: "parse", Msg: "unbalanced ')'"}
		}
		occur := Should
		switch t.kind {
		case tokPlus:
			occur = Must
			ps.next()
		case tokMinus:
			occur = MustNot
			ps.next()
		}
		sub, err := p.parseClause(ps)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			// Nothing consumed; avoid an infinite loop.
			if _, ok := ps.next(); !ok {
				break
			}
			continue
		}
		bq.Add(sub, occur)
	}
	if len(bq.Clauses) == 0 {
		return nil, nil
	}
	// A single Should clause collapses to the bare sub-query for clarity.
	if len(bq.Clauses) == 1 && bq.Clauses[0].Occur == Should {
		return bq.Clauses[0].Query, nil
	}
	return bq, nil
}

// parseClause parses one clause: an optional field qualifier followed by a
// term, phrase, prefix, range, or parenthesized group.
func (p *Parser) parseClause(ps *parseState) (Query, error) {
	field := p.defaultField
	// Look for "field:" prefix.
	if t, ok := ps.peek(); ok && t.kind == tokTerm {
		if p.pos1IsColon(ps) {
			field = t.text
			ps.next() // term
			ps.next() // colon
		}
	}

	t, ok := ps.peek()
	if !ok {
		return nil, nil
	}
	switch t.kind {
	case tokLParen:
		ps.next()
		sub, err := p.parseClauses(ps, true)
		if err != nil {
			return nil, err
		}
		closing, ok := ps.next()
		if !ok || closing.kind != tokRParen {
			return nil, &Error{Op: "parse", Msg: "missing ')'"}
		}
		if sub == nil {
			return nil, nil
		}
		return sub, nil
	case tokLBrack, tokLBrace:
		return p.parseRange(ps, field)
	case tokPhrase:
		ps.next()
		terms := tokenize(t.text)
		return NewPhraseQuery(field, terms...), nil
	case tokTerm:
		ps.next()
		// Prefix query if immediately followed by a star.
		if star, ok := ps.peek(); ok && star.kind == tokStar {
			ps.next()
			return NewPrefixQuery(field, t.text), nil
		}
		return &TermQuery{Field: field, Term: t.text, Boost: 1}, nil
	default:
		return nil, nil
	}
}

// pos1IsColon reports whether the token after the current one is a colon,
// indicating a field qualifier.
func (p *Parser) pos1IsColon(ps *parseState) bool {
	if ps.pos+1 >= len(ps.toks) {
		return false
	}
	return ps.toks[ps.pos+1].kind == tokColon
}

// parseRange parses "[a TO b]" or "{a TO b}", with independent inclusive/
// exclusive bounds. A '*' bound means unbounded on that side.
func (p *Parser) parseRange(ps *parseState, field string) (Query, error) {
	open, _ := ps.next()
	includeLower := open.kind == tokLBrack

	lower, err := p.rangeBound(ps)
	if err != nil {
		return nil, err
	}
	// Expect a "TO" separator (case-insensitive term).
	to, ok := ps.next()
	if !ok || to.kind != tokTerm || !strings.EqualFold(to.text, "TO") {
		return nil, &Error{Op: "parse", Msg: "range query requires 'TO' separator"}
	}
	upper, err := p.rangeBound(ps)
	if err != nil {
		return nil, err
	}
	closing, ok := ps.next()
	if !ok || (closing.kind != tokRBrack && closing.kind != tokRBrace) {
		return nil, &Error{Op: "parse", Msg: "unterminated range query"}
	}
	includeUpper := closing.kind == tokRBrack

	return NewRangeQuery(field, lower, upper, includeLower, includeUpper), nil
}

// rangeBound reads a single range bound: a term, or '*' for unbounded.
func (p *Parser) rangeBound(ps *parseState) (string, error) {
	t, ok := ps.next()
	if !ok {
		return "", &Error{Op: "parse", Msg: "missing range bound"}
	}
	switch t.kind {
	case tokStar:
		return "", nil
	case tokTerm:
		return strings.ToLower(t.text), nil
	default:
		return "", &Error{Op: "parse", Msg: "invalid range bound"}
	}
}
