using UnityEngine;
using UnityEngine.UI;

public class GrappleCrosshairUI : MonoBehaviour
{
    [SerializeField] private Transform cameraTransform;
    [SerializeField] private Image crosshairImage;
    [SerializeField] private float maxDistance = 60f;
    [SerializeField] private LayerMask grappleMask;

    [SerializeField] private Color canGrappleColor = Color.blue;
    [SerializeField] private Color cannotGrappleColor = Color.red;

    private void Update()
    {
        if (cameraTransform == null || crosshairImage == null)
        {
            return;
        }

        bool canGrapple = Physics.Raycast(
            cameraTransform.position,
            cameraTransform.forward,
            maxDistance,
            grappleMask
        );

        crosshairImage.color = canGrapple ? canGrappleColor : cannotGrappleColor;
    }
}