using UnityEngine;

// Initialize the fixed third-person camera. Keep the prefab script GUID.
public class ViewModeSwitcher : MonoBehaviour
{
    [SerializeField] private Camera thirdPersonCamera;
    [SerializeField] private ThirdPersonCameraController thirdPersonCameraController;
    [SerializeField] private PlayerController playerController;

    private void Start()
    {
        if (thirdPersonCamera == null)
        {
            Debug.LogError("Third-person camera is not assigned.", this);
            return;
        }
        thirdPersonCamera.enabled = true;
        if (thirdPersonCameraController != null)
            thirdPersonCameraController.enabled = true;
        if (playerController != null)
            playerController.SetCameraTransform(thirdPersonCamera.transform);
    }
}
