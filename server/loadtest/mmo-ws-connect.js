import http from 'k6/http';
import { check } from 'k6';
import ws from 'k6/ws';

const BASE_URL =
  (__ENV.BASE_URL || 'http://localhost:8081')
    .replace(/\/+$/, '');

const WS_URL =
  (__ENV.WS_URL || 'ws://localhost:8081/ws')
    .replace(/\/+$/, '');

const PASSWORD = 'mmo_bot_password';

export const options = {
  vus: 1,
  iterations: 1,
};

export default function () {
  const loginName = `mmo_bot_${__VU}`;

// Botアカウント作成。
// すでに存在する場合の409も正常扱いにする。
const registerResponse = http.post(
  `${BASE_URL}/api/accounts/register`,
  JSON.stringify({
    login_name: loginName,
    password: PASSWORD,
  }),
  {
    headers: {
      'Content-Type': 'application/json',
    },
    responseCallback: http.expectedStatuses(201, 409),
  }
);
  check(registerResponse, {
    'register succeeded or already exists': (response) =>
      response.status === 201 ||
      response.status === 409,
  });

  // ログインしてsession_tokenを取得。
  const loginResponse = http.post(
    `${BASE_URL}/api/accounts/login`,
    JSON.stringify({
      login_name: loginName,
      password: PASSWORD,
    }),
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  const loginOk = check(loginResponse, {
    'login succeeded': (response) =>
      response.status === 200,
  });

  if (!loginOk) {
    console.error(
      `VU ${__VU}: login failed status=${loginResponse.status}`
    );

    return;
  }

  const loginBody = loginResponse.json();

  const sessionToken =
    loginBody.session_token;

  if (!sessionToken) {
    console.error(
      `VU ${__VU}: session_token not found`
    );

    return;
  }

  const url =
    `${WS_URL}?token=${encodeURIComponent(sessionToken)}`;

  const response = ws.connect(
    url,
    {},
    function (socket) {
      socket.on('open', function () {
        console.log(
          `VU ${__VU}: WebSocket connected`
        );
      });

      socket.on('message', function (message) {
        // connected / player.join 等を受信できればOK。
      });

      socket.on('error', function (error) {
        console.error(
          `VU ${__VU}: WebSocket error: ${error.error()}`
        );
      });

      socket.on('close', function () {
        console.log(
          `VU ${__VU}: WebSocket closed`
        );
      });

      // 60秒接続を維持してから正常切断。
      socket.setTimeout(function () {
        socket.close();
      }, 60000);
    }
  );

  check(response, {
    'WebSocket status is 101': (r) =>
      r && r.status === 101,
  });
}