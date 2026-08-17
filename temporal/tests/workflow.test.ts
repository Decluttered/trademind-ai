import { Worker } from '@temporalio/worker';
import { fileURLToPath } from 'node:url';
import { TestWorkflowEnvironment } from '@temporalio/testing';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { PublicationActivities } from '../src/contracts.js';
import { PublishListingWorkflow } from '../src/workflows.js';

describe('PublishListingWorkflow', () => {
  let environment: TestWorkflowEnvironment | undefined;

  afterEach(async () => {
    await environment?.teardown();
    environment = undefined;
  });

  it('retries a transient publish activity and reconciles only after success', async () => {
    environment = await TestWorkflowEnvironment.createTimeSkipping();
    const calls: string[] = [];
    let publishAttempts = 0;
    const activities: PublicationActivities = {
      revalidateListing: vi.fn(async () => { calls.push('revalidate'); }),
      publishListing: vi.fn(async () => {
        calls.push('publish');
        publishAttempts += 1;
        if (publishAttempts === 1) throw new Error('transient provider failure');
      }),
      reconcileListing: vi.fn(async () => { calls.push('reconcile'); }),
    };
    const worker = await Worker.create({
      connection: environment.nativeConnection,
      taskQueue: 'test-publication',
      workflowsPath: fileURLToPath(new URL('../src/workflows.ts', import.meta.url)),
      activities,
    });

    await worker.runUntil(async () => {
      const result = await environment!.client.workflow.execute(PublishListingWorkflow, {
        workflowId: 'publish-listing-test',
        taskQueue: 'test-publication',
        args: [{ workspaceId: 7, publicationId: '00000000-0000-0000-0000-000000000001' }],
      });
      expect(result).toBeUndefined();
    });
    expect(calls).toEqual(['revalidate', 'publish', 'publish', 'reconcile']);
  }, 30_000);
});
