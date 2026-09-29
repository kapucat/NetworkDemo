using UnityEngine;

public class MMOLocalPlayerNetworkSync : MonoBehaviour
{
    [SerializeField]
    private MMOWebSocketClient webSocketClient;

    [SerializeField]
    private float sendInterval = 0.1f;

    private float nextSendTime;

    private void Update()
    {
        if (webSocketClient == null ||
            !webSocketClient.IsConnected)
        {
            return;
        }

        if (Time.unscaledTime < nextSendTime)
        {
            return;
        }

        nextSendTime =
            Time.unscaledTime + sendInterval;

        webSocketClient.SendPlayerMove(
            transform.position,
            transform.eulerAngles.y);
    }
}