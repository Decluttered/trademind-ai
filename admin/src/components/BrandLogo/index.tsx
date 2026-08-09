/** 复用 Umi 约定的 `src/favicon.png`，避免在构建产物中再生成一份相同 Logo。 */

const logoUrl = '/favicon.png';

type BrandLogoProps = {
  /** CSS height in px; width follows aspect ratio. */
  height?: number;
  className?: string;
  /** Empty string when decorative (text label next to logo). */
  alt?: string;
};

export default function BrandLogo({ height = 28, className, alt = '' }: BrandLogoProps) {
  return (
    <img
      src={logoUrl}
      alt={alt}
      draggable={false}
      className={className}
      style={{
        height,
        width: 'auto',
        maxWidth: '100%',
        objectFit: 'contain',
        display: 'block',
        flexShrink: 0,
        verticalAlign: 'middle',
      }}
    />
  );
}
