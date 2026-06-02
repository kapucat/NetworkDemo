using UnityEngine;

public class RespawnZone : MonoBehaviour
{
    [SerializeField] private Transform respawnPoint;
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

        Rigidbody rb = other.GetComponent<Rigidbody>();

        if (rb != null)
        {
            rb.velocity = Vector3.zero;
            rb.angularVelocity = Vector3.zero;
        }

        other.transform.position = respawnPoint.position;
        other.transform.rotation = respawnPoint.rotation;
    }
}