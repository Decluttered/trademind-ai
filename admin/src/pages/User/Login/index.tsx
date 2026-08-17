import {
  ArrowRightOutlined,
  CheckCircleOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  DashboardOutlined,
  FileImageOutlined,
  InboxOutlined,
  LockOutlined,
  MailOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import { Form, Input, Checkbox, Button, Tabs, Row, Col } from 'antd';
import { history, Link, useLocation } from '@umijs/max';
import { useInitialStateModel } from '@/hooks/useInitialStateModel';
import { message } from 'antd';
import { useEffect, useState, useRef } from 'react';
import BrandLogo from '@/components/BrandLogo';
import LocaleSwitch from '@/components/layout/LocaleSwitch';
import { AUTH_TOKEN_KEY, AUTH_SESSION_MODE_KEY } from '@/constants/auth';
import { formatUserErrorMessage } from '@/constants/errorMessages';
import { useLocale } from '@/locale';
import { login, register, sendEmailCode } from '@/services/auth';
import './index.less';

const PLATFORM_ITEMS = ['Amazon.de', 'eBay.de'];

type AuthTabKey = 'login' | 'register';

function authTabFromPathname(pathname: string): AuthTabKey {
  return pathname.replace(/\/+$/, '') === '/user/register' ? 'register' : 'login';
}

type ApiErrorLike = {
  response?: { data?: { message?: string } };
  message?: string;
};

function getAuthErrorMessage(error: unknown, fallback: string) {
  const ax = error as ApiErrorLike;
  const raw = ax?.response?.data?.message || ax?.message;
  return formatUserErrorMessage(raw, fallback);
}

export default function LoginPage() {
  const { setInitialState, initialState } = useInitialStateModel();
  const { t } = useLocale();
  const location = useLocation();
  const [loading, setLoading] = useState(false);
  const activeTab = authTabFromPathname(location.pathname);

  const [loginForm] = Form.useForm();
  const [registerForm] = Form.useForm();

  const [countdown, setCountdown] = useState(0);
  const countdownTimer = useRef<NodeJS.Timeout | null>(null);

  const featureTags = [
    { icon: <CloudDownloadOutlined />, label: t('login.featureCollect'), className: 'tag-blue' },
    { icon: <RobotOutlined />, label: t('login.featureAI'), className: 'tag-violet' },
    { icon: <FileImageOutlined />, label: t('login.featureImage'), className: 'tag-teal' },
    { icon: <CloudUploadOutlined />, label: t('login.featurePublish'), className: 'tag-indigo' },
    { icon: <InboxOutlined />, label: t('login.featureInventory'), className: 'tag-green' },
    { icon: <DashboardOutlined />, label: t('login.featureDashboard'), className: 'tag-amber' },
  ] as const;

  const loggedIn = Boolean(initialState?.currentUser);
  useEffect(() => {
    if (!loggedIn) return;
    const q = new URLSearchParams(history.location.search);
    history.replace(q.get('redirect') || '/dashboard');
  }, [loggedIn]);

  useEffect(() => {
    return () => {
      if (countdownTimer.current) clearInterval(countdownTimer.current);
    };
  }, []);

  const changeAuthTab = (key: string) => {
    const nextTab: AuthTabKey = key === 'register' ? 'register' : 'login';
    history.replace(
      `${nextTab === 'register' ? '/user/register' : '/user/login'}${location.search}`,
    );
  };

  const onLogin = async (values: any) => {
    setLoading(true);
    try {
      const data = await login(values.account as string, values.password as string);
      localStorage.setItem(AUTH_TOKEN_KEY, data.token);
      if (data.sessionMode) {
        localStorage.setItem(AUTH_SESSION_MODE_KEY, data.sessionMode);
      }
      await setInitialState((s) => ({ ...s, currentUser: data.user }));
      message.success(t('login.successLogin'));
    } catch (e: unknown) {
      message.error(getAuthErrorMessage(e, t('login.failLogin')));
    } finally {
      setLoading(false);
    }
  };

  const onRegister = async (values: any) => {
    setLoading(true);
    try {
      const data = await register({
        email: values.email,
        code: values.code,
        password: values.password,
        confirmPassword: values.confirmPassword,
      });
      localStorage.setItem(AUTH_TOKEN_KEY, data.token);
      await setInitialState((s) => ({ ...s, currentUser: data.user }));
      message.success(t('login.successRegister'));
    } catch (e: unknown) {
      message.error(getAuthErrorMessage(e, t('login.failRegister')));
    } finally {
      setLoading(false);
    }
  };

  const handleSendCode = async () => {
    try {
      await registerForm.validateFields(['email']);
    } catch {
      return;
    }
    const email = registerForm.getFieldValue('email');
    try {
      await sendEmailCode(email, 'register');
      message.success(t('login.codeSent'));
      setCountdown(60);
      countdownTimer.current = setInterval(() => {
        setCountdown((c) => {
          if (c <= 1) {
            if (countdownTimer.current) clearInterval(countdownTimer.current);
            return 0;
          }
          return c - 1;
        });
      }, 1000);
    } catch (e: unknown) {
      message.error(getAuthErrorMessage(e, t('login.codeSendFail')));
    }
  };

  return (
    <div className="login-shell">
      <div className="login-container">
        <div className="login-left">
          <div className="login-left-decor" aria-hidden="true">
            <div className="decor-line decor-line-1" />
            <div className="decor-line decor-line-2" />
            <div className="decor-card decor-card-1" />
            <div className="decor-card decor-card-2" />
            <div className="decor-card decor-card-3" />
          </div>

          <div className="login-left-content">
            <div className="brand">
              <BrandLogo height={32} />
              <div>
                <div className="brand-text">{t('login.brand')}</div>
                <div className="brand-sub">{t('login.brandSub')}</div>
              </div>
            </div>

            <div className="slogan">
              <div className="eyebrow">
                <SafetyCertificateOutlined />
                <span>{t('login.eyebrow')}</span>
              </div>
              <h1>
                {t('login.sloganTitleBefore')}{' '}
                <span className="highlight">{t('login.sloganTitleHighlight')}</span>{' '}
                {t('login.sloganTitleAfter')}
              </h1>
              <p>{t('login.sloganBody')}</p>
            </div>

            <div className="features">
              {featureTags.map((tag, index) => (
                <div
                  key={tag.label}
                  className={`feature-tag ${tag.className}`}
                  style={{ animationDelay: `${index * 80 + 180}ms` }}
                >
                  {tag.icon} {tag.label}
                </div>
              ))}
            </div>

            <div className="hero-board" aria-hidden="true">
              <div className="hero-board__top">
                <div>
                  <span className="hero-board__label">Today GMV</span>
                  <strong>¥128,430</strong>
                </div>
                <span className="hero-board__badge">+18.6%</span>
              </div>
              <div className="hero-board__chart">
                <span className="chart-bar chart-bar-1" />
                <span className="chart-bar chart-bar-2" />
                <span className="chart-bar chart-bar-3" />
                <span className="chart-bar chart-bar-4" />
                <span className="chart-bar chart-bar-5" />
              </div>
              <div className="hero-board__flow">
                {PLATFORM_ITEMS.map((platform) => (
                  <span key={platform}>{platform}</span>
                ))}
              </div>
              <div className="hero-board__task">
                <CheckCircleOutlined />
                <span>{t('login.heroTask')}</span>
              </div>
            </div>
          </div>
        </div>

        <div className="login-right">
          <div className="login-right-inner">
            <div className="login-locale-bar">
              <LocaleSwitch size="small" />
            </div>
            <div className="mobile-brand">
              <BrandLogo height={28} />
              <div>
                <div className="brand-text">{t('login.brand')}</div>
                <div className="brand-sub">{t('login.brandSub')}</div>
              </div>
            </div>

            <div className={`auth-card auth-card-${activeTab}`}>
              <Tabs
                className="auth-tabs"
                activeKey={activeTab}
                centered
                onChange={changeAuthTab}
                items={[
                  { key: 'login', label: t('login.tabLogin') },
                  { key: 'register', label: t('login.tabRegister') },
                ]}
              />

              <div className="welcome-text" key={`welcome-${activeTab}`}>
                <h2>
                  {activeTab === 'login'
                    ? t('login.welcomeBack')
                    : t('login.welcomeRegister')}
                </h2>
                <p>
                  {activeTab === 'login'
                    ? t('login.welcomeLoginHint')
                    : t('login.welcomeRegisterHint')}
                </p>
              </div>

            {activeTab === 'login' ? (
              <Form
                form={loginForm}
                layout="vertical"
                onFinish={onLogin}
                requiredMark={false}
                autoComplete="off"
              >
                <Form.Item
                  name="account"
                  label={t('login.account')}
                  rules={[{ required: true, message: t('login.accountRequired') }]}
                  validateTrigger="onBlur"
                >
                  <Input
                    placeholder={t('login.accountPlaceholder')}
                    prefix={<MailOutlined />}
                    autoComplete="off"
                    data-lpignore="true"
                    data-1p-ignore="true"
                  />
                </Form.Item>

                <Form.Item
                  name="password"
                  label={t('login.password')}
                  rules={[{ required: true, message: t('login.passwordRequired') }]}
                  validateTrigger="onBlur"
                >
                  <Input.Password
                    placeholder={t('login.passwordPlaceholder')}
                    prefix={<LockOutlined />}
                    autoComplete="new-password"
                    data-lpignore="true"
                    data-1p-ignore="true"
                  />
                </Form.Item>

                <div className="form-actions">
                  <Form.Item name="remember" valuePropName="checked" noStyle initialValue={true}>
                    <Checkbox>{t('login.rememberMe')}</Checkbox>
                  </Form.Item>
                  <a href="#" className="forgot-link" onClick={(e) => e.preventDefault()}>
                    {t('login.forgotPassword')}
                  </a>
                </div>

                <Form.Item>
                  <Button
                    type="primary"
                    htmlType="submit"
                    className="submit-btn"
                    loading={loading}
                    disabled={loading}
                  >
                    {t('login.submitLogin')}
                    <ArrowRightOutlined />
                  </Button>
                </Form.Item>

                <div className="register-link">
                  {t('login.noAccount')}
                  <Link to={`/user/register${location.search}`}>
                    {t('login.registerNow')}
                  </Link>
                </div>
              </Form>
            ) : (
              <Form
                form={registerForm}
                layout="vertical"
                onFinish={onRegister}
                requiredMark={false}
              >
                <Form.Item
                  name="email"
                  label={t('login.email')}
                  rules={[
                    { required: true, message: t('login.emailRequired') },
                    { type: 'email', message: t('login.emailInvalid') },
                  ]}
                  validateTrigger="onBlur"
                >
                  <Input
                    placeholder={t('login.emailPlaceholder')}
                    prefix={<MailOutlined />}
                    autoComplete="email"
                  />
                </Form.Item>

                <Form.Item label={t('login.code')} required>
                  <Row gutter={8}>
                    <Col span={15}>
                      <Form.Item
                        name="code"
                        noStyle
                        rules={[{ required: true, message: t('login.codeRequired') }]}
                      >
                        <Input placeholder={t('login.codePlaceholder')} />
                      </Form.Item>
                    </Col>
                    <Col span={9}>
                      <Button
                        className="code-btn"
                        onClick={handleSendCode}
                        disabled={countdown > 0}
                      >
                        {countdown > 0
                          ? t('login.resendIn', { values: { seconds: countdown } })
                          : t('login.sendCode')}
                      </Button>
                    </Col>
                  </Row>
                </Form.Item>

                <Form.Item
                  name="password"
                  label={t('login.password')}
                  rules={[
                    { required: true, message: t('login.passwordRequired') },
                    { min: 6, message: t('login.passwordMin') },
                  ]}
                  validateTrigger="onBlur"
                >
                  <Input.Password
                    placeholder={t('login.passwordMinPlaceholder')}
                    prefix={<LockOutlined />}
                    autoComplete="new-password"
                  />
                </Form.Item>

                <Form.Item
                  name="confirmPassword"
                  label={t('login.confirmPassword')}
                  dependencies={['password']}
                  validateTrigger="onBlur"
                  rules={[
                    { required: true, message: t('login.confirmPasswordRequired') },
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!value || getFieldValue('password') === value) {
                          return Promise.resolve();
                        }
                        return Promise.reject(new Error(t('login.confirmPasswordMismatch')));
                      },
                    }),
                  ]}
                >
                  <Input.Password
                    placeholder={t('login.confirmPasswordPlaceholder')}
                    prefix={<LockOutlined />}
                    autoComplete="new-password"
                  />
                </Form.Item>

                <Form.Item>
                  <Button
                    type="primary"
                    htmlType="submit"
                    className="submit-btn"
                    loading={loading}
                    disabled={loading}
                  >
                    {t('login.submitRegister')}
                    <ArrowRightOutlined />
                  </Button>
                </Form.Item>

                <div className="register-link">
                  {t('login.hasAccount')}
                  <Link to={`/user/login${location.search}`}>
                    {t('login.goLogin')}
                  </Link>
                </div>
              </Form>
            )}

            <div className="agreement">
              {activeTab === 'login'
                ? t('login.agreementLogin')
                : t('login.agreementRegister')}
            </div>
          </div>
        </div>
      </div>
    </div>
    </div>
  );
}
