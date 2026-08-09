import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { CSSProperties, ReactNode } from 'react';
import TmPageContainer from '../TmPageContainer';

vi.mock('@ant-design/pro-components', () => ({
  PageContainer: ({
    children,
    className,
    style,
    title,
  }: {
    children?: ReactNode;
    className?: string;
    style?: CSSProperties;
    title?: ReactNode;
  }) => (
    <section className={className} style={style}>
      <h1>{title}</h1>
      {children}
    </section>
  ),
}));

describe('TmPageContainer', () => {
  it('uses the shared fluid content track by default', () => {
    const { container } = render(<TmPageContainer title="运营总览">页面内容</TmPageContainer>);

    const page = container.querySelector<HTMLElement>('.tm-page-container');
    expect(page).toHaveClass('tm-page-container--fluid');
    expect(page?.style.getPropertyValue('--tm-page-content-max-width')).toBe('1680px');
    expect(screen.getByText('页面内容')).toBeInTheDocument();
  });

  it('keeps an explicit constrained track available for narrow pages', () => {
    const { container } = render(
      <TmPageContainer title="设置" contentMaxWidth={960} fluid={false}>
        设置内容
      </TmPageContainer>,
    );

    const page = container.querySelector<HTMLElement>('.tm-page-container');
    expect(page).not.toHaveClass('tm-page-container--fluid');
    expect(page?.style.getPropertyValue('--tm-page-content-max-width')).toBe('960px');
  });
});
