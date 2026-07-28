using UnityEngine;

public class JumpPad : MonoBehaviour
{
    [SerializeField] private float jumpForce = 20f;
    [SerializeField] private bool resetVerticalVelocity = true;

    private void OnTriggerEnter(Collider other)
    {
        if (!other.CompareTag("Player"))
        {
            return;
        }

        Rigidbody playerRb = other.GetComponent<Rigidbody>();

        if (playerRb == null)
        {
            playerRb = other.GetComponentInParent<Rigidbody>();
        }

        if (playerRb == null)
        {
            Debug.LogWarning("Player Rigidbody was not found.");
            return;
        }

        if (resetVerticalVelocity)
        {
            Vector3 velocity = playerRb.velocity;
            velocity.y = 0f;
            playerRb.velocity = velocity;
        }

        playerRb.AddForce(Vector3.up * jumpForce, ForceMode.Impulse);
    }
}