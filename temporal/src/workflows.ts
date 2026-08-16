import { proxyActivities } from '@temporalio/workflow';
import type { PublicationActivities, PublishListingInput } from './contracts.js';

const activities = proxyActivities<PublicationActivities>({
  startToCloseTimeout: '45 seconds',
  heartbeatTimeout: '15 seconds',
  retry: {
    initialInterval: '2 seconds',
    backoffCoefficient: 2,
    maximumInterval: '30 seconds',
    maximumAttempts: 3,
  },
});

// Activities call versioned Go commands only. The workflow contains no DB,
// eBay, clock, random, or browser access and is therefore deterministic.
export async function PublishListingWorkflow(input: PublishListingInput): Promise<void> {
  await activities.revalidateListing(input);
  await activities.publishListing(input);
  await activities.reconcileListing(input);
}
