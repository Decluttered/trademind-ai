import { NativeConnection, Worker } from '@temporalio/worker';
import { fileURLToPath } from 'node:url';
import { createActivities } from './activities.js';

const address = process.env.TEMPORAL_ADDRESS?.trim() || 'localhost:7233';
const apiBaseUrl = process.env.API_INTERNAL_BASE_URL?.trim() || 'http://localhost:8080';
const serviceToken = process.env.TEMPORAL_SERVICE_TOKEN?.trim() || '';

if (!serviceToken) {
  throw new Error('TEMPORAL_SERVICE_TOKEN is required');
}

const connection = await NativeConnection.connect({ address });
const worker = await Worker.create({
  connection,
  namespace: process.env.TEMPORAL_NAMESPACE?.trim() || 'default',
  taskQueue: 'mindbay-publication',
  workflowsPath: fileURLToPath(new URL('./workflows.ts', import.meta.url)),
  activities: createActivities({ apiBaseUrl, serviceToken }),
});

await worker.run();
