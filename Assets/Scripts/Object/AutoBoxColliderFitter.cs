using UnityEngine;

[ExecuteAlways]
[RequireComponent(typeof(BoxCollider))]
public class AutoBoxColliderFitter : MonoBehaviour
{
    [SerializeField] private float colliderHeight = 0.2f;
    [SerializeField] private float yOffset = 0f;
    [SerializeField] private bool includeInactive = false;

    [ContextMenu("Fit Box Collider To Child Renderers")]
    public void FitBoxColliderToChildRenderers()
    {
        Renderer[] renderers = GetComponentsInChildren<Renderer>(includeInactive);

        if (renderers.Length == 0)
        {
            Debug.LogWarning("No child renderers found.", this);
            return;
        }

        Bounds worldBounds = renderers[0].bounds;

        for (int i = 1; i < renderers.Length; i++)
        {
            worldBounds.Encapsulate(renderers[i].bounds);
        }

        Vector3[] corners = new Vector3[8];

        Vector3 min = worldBounds.min;
        Vector3 max = worldBounds.max;

        corners[0] = new Vector3(min.x, min.y, min.z);
        corners[1] = new Vector3(min.x, min.y, max.z);
        corners[2] = new Vector3(min.x, max.y, min.z);
        corners[3] = new Vector3(min.x, max.y, max.z);
        corners[4] = new Vector3(max.x, min.y, min.z);
        corners[5] = new Vector3(max.x, min.y, max.z);
        corners[6] = new Vector3(max.x, max.y, min.z);
        corners[7] = new Vector3(max.x, max.y, max.z);

        Bounds localBounds = new Bounds(transform.InverseTransformPoint(corners[0]), Vector3.zero);

        for (int i = 1; i < corners.Length; i++)
        {
            localBounds.Encapsulate(transform.InverseTransformPoint(corners[i]));
        }

        BoxCollider boxCollider = GetComponent<BoxCollider>();

        Vector3 size = localBounds.size;
        size.y = colliderHeight;

        Vector3 center = localBounds.center;
        center.y = localBounds.max.y - colliderHeight * 0.5f + yOffset;

        boxCollider.center = center;
        boxCollider.size = size;

        Debug.Log("Box Collider fitted to child renderers.", this);
    }
}