import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ProColumns } from '@ant-design/pro-components';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import TmProTable from '../TmProTable';

type TableRow = {
  id: string;
  name: string;
};

const columns: ProColumns<TableRow>[] = [
  { title: '名称', dataIndex: 'name' },
];

describe('TmProTable', () => {
  beforeEach(() => {
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

  it('opens the density and column setting controls from accessible buttons', async () => {
    const user = userEvent.setup();
    render(
      <TmProTable<TableRow>
        rowKey="id"
        search={false}
        pagination={false}
        columns={columns}
        dataSource={[{ id: 'row-1', name: '测试记录' }]}
      />,
    );

    expect(screen.queryByText('宽松')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '表格密度' }));
    expect(await screen.findByText('宽松')).toBeInTheDocument();

    await user.keyboard('Escape');
    await waitFor(() => {
      expect(screen.queryByText('宽松')).not.toBeVisible();
    });

    await user.click(screen.getByRole('button', { name: '列设置' }));
    expect(await screen.findByText('列展示')).toBeInTheDocument();
  });
});
