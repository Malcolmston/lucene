package lucene

import (
	"testing"
)

func mustParse(t *testing.T, p *Parser, s string) Query {
	t.Helper()
	q, err := p.Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return q
}

func TestParseTermAndField(t *testing.T) {
	p := NewParser(NewStandardAnalyzer(), "body")
	q := mustParse(t, p, "programming")
	tq, ok := q.(*TermQuery)
	if !ok || tq.Field != "body" || tq.Term != "programming" {
		t.Fatalf("term parse: %#v", q)
	}
	q = mustParse(t, p, "title:go")
	tq, ok = q.(*TermQuery)
	if !ok || tq.Field != "title" || tq.Term != "go" {
		t.Fatalf("field parse: %#v", q)
	}
}

func TestParsePhrase(t *testing.T) {
	p := NewParser(nil, "body")
	q := mustParse(t, p, `title:"hello world"`)
	pq, ok := q.(*PhraseQuery)
	if !ok || pq.Field != "title" || len(pq.Terms) != 2 {
		t.Fatalf("phrase parse: %#v", q)
	}
}

func TestParseBoolean(t *testing.T) {
	p := NewParser(nil, "body")
	q := mustParse(t, p, "+go -rust programming")
	bq, ok := q.(*BooleanQuery)
	if !ok || len(bq.Clauses) != 3 {
		t.Fatalf("boolean parse: %#v", q)
	}
	if bq.Clauses[0].Occur != Must || bq.Clauses[1].Occur != MustNot || bq.Clauses[2].Occur != Should {
		t.Errorf("occurs = %v %v %v", bq.Clauses[0].Occur, bq.Clauses[1].Occur, bq.Clauses[2].Occur)
	}
}

func TestParsePrefix(t *testing.T) {
	p := NewParser(nil, "body")
	q := mustParse(t, p, "net*")
	pq, ok := q.(*PrefixQuery)
	if !ok || pq.Prefix != "net" {
		t.Fatalf("prefix parse: %#v", q)
	}
	q = mustParse(t, p, "title:net*")
	if pq, ok := q.(*PrefixQuery); !ok || pq.Field != "title" {
		t.Fatalf("field prefix parse: %#v", q)
	}
}

func TestParseRange(t *testing.T) {
	p := NewParser(nil, "year")
	q := mustParse(t, p, "[2001 TO 2020]")
	rq, ok := q.(*RangeQuery)
	if !ok || rq.Lower != "2001" || rq.Upper != "2020" || !rq.IncludeLower || !rq.IncludeUpper {
		t.Fatalf("range parse: %#v", q)
	}
	q = mustParse(t, p, "year:{a TO *}")
	rq, ok = q.(*RangeQuery)
	if !ok || rq.Lower != "a" || rq.Upper != "" || rq.IncludeLower || rq.IncludeUpper {
		t.Fatalf("exclusive open range parse: %#v", q)
	}
}

func TestParseGrouping(t *testing.T) {
	p := NewParser(nil, "body")
	q := mustParse(t, p, "(go rust) +programming")
	bq, ok := q.(*BooleanQuery)
	if !ok || len(bq.Clauses) != 2 {
		t.Fatalf("group parse: %#v", q)
	}
	if _, ok := bq.Clauses[0].Query.(*BooleanQuery); !ok {
		t.Errorf("first clause not a sub-boolean: %#v", bq.Clauses[0].Query)
	}
}

func TestParseEmptyIsMatchAll(t *testing.T) {
	p := NewParser(nil, "body")
	q := mustParse(t, p, "   ")
	if _, ok := q.(*MatchAllQuery); !ok {
		t.Fatalf("empty parse = %#v, want MatchAll", q)
	}
}

func TestParseErrors(t *testing.T) {
	p := NewParser(nil, "body")
	cases := []string{
		`"unterminated`,
		`[2001 TO 2020`,
		`[2001 2020]`,
		`go )`,
		`(go`,
		`[* *]`,
	}
	for _, c := range cases {
		if _, err := p.Parse(c); err == nil {
			t.Errorf("Parse(%q) expected error", c)
		}
	}
}

func TestParseEndToEnd(t *testing.T) {
	idx := sampleIndex(t)
	res, err := idx.SearchString("body:programming +body:go -rust", 10)
	if err != nil {
		t.Fatal(err)
	}
	ids := hitIDs(res)
	if len(ids) != 2 || contains(ids, "2") {
		t.Errorf("end-to-end query ids = %v", ids)
	}
	// Malformed query surfaces error.
	if _, err := idx.SearchString(`"oops`, 10); err == nil {
		t.Error("expected parse error from SearchString")
	}
}

func TestParserDefaults(t *testing.T) {
	p := NewParser(nil, "")
	if p.defaultField != "text" {
		t.Errorf("default field = %q", p.defaultField)
	}
}
