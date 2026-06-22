using UnityEngine;

public class ThirdPersonCameraController : MonoBehaviour
{
    [Header("References")]
    [SerializeField] private Transform target;
    [SerializeField] private Transform playerBody;

    [Header("Camera")]
    [SerializeField] private Vector3 offset = new Vector3(0f, 2.5f, -5f);
    [SerializeField] private float mouseSensitivity = 200f;
    [SerializeField] private float followSmooth = 12f;
    [SerializeField] private float minPitch = -25f;
    [SerializeField] private float maxPitch = 60f;

    private float yaw;
    private float pitch = 15f;

    private void OnEnable()
    {
        Cursor.lockState = CursorLockMode.Locked;
        Cursor.visible = false;

        if (playerBody != null)
        {
            yaw = playerBody.eulerAngles.y;
        }
    }

    private void LateUpdate()
    {
        if (target == null || playerBody == null)
        {
            return;
        }

        float mouseX = Input.GetAxis("Mouse X") * mouseSensitivity * Time.deltaTime;
        float mouseY = Input.GetAxis("Mouse Y") * mouseSensitivity * Time.deltaTime;

        yaw += mouseX;
        pitch -= mouseY;
        pitch = Mathf.Clamp(pitch, minPitch, maxPitch);

        Quaternion cameraRotation = Quaternion.Euler(pitch, yaw, 0f);

        Vector3 targetPosition = target.position + cameraRotation * offset;
        transform.position = Vector3.Lerp(transform.position, targetPosition, followSmooth * Time.deltaTime);

        transform.LookAt(target.position + Vector3.up * 1.3f);

        playerBody.rotation = Quaternion.Euler(0f, yaw, 0f);
    }
}