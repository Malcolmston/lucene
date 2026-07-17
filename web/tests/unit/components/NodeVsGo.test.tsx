import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { NodeVsGo } from '../../../src/components/NodeVsGo';
import { LUCENE } from '../../../src/data';

describe('NodeVsGo', () => {
  it('renders the comparison heading and both Java and Go columns', () => {
    const { container } = render(<NodeVsGo lib={LUCENE} />);
    expect(container.querySelector(`#${LUCENE.id}-cmp`)).not.toBeNull();
    expect(screen.getByText('Java')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(container.querySelectorAll('.compare .code').length).toBe(2);
  });
});
