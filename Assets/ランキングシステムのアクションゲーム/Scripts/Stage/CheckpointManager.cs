using UnityEngine;
using System;

public class CheckpointManager : MonoBehaviour
{
    [Header("Respawn")]
    [SerializeField] private Transform defaultRespawnPoint;
    [SerializeField] private float respawnYOffset = 0.2f;



    private Transform currentCheckpoint;
    private int currentCheckpointOrder = 0;

    private Rigidbody rb;

    public static event Action OnPlayerRespawned;

    private void Awake()
    {
        rb = GetComponent<Rigidbody>();

        if (defaultRespawnPoint != null)
        {
            currentCheckpoint = defaultRespawnPoint;
        }
    }

    public void SetCheckpoint(Transform checkpointTransform, int checkpointOrder)
    {
        if (checkpointTransform == null)
        {
            return;
        }

        // 古いチェックポイントに戻らないようにする
        if (checkpointOrder < currentCheckpointOrder)
        {
            return;
        }

        currentCheckpoint = checkpointTransform;
        currentCheckpointOrder = checkpointOrder;

        Debug.Log("Checkpoint updated: " + checkpointTransform.name);
    }

    public void Respawn()
    {
        if (currentCheckpoint == null)
        {
            Debug.LogWarning("Respawn failed: currentCheckpoint is null.");
            return;
        }

        Vector3 respawnPosition = currentCheckpoint.position + Vector3.up * respawnYOffset;

        // リスポーン時は必ずワールドのZ-方向を向く
        Quaternion respawnRotation = Quaternion.Euler(0f, 180f, 0f);

        if (rb != null)
        {
            rb.velocity = Vector3.zero;
            rb.angularVelocity = Vector3.zero;
            rb.useGravity = true;

            rb.position = respawnPosition;
            rb.rotation = respawnRotation;
        }

        transform.SetPositionAndRotation(respawnPosition, respawnRotation);

        FirstPersonLook firstPersonLook = GetComponentInChildren<FirstPersonLook>(true);
        if (firstPersonLook != null)
        {
            firstPersonLook.ResetLook(respawnRotation);
        }

        ThirdPersonCameraController thirdPersonCameraController = GetComponentInChildren<ThirdPersonCameraController>(true);
        if (thirdPersonCameraController != null)
        {
            thirdPersonCameraController.ResetCamera(respawnRotation);
        }

        OnPlayerRespawned?.Invoke();
    }
}