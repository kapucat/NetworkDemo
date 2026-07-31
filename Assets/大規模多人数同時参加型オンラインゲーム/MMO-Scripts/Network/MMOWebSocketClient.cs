using System;
using System.Text;
using NativeWebSocket;
using UnityEngine;

public class MMOWebSocketClient : MonoBehaviour
{
    [SerializeField]
    private string serverUrl = "ws://localhost:8080/ws";

    private WebSocket websocket;

    [Serializable]
    private class ServerMessage
    {
        public string type;
        public string login_name;
        public string message;
        public string sent_at;
    }

    private async void Start()
    {
        Application.runInBackground = true;

        websocket = new WebSocket(serverUrl);

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

            Debug.Log(
                $"WebSocket受信: {json}");

            try
            {
                ServerMessage serverMessage =
                    JsonUtility.FromJson<ServerMessage>(json);

                Debug.Log(
                    $"種類: {serverMessage.type} / " +
                    $"内容: {serverMessage.message}");
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

    private async void OnDestroy()
    {
        if (websocket == null)
        {
            return;
        }

        if (websocket.State == WebSocketState.Open ||
            websocket.State == WebSocketState.Connecting)
        {
            await websocket.Close();
        }
    }
}