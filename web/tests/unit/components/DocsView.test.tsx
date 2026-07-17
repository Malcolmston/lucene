import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import type { DocIndex } from 'go-ui';

// A minimal DocIndex the stubbed fetch returns for DocsApp's doc.json request.
const DOC_INDEX: DocIndex = {
  module: 'github.com/malcolmston/lucene',
  packages: [
    {
      importPath: 'github.com/malcolmston/lucene',
      name: 'lucene',
      synopsis: 'Package lucene is a standard-library-only in-memory full-text search engine.',
      doc: 'Package lucene is a standard-library-only in-memory full-text search engine.',
      consts: [],
      vars: [],
      types: [
        {
          name: 'Index',
          signature: 'type Index struct{}',
          doc: 'Index is an in-memory inverted index over a set of documents.',
          consts: [],
          vars: [],
          funcs: [],
          methods: [],
        },
      ],
      funcs: [{ name: 'NewIndex', signature: 'func NewIndex(analyzer *Analyzer) *Index', doc: 'NewIndex creates an empty index.' }],
    },
  ],
};

describe('DocsView', () => {
  beforeEach(() => {
    // DocsApp fetches doc.json; return the small index.
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('doc.json')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(DOC_INDEX) } as Response);
      }
      return new Promise<Response>(() => {});
    }) as unknown as typeof fetch;
  });

  it('renders the inline React API reference from the fetched doc.json', async () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(
      screen.getByRole('heading', { level: 2, name: /API documentation/ }),
    ).toBeInTheDocument();

    // DocsApp fetches asynchronously, then renders the package view + symbols.
    expect(await screen.findByRole('heading', { name: /package lucene/ })).toBeInTheDocument();
    expect(container.querySelector('#sym-NewIndex'), 'func NewIndex symbol card').not.toBeNull();
    expect(container.querySelector('#sym-Index'), 'type Index symbol card').not.toBeNull();

    // The secondary link to the raw generated static HTML remains.
    expect(screen.getByRole('link', { name: /Open the raw generated HTML/ })).toHaveAttribute('href', './api/');
  });
});
