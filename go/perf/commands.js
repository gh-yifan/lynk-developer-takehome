import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const getLatency = new Trend('commands_get_latency', true);
const postLatency = new Trend('commands_post_latency', true);
const patchLatency = new Trend('commands_patch_latency', true);
const deleteLatency = new Trend('commands_delete_latency', true);
const completedCommands = new Counter('commands_completed');

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const queueSize = Number(__ENV.QUEUE_SIZE || 0);
const queueTag = { queue_size: String(queueSize) };
const smokeProfile = __ENV.K6_PROFILE === 'smoke';
const profilingProfile = __ENV.K6_PROFILE === 'profile';
const throughputProfile = __ENV.K6_PROFILE === 'throughput';
const throughputVUs = Number(__ENV.THROUGHPUT_VUS || 20);
const throughputDuration = __ENV.THROUGHPUT_DURATION || '60s';

export const options = {
  stages: smokeProfile
    ? [
        { duration: '2s', target: 2 },
        { duration: '5s', target: 2 },
        { duration: '2s', target: 0 },
      ]
    : throughputProfile
      ? [
          { duration: '5s', target: throughputVUs },
          { duration: throughputDuration, target: throughputVUs },
          { duration: '5s', target: 0 },
        ]
      : profilingProfile
        ? [
            { duration: '5s', target: 5 },
            { duration: '60s', target: 5 },
            { duration: '5s', target: 0 },
          ]
        : [
            { duration: '30s', target: 5 },
            { duration: '1m', target: 20 },
            { duration: '2m', target: 20 },
            { duration: '30s', target: 0 },
          ],
  thresholds: {
    commands_get_latency: ['p(95)<500'],
    commands_post_latency: ['p(95)<500'],
    commands_patch_latency: ['p(95)<500'],
    commands_delete_latency: ['p(95)<500'],
  },
};

const jsonParams = {
  headers: { 'Content-Type': 'application/json' },
  timeout: '10s',
};

export function setup() {
  if (!Number.isInteger(queueSize) || queueSize < 0) {
    throw new Error(`QUEUE_SIZE must be a non-negative integer, got ${queueSize}`);
  }

  for (let index = 0; index < queueSize; index += 1) {
    const response = http.post(
      `${baseURL}/commands`,
      JSON.stringify({
        command: `k6 baseline ${index} ${Date.now()}`,
        priority: 'LOW',
      }),
      jsonParams,
    );
    if (response.status !== 201) {
      throw new Error(`Failed to seed queue item ${index + 1}/${queueSize}: HTTP ${response.status}`);
    }
  }

  return { queueSize };
}

export default function (data) {
  const suffix = `${__VU}-${__ITER}-${Date.now()}`;
  const commandText = `k6 performance test ${suffix}`;
  let commandID;

  const createResponse = http.post(
    `${baseURL}/commands`,
    JSON.stringify({ command: commandText, priority: 'LOW' }),
    jsonParams,
  );
  postLatency.add(createResponse.timings.duration, queueTag);
  const created = check(createResponse, {
    'POST /commands returns 201': (response) => response.status === 201,
    'POST response includes command ID': (response) => {
      try {
        commandID = response.json('command.id');
        return typeof commandID === 'string' && commandID.length > 0;
      } catch (_) {
        return false;
      }
    },
  });
  if (!created || !commandID) {
    sleep(0.1);
    return;
  }

  const patchExecuting = http.patch(
    `${baseURL}/commands/${commandID}/status`,
    JSON.stringify({ status: 'EXECUTING' }),
    jsonParams,
  );
  patchLatency.add(patchExecuting.timings.duration, queueTag);
  if (!check(patchExecuting, {
    'PATCH /commands/{id}/status accepts EXECUTING': (response) => response.status === 200,
  })) {
    cleanupCommand(commandID);
    sleep(0.1);
    return;
  }

  const getResponse = http.get(`${baseURL}/commands`);
  getLatency.add(getResponse.timings.duration, queueTag);
  let lifecycleSucceeded = check(getResponse, {
    'GET /commands returns 200 and includes command': (response) => {
      if (response.status !== 200) return false;
      try {
        return response.json('commands').some((command) => command.id === commandID);
      } catch (_) {
        return false;
      }
    },
  });

  const terminalStatus = (__ITER + data.queueSize) % 2 === 0 ? 'COMPLETED' : 'FAILED';
  const patchTerminal = http.patch(
    `${baseURL}/commands/${commandID}/status`,
    JSON.stringify({ status: terminalStatus }),
    jsonParams,
  );
  patchLatency.add(patchTerminal.timings.duration, queueTag);
  const terminalSucceeded = check(patchTerminal, {
    [`PATCH /commands/{id}/status accepts ${terminalStatus}`]: (response) => {
      if (response.status !== 200) return false;
      try {
        return response.json('command.status') === terminalStatus;
      } catch (_) {
        return false;
      }
    },
  });
  lifecycleSucceeded = lifecycleSucceeded && terminalSucceeded;

  const deleted = cleanupCommand(commandID);
  if (lifecycleSucceeded && deleted) {
    completedCommands.add(1, queueTag);
  }
  sleep(0.1);
}

function cleanupCommand(commandID) {
  const response = http.del(`${baseURL}/commands/${commandID}`);
  deleteLatency.add(response.timings.duration, queueTag);
  return check(response, {
    'DELETE /commands/{id} returns 200': (result) => result.status === 200,
  });
}
