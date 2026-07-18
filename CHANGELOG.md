# Changelog

All notable changes to this project are documented here. This project adheres
to semantic versioning.

## [0.2.0] - 2026-07-18

Feature expansion toward parity with Apache Lucene. All additions are pure
standard library, deterministic, and covered by known-answer table tests.

### Added

- **New query types** (`queries.go`):
  - `BoostQuery` / `NewBoostQuery` — scale a wrapped query's scores by a
    constant factor (mirrors Lucene's `BoostQuery`).
  - `ConstantScoreQuery` / `NewConstantScoreQuery` — assign every match a fixed
    score, for pure filtering (mirrors `ConstantScoreQuery`).
  - `DisjunctionMaxQuery` / `NewDisjunctionMaxQuery` — score a document by the
    maximum matching clause plus a tie-breaker fraction of the rest (mirrors
    `DisjunctionMaxQuery`, the standard cross-field query).
  - `TermsQuery` / `NewTermsQuery` — match any of several terms in a field
    (mirrors `TermInSetQuery`).
  - `RegexpQuery` / `NewRegexpQuery` — match terms against an anchored RE2
    regular expression (mirrors `RegexpQuery`).

- **Analysis building blocks** (`analysis_extra.go`): `NGrams`, `EdgeNGrams`
  (autocomplete indexing), and `Shingles` (word n-grams), mirroring Lucene's
  n-gram, edge-n-gram, and shingle token filters.

- **Phonetic encoding** (`phonetic.go`): `Soundex`, the classic homophone
  encoder from Lucene's analysis module.

- **String distance / similarity** (`distance.go`):
  `DamerauLevenshteinDistance` (adds transpositions to edit distance),
  `JaroSimilarity`, and `JaroWinklerSimilarity`, matching the distances used by
  Lucene's suggest module.

- **Index statistics** (`stats.go`): `Index.DocFreq`, `Index.TotalTermFreq`,
  `Index.TermCount`, and `Index.Terms`, exposing the term-dictionary statistics
  available from Lucene's `IndexReader`.

- **Faceting** (`facet.go`): `FacetResult` and `Index.FacetCounts`, a
  lightweight terms-facet over a field.

- **Suggesters** (`suggest.go`): `Index.Suggest` (prefix autocomplete ranked by
  document frequency) and `Index.SpellCheck` (edit-distance corrections),
  echoing Lucene's suggest/spellchecker.

- **Similar documents** (`morelikethis.go`): `Index.MoreLikeThis`, the
  tf-idf-driven related-document query modelled on Lucene's `MoreLikeThis`.

### Notes

Still absent versus upstream Lucene: on-disk segment persistence, numeric /
point fields and numeric range queries, positional slop in phrase queries,
stored-field sorting, and a pluggable Similarity interface.
