using UnityEngine;

public class ViewModeSwitcher : MonoBehaviour
{
    [Header("Cameras")]
    [SerializeField] private Camera firstPersonCamera;
    [SerializeField] private Camera thirdPersonCamera;

    [Header("Camera Controls")]
    [SerializeField] private FirstPersonLook firstPersonLook;
    [SerializeField] private ThirdPersonCameraController thirdPersonCameraController;

    [Header("Player Controllers")]
    [SerializeField] private PlayerController playerController;
    [SerializeField] private GrappleController grappleController;
    [SerializeField] private WallRunController wallRunController;

    [Header("Settings")]
    [SerializeField] private bool startThirdPerson = false;
    [SerializeField] private KeyCode switchKey = KeyCode.V;

    private bool isThirdPerson;

    private void Start()
    {
        SetViewMode(startThirdPerson);
    }

    private void Update()
    {
        if (Input.GetKeyDown(switchKey))
        {
            SetViewMode(!isThirdPerson);
        }
    }

    public void SetViewMode(bool useThirdPerson)
    {
        isThirdPerson = useThirdPerson;

        if (firstPersonCamera != null)
        {
            firstPersonCamera.enabled = !isThirdPerson;
        }

        if (thirdPersonCamera != null)
        {
            thirdPersonCamera.enabled = isThirdPerson;
        }

        if (firstPersonLook != null)
        {
            firstPersonLook.enabled = !isThirdPerson;
        }

        if (thirdPersonCameraController != null)
        {
            thirdPersonCameraController.enabled = isThirdPerson;
        }

        Transform activeCameraTransform = isThirdPerson
            ? thirdPersonCamera.transform
            : firstPersonCamera.transform;

        if (playerController != null)
        {
            playerController.SetCameraTransform(activeCameraTransform);
        }

        if (grappleController != null)
        {
            grappleController.SetCameraTransform(activeCameraTransform);
        }

        if (wallRunController != null)
        {
            wallRunController.SetCameraTransform(activeCameraTransform);

            // 1êlèÃÇÕï«ëñÇËONÅA3êlèÃÇÕï«ëñÇËOFF
            wallRunController.SetWallRunEnabled(!isThirdPerson);
        }
    }
}