import { GlobalOutlined } from '@ant-design/icons';
import { Select } from 'antd';
import type { AdminLocale } from '@/locale';
import { ADMIN_LOCALES, useLocale } from '@/locale';

type LocaleSwitchProps = {
  className?: string;
  size?: 'small' | 'middle' | 'large';
};

export default function LocaleSwitch({
  className,
  size = 'middle',
}: LocaleSwitchProps) {
  const { locale, setLocale, t } = useLocale();

  return (
    <Select
      className={className}
      size={size}
      value={locale}
      aria-label={t('common.language')}
      popupMatchSelectWidth={false}
      suffixIcon={<GlobalOutlined aria-hidden />}
      options={ADMIN_LOCALES.map((code: AdminLocale) => ({
        value: code,
        label: t(`locale.${code}`),
      }))}
      onChange={(value) => setLocale(value as AdminLocale)}
    />
  );
}
