/**
 * @ant-design/pro-card's CardProps does not include antd Card's variant;
 * the business code consistently writes variant="outlined", so this supplements the type to avoid IDE errors.
 */
export {};

type ProCardVariant = 'borderless' | 'outlined' | 'filled';

type ProCardPropsWithVariant = import('@ant-design/pro-components').ProCardProps & {
  variant?: ProCardVariant;
};

declare module '@ant-design/pro-components' {
  import type { FC } from 'react';

  export const ProCard: FC<ProCardPropsWithVariant> & {
    isProCard: boolean;
    Divider: FC<Record<string, unknown>>;
    TabPane: FC<Record<string, unknown>>;
    Group: FC<ProCardPropsWithVariant>;
  };
}
