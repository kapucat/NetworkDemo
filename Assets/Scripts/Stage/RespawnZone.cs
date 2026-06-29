using UnityEngine;

public class RespawnZone : MonoBehaviour
{
    [SerializeField] private Transform fallbackRespawnPoint;
    [SerializeField] private DeathCounter deathCounter;

    private void OnTriggerEnter(Collider other)
    {
        if (!other.CompareTag("Player"))
        {
            return;
        }

        if (deathCounter != null)
        {
            deathCounter.AddDeath();
        }

        CheckpointManager checkpointManager = other.GetComponent<CheckpointManager>();

        if (checkpointManager != null)
        {
            checkpointManager.Respawn();
            return;
        }

        // Fallback when CheckpointManager is missing
        if (fallbackRespawnPoint == null)
        {
            Debug.LogWarning("Fallback Respawn Point is not assigned.");
            return;
        }

        Rigidbody rb = other.GetComponent<Rigidbody>();

        if (rb != null)
        {
            rb.velocity = Vector3.zero;
            rb.angularVelocity = Vector3.zero;
        }

        other.transform.position = fallbackRespawnPoint.position;
        other.transform.rotation = fallbackRespawnPoint.rotation;
    }
}