using System;
using System.Text;
using NativeWebSocket;
using UnityEngine;

public class MMOWebSocketClient : MonoBehaviour
{
    [SerializeField]
    private string serverUrl = "ws://localhost:8080/ws";

    private WebSocket websocket;

    public event Action<string, string, string>
        OnChatMessageReceived;
    public event Action<long, string>
    OnPlayerJoined;

    public event Action<long, string>
        OnPlayerLeft;

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

        string sessionToken =
            MMOSession.Instance.SessionToken;

        string separator =
            serverUrl.Contains("?") ? "&" : "?";

        string authenticatedUrl =
            serverUrl +
            separator +
            "token=" +
            Uri.EscapeDataString(sessionToken);

        websocket =
            new WebSocket(authenticatedUrl);
        websocket.OnOpen += () =>
        {
            Debug.Log("WebSocket接続成功");
        };

        websocket.OnError += errorMessage =>
        {
            Debug.LogError(
                $"WebSocketエラー: {errorMessage}");
        };

        websocket.OnClose += closeCode =>
        {
            Debug.Log(
                $"WebSocket切断: {closeCode}");
        };







        websocket.OnMessage += messageBytes =>
        {
            string json =
                Encoding.UTF8.GetString(messageBytes);

            Debug.Log($"WebSocket受信: {json}");

            try
            {
                WebSocketMessage receivedMessage =
                    JsonUtility.FromJson<WebSocketMessage>(json);





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
        };

        try
        {
            await websocket.Connect();
        }
        catch (Exception exception)
        {
            Debug.LogError(
                $"WebSocket接続処理に失敗しました: " +
                $"{exception.Message}");
        }
    }

    public async void SendChat(string message)
    {
        if (!IsConnected)
        {
            Debug.LogWarning(
                "WebSocketへ接続されていません");
            return;
        }

        string loginName = "";

        if (MMOSession.Instance != null)
        {
            loginName = MMOSession.Instance.LoginName;
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
                $"WebSocket送信失敗: {exception.Message}");
        }
    }

    private async void OnDestroy()
    {
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