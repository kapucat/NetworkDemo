using UnityEngine;

public class RespawnablePickup : MonoBehaviour
{
    private Renderer[] renderers;
    private Collider[] colliders;

    private void Awake()
    {
        renderers = GetComponentsInChildren<Renderer>(true);
        colliders = GetComponentsInChildren<Collider>(true);
    }

    public void HidePickup()
    {
        SetPickupActive(false);
    }

    public void ResetPickup()
    {
        SetPickupActive(true);
    }

    private void SetPickupActive(bool active)
    {
        foreach (Renderer r in renderers)
        {
            if (r != null)
            {
                r.enabled = active;
            }
        }

        foreach (Collider c in colliders)
        {
            if (c != null)
            {
                c.enabled = active;
            }
        }
    }
}