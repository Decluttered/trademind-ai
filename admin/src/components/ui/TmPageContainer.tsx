import type { CSSProperties, ReactNode } from 'react';
import { PageContainer, type PageContainerProps } from '@ant-design/pro-components';
import { layoutTokens } from '@/constants/layoutTokens';

export type TmPageContainerProps = PageContainerProps & {
  /** Preferred maximum width of the page content; the fluid layout keeps expanding on wider viewports to avoid excessive whitespace on the sides. */
  contentMaxWidth?: number;
  /** Whether to constrain the outer whitespace on ultra-wide viewports; fixed-width content tracks like narrow forms can explicitly turn this off. */
  fluid?: boolean;
  /** Removes left/right padding on the content area, only for embedded special-case scenarios. */
  padded?: boolean;
  /** Style for the page content wrapper. */
  contentWrapperStyle?: CSSProperties;
};

/**
 * Unified page container: title + description on separate lines, consistent padding and max width.
 */
export default function TmPageContainer({
  title,
  subTitle,
  contentMaxWidth = layoutTokens.pageMaxWidth,
  fluid = true,
  padded = true,
  contentWrapperStyle,
  children,
  className,
  ...rest
}: TmPageContainerProps) {
  const wrapperStyle: CSSProperties = {
    '--tm-page-content-max-width': `${contentMaxWidth}px`,
    ...contentWrapperStyle,
  } as CSSProperties;

  return (
    <PageContainer
      {...rest}
      className={['tm-page-container', fluid ? 'tm-page-container--fluid' : '', className]
        .filter(Boolean)
        .join(' ')}
      style={{
        '--tm-page-content-max-width': `${contentMaxWidth}px`,
        ...rest.style,
      } as CSSProperties}
      title={title}
      subTitle={subTitle}
    >
      <div
        className={['tm-page-container__content', padded ? '' : 'tm-page-container__content--unpadded']
          .filter(Boolean)
          .join(' ')}
        style={wrapperStyle}
      >
        {children}
      </div>
    </PageContainer>
  );
}

export function TmPageHeaderExtra({ children }: { children: ReactNode }) {
  return <div className="tm-page-header-extra">{children}</div>;
}
