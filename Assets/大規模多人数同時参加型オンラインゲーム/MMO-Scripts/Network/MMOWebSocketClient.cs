using System;
using System.Text;
using System.Threading.Tasks;
using NativeWebSocket;
using UnityEngine;

public class MMOWebSocketClient : MonoBehaviour
{
    private WebSocket websocket;

    private bool isConnecting;
    private bool reconnectScheduled;
    private bool isShuttingDown;

    private const int ReconnectDelayMilliseconds = 3000;

    public event Action<string, string, string>
        OnChatMessageReceived;

    public event Action<long, string>
        OnPlayerJoined;

    public event Action<long, string>
        OnPlayerLeft;

    public event Action<long, string, Vector3, float>
        OnPlayerMoved;

    public bool IsConnected =>
        websocket != null &&
        websocket.State == WebSocketState.Open;

    [Serializable]
    private class WebSocketMessage
    {
        public string type;
        public long account_id;
        public string login_name;
        public string message;
        public string sent_at;
        public float x;
        public float y;
        public float z;
        public float rotation_y;
    }

    private async void Start()
    {
        Application.runInBackground = true;

        if (MMOSession.Instance == null ||
            !MMOSession.Instance.IsLoggedIn)
        {
            Debug.LogError(
                "ログインセッションが存在しないためWebSocketへ接続できません");

            return;
        }

        await ConnectWebSocket();
    }

    private async Task ConnectWebSocket()
    {
        if (isShuttingDown ||
            isConnecting ||
            IsConnected)
        {
            return;
        }

        if (MMOSession.Instance == null ||
            !MMOSession.Instance.IsLoggedIn)
        {
            Debug.LogError(
                "ログインセッションが存在しないためWebSocketへ接続できません");

            return;
        }

        isConnecting = true;

        try
        {
            string sessionToken =
                MMOSession.Instance.SessionToken;

            string serverUrl =
                MMOServerConfig.WebSocketUrl;

            string separator =
                serverUrl.Contains("?")
                    ? "&"
                    : "?";

            string authenticatedUrl =
                serverUrl +
                separator +
                "token=" +
                Uri.EscapeDataString(sessionToken);

            websocket =
                new WebSocket(authenticatedUrl);

            websocket.OnOpen += () =>
            {
                reconnectScheduled = false;

                Debug.Log(
                    "WebSocket接続成功");
            };

            websocket.OnError += errorMessage =>
            {
                Debug.LogError(
                    $"WebSocketエラー: {errorMessage}");
            };

            websocket.OnClose += closeCode =>
            {
                Debug.LogWarning(
                    $"WebSocket切断: {closeCode}");

                if (!isShuttingDown)
                {
                    _ = ScheduleReconnect();
                }
            };

            websocket.OnMessage +=
                HandleWebSocketMessage;

            await websocket.Connect();
        }
        catch (Exception exception)
        {
            Debug.LogError(
                $"WebSocket接続処理に失敗しました: " +
                $"{exception.Message}");

            if (!isShuttingDown)
            {
                _ = ScheduleReconnect();
            }
        }
        finally
        {
            isConnecting = false;
        }
    }

    private async Task ScheduleReconnect()
    {
        if (isShuttingDown ||
            reconnectScheduled)
        {
            return;
        }

        reconnectScheduled = true;

        Debug.Log(
            "3秒後にWebSocket再接続を試します");

        await Task.Delay(
            ReconnectDelayMilliseconds);

        reconnectScheduled = false;

        if (isShuttingDown)
        {
            return;
        }

        await ConnectWebSocket();
    }

    private void HandleWebSocketMessage(
        byte[] messageBytes)
    {
        string json =
            Encoding.UTF8.GetString(messageBytes);

        Debug.Log(
            $"WebSocket受信: {json}");

        try
        {
            WebSocketMessage receivedMessage =
                JsonUtility.FromJson<WebSocketMessage>(
                    json);

            if (receivedMessage == null)
            {
                return;
            }

            Debug.Log(
                $"種類: {receivedMessage.type} / " +
                $"内容: {receivedMessage.message}");

            if (receivedMessage.type == "player.join")
            {
                OnPlayerJoined?.Invoke(
                    receivedMessage.account_id,
                    receivedMessage.login_name);

                return;
            }

            if (receivedMessage.type == "player.leave")
            {
                OnPlayerLeft?.Invoke(
                    receivedMessage.account_id,
                    receivedMessage.login_name);

                return;
            }

            if (receivedMessage.type == "player.move")
            {
                Vector3 position =
                    new Vector3(
                        receivedMessage.x,
                        receivedMessage.y,
                        receivedMessage.z);

                OnPlayerMoved?.Invoke(
                    receivedMessage.account_id,
                    receivedMessage.login_name,
                    position,
                    receivedMessage.rotation_y);

                return;
            }

            if (receivedMessage.type == "chat" ||
                receivedMessage.type == "chat.message")
            {
                OnChatMessageReceived?.Invoke(
                    receivedMessage.login_name,
                    receivedMessage.message,
                    receivedMessage.sent_at);
            }
        }
        catch (Exception exception)
        {
            Debug.LogWarning(
                $"WebSocketメッセージ解析失敗: " +
                $"{exception.Message}");
        }
    }

    public async void SendChat(
        string message)
    {
        if (!IsConnected)
        {
            Debug.LogWarning(
                "WebSocketへ接続されていません");

            return;
        }

        WebSocketMessage sendMessage =
            new WebSocketMessage
            {
                type = "chat",
                message = message,
                sent_at = ""
            };

        string json =
            JsonUtility.ToJson(sendMessage);

        try
        {
            await websocket.SendText(json);

            Debug.Log(
                $"WebSocket送信: {json}");
        }
        catch (Exception exception)
        {
            Debug.LogError(
                $"WebSocket送信失敗: " +
                $"{exception.Message}");
        }
    }

    public async void SendPlayerMove(
        Vector3 position,
        float rotationY)
    {
        if (!IsConnected)
        {
            return;
        }

        WebSocketMessage sendMessage =
            new WebSocketMessage
            {
                type = "player.move",
                x = position.x,
                y = position.y,
                z = position.z,
                rotation_y = rotationY,
                sent_at = ""
            };

        string json =
            JsonUtility.ToJson(sendMessage);

        try
        {
            await websocket.SendText(json);
        }
        catch (Exception exception)
        {
            Debug.LogWarning(
                $"Player移動送信失敗: " +
                $"{exception.Message}");
        }
    }

    private async void OnDestroy()
    {
        isShuttingDown = true;
        reconnectScheduled = false;

        if (websocket == null)
        {
            return;
        }

        if (websocket.State == WebSocketState.Open ||
            websocket.State == WebSocketState.Connecting)
        {
            try
            {
                await websocket.Close();
            }
            catch (Exception exception)
            {
                Debug.LogWarning(
                    $"WebSocket切断処理失敗: " +
                    $"{exception.Message}");
            }
        }
    }
}