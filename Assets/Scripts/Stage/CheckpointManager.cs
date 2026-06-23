using UnityEngine;

public class CheckpointManager : MonoBehaviour
{
    [Header("Respawn")]
    [SerializeField] private Transform defaultRespawnPoint;
    [SerializeField] private bool useCheckpointRotation = true;
    [SerializeField] private float respawnYOffset = 0.2f;

    private Transform currentCheckpoint;
    private int currentCheckpointOrder = 0;

    private Rigidbody rb;

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

        if (rb != null)
        {
            rb.velocity = Vector3.zero;
            rb.angularVelocity = Vector3.zero;
            rb.useGravity = true;

            rb.position = respawnPosition;

            if (useCheckpointRotation)
            {
                rb.rotation = currentCheckpoint.rotation;
            }
        }
        else
        {
            transform.position = respawnPosition;

            if (useCheckpointRotation)
            {
                transform.rotation = currentCheckpoint.rotation;
            }
        }
    }
}