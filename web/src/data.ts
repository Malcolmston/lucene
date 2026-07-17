// Library content for the lucene documentation site. Mirrors the shape used by
// the malcolmston/go landing site's data.ts so the sibling sites stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const LUCENE: Lib = {
  id:"lucene", name:"Lucene", icon:'<i class="fa-solid fa-magnifying-glass"></i>', accent:"#3fb6a8",
  pkg:"github.com/malcolmston/lucene", node:"apache/lucene",
  repo:"https://github.com/malcolmston/lucene", docs:"https://malcolmston.github.io/lucene/",
  tagline:"Embedded full-text search for Go, in the style of Apache Lucene.",
  blurb:"A small, dependency-free, in-memory full-text search engine written in pure Go, modelled on Apache "+
    "Lucene. A configurable analysis pipeline tokenizes, lowercases, drops stop words and stems your text; "+
    "an inverted index records term frequencies and positions for every field; and a rich query model — with "+
    "a query-string parser — is ranked by BM25 into deterministic top-N hits. Everything is built on the Go "+
    "standard library alone: no cgo, no third-party modules, nothing to audit but the toolchain.",
  tags:["analyzer pipeline","inverted index","BM25 ranking","phrase queries","query parser","prefix & range","highlighter","stdlib-only"],
  features:[
    "Analysis pipeline — <code>NewStandardAnalyzer</code> tokenizes, lowercases, drops stop words and stems, tunable via <code>WithStopWords</code> and <code>WithStemming</code>",
    "Inverted index — <code>NewIndex</code> with <code>Add</code> / <code>Delete</code> over <code>Document</code> values; postings carry term frequencies and positions, and it is safe for concurrent use",
    "Query model — <code>TermQuery</code>, <code>PhraseQuery</code>, <code>BooleanQuery</code> (<code>Must</code>/<code>Should</code>/<code>MustNot</code>), <code>PrefixQuery</code>, <code>RangeQuery</code> and <code>MatchAllQuery</code>",
    "Query-string parser — <code>NewParser</code> and <code>Parse</code> turn <code>title:go &quot;phrase&quot; +must -not net* [a TO z]</code> into a query tree",
    "BM25 relevance — <code>Search</code> returns a <code>Result</code> of top-ranked <code>Hit</code> values, ties broken by document ID for fully deterministic output",
    "One-call search — <code>SearchString</code> parses and executes a query string against the index's analyzer in a single step",
    "Highlighting — <code>NewHighlighter</code> and <code>Highlight</code> wrap matched (and stemmed) terms in custom markers while preserving the original text",
    "Zero dependencies — pure Go standard library, no cgo, no third-party modules"
  ],
  node_code:
`import org.apache.lucene.analysis.standard.StandardAnalyzer;
import org.apache.lucene.document.*;
import org.apache.lucene.index.*;
import org.apache.lucene.queryparser.classic.QueryParser;
import org.apache.lucene.search.*;
import org.apache.lucene.store.ByteBuffersDirectory;

Directory dir = new ByteBuffersDirectory();
IndexWriter w = new IndexWriter(dir, new IndexWriterConfig(new StandardAnalyzer()));

Document doc = new Document();
doc.add(new TextField("title", "The Go Programming Language", Field.Store.YES));
doc.add(new TextField("body", "Go is an open source programming language.", Field.Store.YES));
w.addDocument(doc);
w.close();

IndexSearcher searcher = new IndexSearcher(DirectoryReader.open(dir));
Query q = new QueryParser("body", new StandardAnalyzer()).parse("programming +go -rust");
TopDocs hits = searcher.search(q, 10);
System.out.println("matches: " + hits.totalHits.value);`,
  go_code:
`import "github.com/malcolmston/lucene"

idx := lucene.NewIndex(lucene.NewStandardAnalyzer())
_ = idx.Add(lucene.Document{ID: "1", Fields: map[string]string{
	"title": "The Go Programming Language",
	"body":  "Go is an open source programming language.",
}})

// Parse a query string and take the top 10 hits by BM25 score.
res, _ := idx.SearchString("body:programming +body:go -rust", 10)
fmt.Println("matches:", res.Total)
for _, hit := range res.Hits {
	fmt.Printf("  %s  %.3f\n", hit.ID, hit.Score)
}`,
  integrate:
`<span class="tok-c">// Build a boolean query by hand — the programmatic form of a</span>
<span class="tok-c">// "+must -not" query string, with clause-level Occur semantics.</span>
bq := lucene.NewBooleanQuery().
	Add(lucene.NewPhraseQuery("title", "programming", "language"), lucene.Must).
	Add(lucene.NewTermQuery("body", "google"), lucene.Should).
	Add(lucene.NewTermQuery("body", "rust"), lucene.MustNot)

res := idx.Search(bq, 10)
for _, hit := range res.Hits {
	fmt.Printf("%s  score=%.3f\n", hit.ID, hit.Score)
}

<span class="tok-c">// Prefix and range queries reach whole families of terms at once.</span>
res = idx.Search(lucene.NewPrefixQuery("body", "net"), 10)
res = idx.Search(lucene.NewRangeQuery("year", "2000", "2020", true, true), 10)

<span class="tok-c">// Highlight matched (and stemmed) words in a snippet with custom markers.</span>
h := lucene.NewHighlighter(idx.Analyzer(), "[", "]")
fmt.Println(h.Highlight("Go is a great programming language.",
	lucene.NewTermQuery("body", "programming")))`
};
