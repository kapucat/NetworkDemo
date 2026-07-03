using UnityEngine;

[ExecuteAlways]
[RequireComponent(typeof(BoxCollider))]
public class AnimatedBoundsColliderFollower : MonoBehaviour
{
    [Header("Target")]
    [SerializeField] private SkinnedMeshRenderer targetRenderer;

    [Header("Collider Size")]
    [SerializeField] private float sizeMultiplierX = 0.85f;
    [SerializeField] private float sizeMultiplierZ = 0.85f;
    [SerializeField] private float colliderHeight = 0.25f;

    [Header("Position Offset")]
    [SerializeField] private float yOffsetFromTop = -0.05f;
    [SerializeField] private float xOffset = 0f;
    [SerializeField] private float zOffset = 0f;

    [Header("Update")]
    [SerializeField] private bool followEveryFrame = true;

    private BoxCollider boxCollider;

    private void Awake()
    {
        Setup();
    }

    private void Reset()
    {
        Setup();
    }

    private void LateUpdate()
    {
        if (followEveryFrame)
        {
            FitColliderToRendererTop();
        }
    }

    private void Setup()
    {
        boxCollider = GetComponent<BoxCollider>();

        if (targetRenderer == null)
        {
            targetRenderer = GetComponentInChildren<SkinnedMeshRenderer>();
        }
    }

    [ContextMenu("Fit Collider To Renderer Top")]
    public void FitColliderToRendererTop()
    {
        if (boxCollider == null)
        {
            boxCollider = GetComponent<BoxCollider>();
        }

        if (targetRenderer == null)
        {
            targetRenderer = GetComponentInChildren<SkinnedMeshRenderer>();
        }

        if (targetRenderer == null)
        {
            Debug.LogWarning("Target SkinnedMeshRenderer was not found.", this);
            return;
        }

        Bounds worldBounds = targetRenderer.bounds;

        Vector3 worldCenter = worldBounds.center;
        Vector3 worldTopCenter = new Vector3(
            worldCenter.x,
            worldBounds.max.y + yOffsetFromTop,
            worldCenter.z
        );

        Vector3 localCenter = transform.InverseTransformPoint(worldTopCenter);
        localCenter.x += xOffset;
        localCenter.z += zOffset;

        Vector3 worldSize = worldBounds.size;

        Vector3 localSize = new Vector3(
            worldSize.x / Mathf.Abs(transform.lossyScale.x) * sizeMultiplierX,
            colliderHeight,
            worldSize.z / Mathf.Abs(transform.lossyScale.z) * sizeMultiplierZ
        );

        boxCollider.center = localCenter;
        boxCollider.size = localSize;
    }
}