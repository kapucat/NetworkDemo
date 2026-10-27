import http from 'k6/http';
import { check } from 'k6';
import ws from 'k6/ws';
import { Counter } from 'k6/metrics';

// 負荷試験専用環境だけを使う（通常の8080/VPSには送らない）。
const BASE_URL = 'http://localhost:8081';
const WS_URL = 'ws://localhost:8081/ws';
const PASSWORD = 'mmo_bot_password';
const MOVE_INTERVAL_MS = 100;
const SESSION_MS = 60000;
const SEND_UNTIL_MS = 59000; // 最後の1秒は受信待ちに使う。

const moveSent = new Counter('mmo_moves_sent');
const moveReceived = new Counter('mmo_moves_received');
const selfMoveReceived = new Counter('mmo_self_moves_received');
const invalidMoves = new Counter('mmo_invalid_moves');
const wsErrors = new Counter('mmo_ws_errors');
const unexpectedDisconnects = new Counter('mmo_unexpected_disconnects');

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    checks: ['rate==1'],
    mmo_invalid_moves: ['count==0'],
    mmo_ws_errors: ['count==0'],
    mmo_unexpected_disconnects: ['count==0'],
  },
};

export default function () {
  // エラー0件でも閾値を評価できるよう、ゼロ件を明示する。
  invalidMoves.add(0);
  wsErrors.add(0);
  unexpectedDisconnects.add(0);

  const loginName = `mmo_bot_${__VU}`;
  const headers = { 'Content-Type': 'application/json' };

  const registered = http.post(
    `${BASE_URL}/api/accounts/register`,
    JSON.stringify({ login_name: loginName, password: PASSWORD }),
    {
      headers,
      responseCallback: http.expectedStatuses(201, 409),
    }
  );
  check(registered, {
    'register 201 or 409': (r) => r.status === 201 || r.status === 409,
  });

  const login = http.post(
    `${BASE_URL}/api/accounts/login`,
    JSON.stringify({ login_name: loginName, password: PASSWORD }),
    { headers }
  );
  const loginOk = check(login, {
    'login 200': (r) => r.status === 200,
  });
  if (!loginOk) {
    console.error(`VU ${__VU}: login failed, status=${login.status}`);
    return;
  }

  const loginData = login.json();
  const token = loginData.session_token;
  const accountId = Number(loginData.account_id);
  if (!token || !Number.isFinite(accountId)) {
    console.error(`VU ${__VU}: session_token or account_id missing`);
    check(false, { 'valid login data': (ok) => ok });
    return;
  }

  let openedAt = 0;
  let closedAt = 0;
  let lastSentAt = 0;
  let sent = 0;
  let ownEchoes = 0;
  let errors = 0;
  let invalid = 0;
  let plannedClose = false;
  let earlyClose = false;

  const url = `${WS_URL}?token=${encodeURIComponent(token)}`;
  const response = ws.connect(url, {}, function (socket) {
    socket.on('open', function () {
      openedAt = Date.now();
      console.log(`VU ${__VU}: connected, account_id=${accountId}`);

      socket.setInterval(function () {
        const now = Date.now();

if (plannedClose ||
    now - openedAt < 3000 ||
    now - openedAt >= SEND_UNTIL_MS ||
    now - lastSentAt < 90) {
  return;
}


        // 0.05m / 0.1秒 = 約0.5m/s。60秒でも±1000に収まる。
        const nextX = Number(((sent + 1) * 0.05).toFixed(3));
        try {
          socket.send(JSON.stringify({
            type: 'player.move',
            x: nextX,
            y: 0,
            z: 0,
            rotation_y: 0,
            sent_at: new Date(now).toISOString(),
          }));

          sent += 1;
          lastSentAt = now;
          moveSent.add(1);
        } catch (e) {
          errors += 1;
          wsErrors.add(1);
          console.error(`VU ${__VU}: move send failed: ${e}`);
          socket.close();
        }
      }, MOVE_INTERVAL_MS);

      socket.setTimeout(function () {
        plannedClose = true;
        socket.close();
      }, SESSION_MS);
    });

    socket.on('message', function (raw) {
      try {
        const msg = JSON.parse(raw);
        if (msg.type === 'player.move') {
          moveReceived.add(1);
          if (Number(msg.account_id) === accountId) {
            ownEchoes += 1;
            selfMoveReceived.add(1);
          }
        } else if (msg.type === 'error') {
          errors += 1;
          if (msg.message === 'invalid player move') {
            invalid += 1;
            invalidMoves.add(1);
          } else {
            wsErrors.add(1);
          }
          console.error(`VU ${__VU}: server error: ${msg.message}`);
        }
      } catch (e) {
        errors += 1;
        wsErrors.add(1);
        console.error(`VU ${__VU}: invalid received JSON: ${e}`);
      }
    });

    socket.on('error', function (e) {
      const detail = typeof e.error === 'function' ? e.error() : String(e);
      if (plannedClose && detail.includes('websocket: close sent')) {
        return;
      }
      errors += 1;
      wsErrors.add(1);
      console.error(`VU ${__VU}: WebSocket error: ${detail}`);
    });

    socket.on('close', function () {
      closedAt = Date.now();
      if (!plannedClose) {
        earlyClose = true;
        unexpectedDisconnects.add(1);
        console.error(`VU ${__VU}: unexpected early disconnect`);
      }
      console.log(`VU ${__VU}: sent=${sent}, self_echo=${ownEchoes}, invalid=${invalid}, errors=${errors}`);
    });
  });

  const elapsedMs = closedAt && openedAt ? closedAt - openedAt : 0;
  check(response, { 'WebSocket 101': (r) => !!r && r.status === 101 });
  check({ elapsedMs, sent, ownEchoes, invalid, errors, plannedClose, earlyClose }, {
    '60s connection maintained': (s) =>
      s.plannedClose && !s.earlyClose && s.elapsedMs >= SESSION_MS - 100,
    'moves sent': (s) => s.sent > 0,
    'own move broadcast received': (s) => s.ownEchoes > 0,
    'all own moves broadcast received': (s) => s.ownEchoes === s.sent,
    'no invalid player move': (s) => s.invalid === 0,
    'no WebSocket errors': (s) => s.errors === 0,
  });
  console.log(`VU ${__VU}: duration=${elapsedMs}ms, sent=${sent}, self_echo=${ownEchoes}`);
}
