using System;
using System.Collections.Generic;
using System.Collections.Concurrent;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using NativeWebSocket;
using UnityEngine;
using UnityEngine.SceneManagement;

public class MMOWebSocketClient : MonoBehaviour
{
    private WebSocket websocket;
    private readonly ConcurrentQueue<Action> roomMainThread = new();
    private readonly Dictionary<long, string> roomPlayers = new();
    public MMORoomState Room { get; private set; }
    public bool CanMove => IsConnected && Room != null && Room.phase == "playing";
    public event Action<MMORoomPacket> OnRoomMessage;
    public event Action OnWorldReset;

    private void Awake() {
        if (GetComponent<MMORoomLobby>() == null) gameObject.AddComponent<MMORoomLobby>();
    }
    private void ResetRoom() {
        Room = null; roomPlayers.Clear(); OnWorldReset?.Invoke();
        OnRoomMessage?.Invoke(new MMORoomPacket { type = "room.reset" });
    }
    public async void SendRoom(string type, string message = null, string roomId = null, bool ready = false) {
        if (!IsConnected) return;
        var packet = new MMORoomPacket { type = type, message = message,
            room_id = roomId ?? Room?.id, ready = ready,
            client_unix_ms = type == "room.ping" ? (long)(Time.realtimeSinceStartupAsDouble * 1000) : 0 };
        try { await websocket.SendText(JsonUtility.ToJson(packet)); }
        catch (Exception) { OnRoomMessage?.Invoke(new MMORoomPacket { type = "room.error", message = "送信できませんでした。接続を確認してください。" }); }
    }
    private void ReceiveRoom(MMORoomPacket packet) {
        if (packet.type == "room.left") {
            if (Room != null && Room.id == packet.room_id) ResetRoom();
        }
        if (packet.type == "room.state" && packet.room != null) {
            if (Room == null || Room.id != packet.room.id) { roomPlayers.Clear(); OnWorldReset?.Invoke(); }
            Room = packet.room;
            var members = new HashSet<long>();
            foreach (var member in Room.members ?? Array.Empty<MMORoomMember>()) members.Add(member.account_id);
            foreach (var id in new List<long>(roomPlayers.Keys)) {
                if (!members.Contains(id)) { OnPlayerLeft?.Invoke(id, roomPlayers[id]); roomPlayers.Remove(id); }
            }
            foreach (var member in Room.members ?? Array.Empty<MMORoomMember>()) {
                if (!roomPlayers.ContainsKey(member.account_id)) {
                    roomPlayers.Add(member.account_id, member.login_name);
                    OnPlayerJoined?.Invoke(member.account_id, member.login_name);
                }
            }
        }
        OnRoomMessage?.Invoke(packet);
    }

    private bool isConnecting;
    private bool reconnectScheduled;
    private volatile bool isShuttingDown;
    private volatile bool isAdminDisconnected;
    private int returnToLoginPending;

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
        !isShuttingDown &&
        !isAdminDisconnected &&
        websocket != null &&
        websocket.State == WebSocketState.Open;

    [Serializable]
    private class WebSocketMessage
    {
        public string type;
        public string room_id;
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
            isAdminDisconnected ||
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
                Uri.EscapeDataString(sessionToken) + "&rooms=1";

            WebSocket activeSocket =
                new WebSocket(authenticatedUrl);
            websocket = activeSocket;

            activeSocket.OnOpen += () =>
            {
                if (!ReferenceEquals(websocket, activeSocket) || isShuttingDown || isAdminDisconnected)
                {
                    _ = activeSocket.Close();
                    return;
                }
                reconnectScheduled = false;
                roomMainThread.Enqueue(() => { ResetRoom(); SendRoom("room.list"); SendRoom("room.ping"); });

            };

            activeSocket.OnError += errorMessage =>
            {
                if (!ReferenceEquals(websocket, activeSocket) || isShuttingDown) return;
                Debug.LogError(
                    $"WebSocketエラー: {errorMessage}");
            };

            activeSocket.OnClose += closeCode =>
            {
                if (!ReferenceEquals(websocket, activeSocket) || isShuttingDown) return;

                roomMainThread.Enqueue(ResetRoom);
                // 1008は管理サーバーからの接続終了。通常の通信断と分ける。
                if (closeCode == WebSocketCloseCode.PolicyViolation)
                {
                    isAdminDisconnected = true;
                    isShuttingDown = true;
                    reconnectScheduled = false;
                    Interlocked.Exchange(ref returnToLoginPending, 1);
                    return;
                }

                Debug.LogWarning(
                    $"WebSocket切断: {closeCode}");

                if (!isShuttingDown)
                {
                    _ = ScheduleReconnect();
                }
            };

            activeSocket.OnMessage += message =>
            {
                if (ReferenceEquals(websocket, activeSocket) && !isShuttingDown)
                    roomMainThread.Enqueue(() => HandleWebSocketMessage(message));
            };

            await activeSocket.Connect();
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
            isAdminDisconnected ||
            reconnectScheduled)
        {
            return;
        }

        reconnectScheduled = true;


        await Task.Delay(
            ReconnectDelayMilliseconds);

        reconnectScheduled = false;

        if (isShuttingDown || isAdminDisconnected)
        {
            return;
        }

        await ConnectWebSocket();
    }

    // シーン切替とUnityオブジェクトの操作は必ずメインスレッドで行う。
    private void Update()
    {
        while (roomMainThread.TryDequeue(out var action)) action();
        if (Interlocked.Exchange(ref returnToLoginPending, 0) == 0) return;

        if (MMOSession.Instance != null)
            MMOSession.Instance.Clear();

        Debug.LogWarning("サーバーから接続を終了されました。再ログインしてください。");
        SceneManager.LoadScene("LoginScene");
    }

    private void HandleWebSocketMessage(
        byte[] messageBytes)
    {
        string json =
            Encoding.UTF8.GetString(messageBytes);


        try
        {
            WebSocketMessage receivedMessage =
                JsonUtility.FromJson<WebSocketMessage>(
                    json);

            if (receivedMessage == null)
            {
                return;
            }


            if (receivedMessage.type != null && receivedMessage.type.StartsWith("room.")) {
                ReceiveRoom(JsonUtility.FromJson<MMORoomPacket>(json)); return;
            }
            if (receivedMessage.type == "player.move" || receivedMessage.type == "chat" || receivedMessage.type == "chat.message") {
                if (Room == null || receivedMessage.room_id != Room.id) return;
            }
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
                room_id = Room?.id,
                message = message,
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
            Debug.LogError(
                $"WebSocket送信失敗: " +
                $"{exception.Message}");
        }
    }

    public async void SendPlayerMove(
        Vector3 position,
        float rotationY)
    {
        if (!CanMove)
        {
            return;
        }

        WebSocketMessage sendMessage =
            new WebSocketMessage
            {
                type = "player.move",
                room_id = Room?.id,
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