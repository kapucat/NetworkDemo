using UnityEngine;

public class MMORemotePlayer : MonoBehaviour
{
    [SerializeField]
    private float moveSmooth = 12f;

    [SerializeField]
    private float rotationSmooth = 12f;

    private Vector3 targetPosition;
    private Quaternion targetRotation;

    private bool hasNetworkState;

    private void Update()
    {
        if (!hasNetworkState)
        {
            return;
        }

        float moveT =
            1f - Mathf.Exp(
                -moveSmooth * Time.deltaTime);

        float rotationT =
            1f - Mathf.Exp(
                -rotationSmooth * Time.deltaTime);

        transform.position =
            Vector3.Lerp(
                transform.position,
                targetPosition,
                moveT);

        transform.rotation =
            Quaternion.Slerp(
                transform.rotation,
                targetRotation,
                rotationT);
    }

    public void SetNetworkState(
        Vector3 position,
        float rotationY)
    {
        if (!hasNetworkState)
        {
            transform.position = position;

            transform.rotation =
                Quaternion.Euler(
                    0f,
                    rotationY,
                    0f);

            hasNetworkState = true;
        }

        targetPosition = position;

        targetRotation =
            Quaternion.Euler(
                0f,
                rotationY,
                0f);
    }
}