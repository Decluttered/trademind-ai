import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AppTopNav, { resolveUserLabels } from '../AppTopNav';

const user: API.CurrentUser = {
  id: 'test-user',
  username: 'operator@example.test',
  email: 'operator@example.test',
  displayName: '运营账号',
};

beforeEach(() => {
  document.documentElement.scrollTop = 0;
  document.body.scrollTop = 0;
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }),
  });
});

describe('AppTopNav', () => {
  it('shows the current account in the content navigation', () => {
    render(<AppTopNav user={user} onLogout={vi.fn()} />);

    const navigation = screen.getByRole('navigation', { name: '内容导航栏' });
    expect(navigation).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '当前用户 运营账号' })).toBeInTheDocument();
    expect(screen.getByText('operator@example.test')).toBeInTheDocument();
  });

  it('keeps logout available from the account menu', async () => {
    const onLogout = vi.fn();
    const interaction = userEvent.setup();
    render(<AppTopNav user={user} onLogout={onLogout} />);

    const accountTrigger = screen.getByRole('button', { name: '当前用户 运营账号' });
    expect(accountTrigger).toHaveAttribute('aria-expanded', 'false');

    await interaction.click(accountTrigger);
    expect(accountTrigger).toHaveAttribute('aria-expanded', 'true');
    await interaction.click(await screen.findByRole('menuitem', { name: /退出登录/ }));

    expect(onLogout).toHaveBeenCalledTimes(1);
    expect(accountTrigger).toHaveAttribute('aria-expanded', 'false');
  });

  it('switches to the frosted navigation state after the page scrolls', async () => {
    render(<AppTopNav user={user} onLogout={vi.fn()} />);

    const navigation = screen.getByRole('navigation', { name: '内容导航栏' });
    expect(navigation).not.toHaveClass('tm-app-top-nav--scrolled');

    document.documentElement.scrollTop = 24;
    fireEvent.scroll(window);
    await waitFor(() => expect(navigation).toHaveClass('tm-app-top-nav--scrolled'));

    document.documentElement.scrollTop = 0;
    fireEvent.scroll(window);
    await waitFor(() => expect(navigation).not.toHaveClass('tm-app-top-nav--scrolled'));
  });

  it('shortens an email display name while retaining the full account', () => {
    expect(resolveUserLabels({ ...user, displayName: 'operator@example.test' })).toEqual({
      primary: 'operator',
      secondary: 'operator@example.test',
      initial: 'O',
    });
  });
});
