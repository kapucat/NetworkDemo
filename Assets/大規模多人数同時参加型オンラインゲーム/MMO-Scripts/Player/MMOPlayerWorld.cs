using System.Collections.Generic;
using UnityEngine;

public class MMOPlayerWorld : MonoBehaviour
{
    [Header("通信")]
    [SerializeField]
    private MMOWebSocketClient webSocketClient;

    [Header("プレイヤー")]
    [SerializeField]
    private Transform localPlayer;

    [SerializeField]
    private GameObject remotePlayerPrefab;

    [SerializeField]
    private Transform remotePlayersRoot;

    private readonly Dictionary<long, GameObject>
        remotePlayers = new();

    private void OnEnable()
    {
        if (webSocketClient == null)
        {
            return;
        }

        webSocketClient.OnPlayerJoined +=
            HandlePlayerJoined;

        webSocketClient.OnPlayerLeft +=
            HandlePlayerLeft;

        webSocketClient.OnPlayerMoved +=
            HandlePlayerMoved;
    }

    private void OnDisable()
    {
        if (webSocketClient == null)
        {
            return;
        }

        webSocketClient.OnPlayerJoined -=
            HandlePlayerJoined;

        webSocketClient.OnPlayerLeft -=
            HandlePlayerLeft;

        webSocketClient.OnPlayerMoved -=
            HandlePlayerMoved;  
    }

    private void HandlePlayerJoined(
        long accountId,
        string loginName)
    {
        // 自分自身は生成しない
        if (MMOSession.Instance != null &&
            accountId == MMOSession.Instance.AccountId)
        {
            return;
        }

        // すでに存在するなら何もしない
        if (remotePlayers.ContainsKey(accountId))
        {
            return;
        }

        if (remotePlayerPrefab == null)
        {
            Debug.LogWarning(
                "RemotePlayerPrefabが設定されていません");
            return;
        }

        Vector3 spawnPosition =
            Vector3.zero;

        if (localPlayer != null)
        {
            float offset =
                2f + remotePlayers.Count * 1.5f;

            spawnPosition =
                localPlayer.position +
                new Vector3(
                    offset,
                    0f,
                    2f);
        }

        GameObject remotePlayer =
            Instantiate(
                remotePlayerPrefab,
                spawnPosition,
                Quaternion.identity,
                remotePlayersRoot);

        remotePlayer.name =
            $"RemotePlayer_{loginName}_{accountId}";

        MMONamePlate namePlate =
            remotePlayer.GetComponentInChildren<MMONamePlate>(true);

        if (namePlate != null)
        {
            namePlate.SetLoginName(loginName);
        }
        else
        {
            Debug.LogWarning(
                $"MMONamePlateが見つかりません: {loginName}");
        }

        remotePlayers.Add(
            accountId,
            remotePlayer);

    }

    private void HandlePlayerLeft(
        long accountId,
        string loginName)
    {
        if (!remotePlayers.TryGetValue(
                accountId,
                out GameObject remotePlayer))
        {
            return;
        }

        Destroy(remotePlayer);

        remotePlayers.Remove(accountId);

    }

    private void HandlePlayerMoved(
        long accountId,
        string loginName,
        Vector3 position,
        float rotationY)
    {
        if (MMOSession.Instance != null &&
            accountId == MMOSession.Instance.AccountId)
        {
            return;
        }

        if (!remotePlayers.TryGetValue(
                accountId,
                out GameObject remotePlayer))
        {
            return;
        }

        MMORemotePlayer remoteController =
            remotePlayer.GetComponent<MMORemotePlayer>();

        if (remoteController == null)
        {
            return;
        }

        remoteController.SetNetworkState(
            position,
            rotationY);
    }
}