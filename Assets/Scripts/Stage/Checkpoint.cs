using UnityEngine;

public class Checkpoint : MonoBehaviour
{
    [Header("Checkpoint")]
    [SerializeField] private Transform respawnPoint;
    [SerializeField] private int checkpointOrder = 1;

    private void Reset()
    {
        respawnPoint = transform;
    }

    private void Awake()
    {
        if (respawnPoint == null)
        {
            respawnPoint = transform;
        }
    }

    private void OnTriggerEnter(Collider other)
    {
        if (!other.CompareTag("Player"))
        {
            return;
        }

        CheckpointManager checkpointManager = other.GetComponent<CheckpointManager>();

        if (checkpointManager == null)
        {
            Debug.LogWarning("CheckpointManager is not found on Player.");
            return;
        }

        checkpointManager.SetCheckpoint(respawnPoint, checkpointOrder);
    }
}