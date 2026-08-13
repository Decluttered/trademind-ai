import { describe, expect, it } from 'vitest';
import {
  operationLogActionLabel,
  operationLogResourceLabel,
} from '../operationLogs';

describe('operation log alert labels', () => {
  it('keeps system alert audit records user-facing', () => {
    expect(operationLogResourceLabel('alert_event')).toBe('系统告警');
    expect(operationLogActionLabel('alert.acknowledge')).toBe('确认系统告警');
    expect(operationLogActionLabel('alert.silence')).toBe('静默系统告警');
  });
});
