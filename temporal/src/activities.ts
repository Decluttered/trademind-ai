import { ApplicationFailure, Context } from '@temporalio/activity';
import type { PublicationActivities, PublishListingInput } from './contracts.js';

export interface ActivityConfig {
  apiBaseUrl: string;
  serviceToken: string;
}

export function createActivities(config: ActivityConfig): PublicationActivities {
  const command = async (step: 'revalidate' | 'publish' | 'reconcile', input: PublishListingInput) => {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 40_000);
    try {
      Context.current().heartbeat({ step, state: 'started' });
      const response = await fetch(
        `${config.apiBaseUrl.replace(/\/$/, '')}/internal/v1/mindbay/publications/${encodeURIComponent(input.publicationId)}/${step}`,
        {
          method: 'POST',
          headers: {
            authorization: `Bearer ${config.serviceToken}`,
            'content-type': 'application/json',
          },
          body: JSON.stringify({ workspaceId: input.workspaceId }),
          signal: controller.signal,
        },
      );
      if (!response.ok) {
        const message = (await response.text()).slice(0, 500);
        const detail = `publication ${step} failed (${response.status}): ${message}`;
        const terminal = [400, 401, 403, 404, 422].includes(response.status) || (step === 'revalidate' && response.status === 409);
        if (terminal) throw ApplicationFailure.nonRetryable(detail, `HTTP_${response.status}`);
        throw new Error(detail);
      }
      Context.current().heartbeat({ step, state: 'completed' });
    } finally {
      clearTimeout(timeout);
    }
  };

  return {
    revalidateListing: (input) => command('revalidate', input),
    publishListing: (input) => command('publish', input),
    reconcileListing: (input) => command('reconcile', input),
  };
}
