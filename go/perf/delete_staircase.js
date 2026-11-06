import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const queueSize = Number(__ENV.QUEUE_SIZE || 20000);
const bucketSize = Number(__ENV.DELETE_BUCKET_SIZE || __ENV.DELETE_SAMPLE_EVERY || 100);
const deleteLatency = new Trend('commands_delete_staircase_latency', true);
const deletedCommands = new Counter('commands_deleted');
let nextDeleteIndex = 0;

export const options = {
  setupTimeout: '10m',
  scenarios: {
    drain_queue: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: Math.max(queueSize, 1),
      maxDuration: __ENV.MAX_DURATION || '30m',
    },
  },
  thresholds: {
    commands_delete_staircase_latency: ['p(95)<500'],
  },
};

const jsonParams = {
  headers: { 'Content-Type': 'application/json' },
  timeout: '30s',
};

export function setup() {
  if (!Number.isInteger(queueSize) || queueSize < 1) {
    throw new Error(`QUEUE_SIZE must be a positive integer, got ${queueSize}`);
  }
  if (!Number.isInteger(bucketSize) || bucketSize < 1) {
    throw new Error(`DELETE_BUCKET_SIZE must be a positive integer, got ${bucketSize}`);
  }

  const before = http.get(`${baseURL}/commands`);
  if (before.status !== 200) {
    throw new Error(`GET /commands before setup returned HTTP ${before.status}`);
  }
  const existing = before.json('commands');
  if (!Array.isArray(existing) || existing.length !== 0) {
    throw new Error('Delete staircase requires an empty in-memory queue; restart the API before running');
  }

  const ids = [];
  for (let index = 0; index < queueSize; index += 1) {
    const response = http.post(
      `${baseURL}/commands`,
      JSON.stringify({
        command: `delete staircase ${index}`,
        priority: 'LOW',
      }),
      jsonParams,
    );
    if (response.status !== 201) {
      throw new Error(`Failed to prepare command ${index + 1}/${queueSize}: HTTP ${response.status}`);
    }

    let id;
    try {
      id = response.json('command.id');
    } catch (_) {
      throw new Error(`Create response ${index + 1} did not contain command.id`);
    }
    if (typeof id !== 'string' || id.length === 0) {
      throw new Error(`Create response ${index + 1} did not contain a valid command.id`);
    }
    ids.push(id);
  }

  const after = http.get(`${baseURL}/commands`);
  if (after.status !== 200 || after.json('commands').length !== queueSize) {
    throw new Error(`Queue preparation did not produce exactly ${queueSize} commands`);
  }

  return { ids, queueSize, bucketSize };
}

export default function (data) {
  if (nextDeleteIndex >= data.ids.length) return;

  const queueDepth = data.queueSize - nextDeleteIndex;
  const id = data.ids[nextDeleteIndex];
  const response = http.del(
    `${baseURL}/commands/${id}`,
    null,
    {
      timeout: '30s',
      tags: { name: 'DELETE /commands/{id}' },
    },
  );
  const succeeded = check(response, {
    'DELETE /commands/{id} returns 200': (result) => result.status === 200,
  });
  if (!succeeded) {
    throw new Error(`Delete failed at queue depth ${queueDepth}: HTTP ${response.status}`);
  }

  const bucketIndex = Math.floor((data.queueSize - queueDepth) / data.bucketSize);
  const bucketStart = data.queueSize - bucketIndex * data.bucketSize;
  const bucketEnd = Math.max(1, bucketStart - data.bucketSize + 1);
  deleteLatency.add(response.timings.duration, {
    queue_depth_band: `${bucketStart}-${bucketEnd}`,
  });
  deletedCommands.add(1);
  nextDeleteIndex += 1;
}

export function teardown(data) {
  const response = http.get(`${baseURL}/commands`);
  if (response.status !== 200 || response.json('commands').length !== 0) {
    throw new Error(`Delete staircase left commands in the queue; expected it to be empty after deleting ${data.queueSize}`);
  }
}
