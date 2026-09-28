using UnityEngine;
using UnityEngine.InputSystem;

public class MMOTopDownCamera : MonoBehaviour
{
    [SerializeField]
    private Transform target;

    [Header("Camera")]
    [SerializeField]
    private float distance = 10f;

    [SerializeField]
    [Range(20f, 80f)]
    private float pitch = 55f;

    [SerializeField]
    private float followSpeed = 10f;

    [SerializeField]
    private float lookHeight = 0.8f;

    [Header("Zoom")]
    [SerializeField]
    private float zoomStep = 1f;

    [SerializeField]
    private float minDistance = 5f;

    [SerializeField]
    private float maxDistance = 16f;

    private void LateUpdate()
    {
        if (target == null)
        {
            return;
        }

        HandleZoom();

        Quaternion cameraRotation =
            Quaternion.Euler(
                pitch,
                0f,
                0f);

        Vector3 offset =
            cameraRotation *
            new Vector3(
                0f,
                0f,
                -distance);

        Vector3 desiredPosition =
            target.position + offset;

        transform.position =
            Vector3.Lerp(
                transform.position,
                desiredPosition,
                followSpeed * Time.deltaTime);

        transform.LookAt(
            target.position +
            Vector3.up * lookHeight);
    }

    private void HandleZoom()
    {
        if (Mouse.current == null)
        {
            return;
        }

        float scroll =
            Mouse.current.scroll.ReadValue().y;

        if (Mathf.Abs(scroll) < 0.01f)
        {
            return;
        }

        if (scroll > 0f)
        {
            distance -= zoomStep;
        }
        else
        {
            distance += zoomStep;
        }

        distance =
            Mathf.Clamp(
                distance,
                minDistance,
                maxDistance);
    }
}