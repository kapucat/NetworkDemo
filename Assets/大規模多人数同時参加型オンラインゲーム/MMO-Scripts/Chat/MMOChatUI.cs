using System.Collections.Generic;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using System.Collections;

public class MMOChatUI : MonoBehaviour
{
    [Header("チャットUI")]
    [SerializeField]
    private TMP_InputField chatInputField;

    [SerializeField]
    private TMP_Text chatLogText;

    [SerializeField]
    private Button sendButton;

    [SerializeField]
    private ScrollRect chatScrollRect;

    [Header("通信")]
    [SerializeField]
    private MMOWebSocketClient webSocketClient;

    [Header("表示設定")]
    [SerializeField]
    private int maxMessageCount = 50;

    private readonly List<string> messages = new();

    private Coroutine scrollCoroutine;

    [SerializeField]
    private float sendInterval = 0.15f;

    private float nextSendTime;



    private void Awake()
    {
        if (chatLogText != null)
        {
            chatLogText.text = "";
            chatLogText.richText = false;
        }
    }

    private void OnEnable()
    {
        if (webSocketClient != null)
        {
            webSocketClient.OnChatMessageReceived +=
                HandleChatMessageReceived;
        }

        if (chatInputField != null)
        {
            chatInputField.onSubmit.AddListener(
                HandleInputSubmit);
        }
    }

    private void OnDisable()
    {
        if (webSocketClient != null)
        {
            webSocketClient.OnChatMessageReceived -=
                HandleChatMessageReceived;
        }

        if (chatInputField != null)
        {
            chatInputField.onSubmit.RemoveListener(
                HandleInputSubmit);
        }
    }

    public void OnSendButtonClicked()
    {
        if (Time.unscaledTime < nextSendTime)
        {
            return;
        }

        if (chatInputField == null)
        {
            return;
        }

        string message = chatInputField.text.Trim();

        if (string.IsNullOrWhiteSpace(message))
        {
            return;
        }

        if (message.Length > 200)
        {
            AppendSystemMessage(
                "メッセージは200文字以内で入力してください");
            return;
        }

        if (webSocketClient == null ||
            !webSocketClient.IsConnected)
        {
            AppendSystemMessage(
                "サーバーへ接続されていません");
            return;
        }

        nextSendTime =
            Time.unscaledTime + sendInterval;

        webSocketClient.SendChat(message);

        chatInputField.text = "";
        chatInputField.ActivateInputField();
    }
    private void HandleChatMessageReceived(
        string loginName,
        string message,
        string sentAt)
    {
        string displayName =
            string.IsNullOrWhiteSpace(loginName)
                ? "Unknown"
                : loginName;

        AppendMessage(
            $"{displayName}：{message}");
    }

    private void AppendSystemMessage(string message)
    {
        AppendMessage($"[SYSTEM] {message}");
    }

    private void AppendMessage(string message)
    {
        bool shouldAutoScroll = IsNearBottom();

        messages.Add(message);

        while (messages.Count > maxMessageCount)
        {
            messages.RemoveAt(0);
        }

        if (chatLogText != null)
        {
            chatLogText.text =
                string.Join("\n", messages);
        }

        if (shouldAutoScroll &&
            chatScrollRect != null)
        {
            if (scrollCoroutine != null)
            {
                StopCoroutine(scrollCoroutine);
            }

            scrollCoroutine =
                StartCoroutine(ScrollToBottom());
        }
    }
    private void HandleInputSubmit(string inputText)
    {
        OnSendButtonClicked();
    }

    private bool IsNearBottom()
    {
        if (chatScrollRect == null ||
            chatScrollRect.content == null ||
            chatScrollRect.viewport == null)
        {
            return true;
        }

        bool contentFits =
            chatScrollRect.content.rect.height <=
            chatScrollRect.viewport.rect.height + 1f;

        return contentFits ||
               chatScrollRect.verticalNormalizedPosition <= 0.05f;
    }

    private IEnumerator ScrollToBottom()
    {
        // TMPとLayout Groupのサイズ計算を待つ
        yield return null;

        Canvas.ForceUpdateCanvases();

        LayoutRebuilder.ForceRebuildLayoutImmediate(
            chatScrollRect.content);

        Canvas.ForceUpdateCanvases();

        chatScrollRect.verticalNormalizedPosition = 0f;
        scrollCoroutine = null;
    }

}