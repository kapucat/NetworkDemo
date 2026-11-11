using UnityEngine;

public class MMOMoveClickMarker : MonoBehaviour
{
    [SerializeField]
    private float lifeTime = 0.7f;

    [SerializeField]
    private float startScale = 0.5f;

    [SerializeField]
    private float endScale = 1.0f;

    private static readonly int ColorProperty = Shader.PropertyToID("_Color");

    private float elapsed;
    private Renderer cachedRenderer;
    private MaterialPropertyBlock properties;
    private Color baseColor;

    private void Awake()
    {
        transform.localScale = Vector3.one * startScale;
        cachedRenderer = GetComponent<Renderer>();

        if (cachedRenderer == null)
        {
            return;
        }

        Material material = cachedRenderer.sharedMaterial;
        if (material == null || !material.HasProperty(ColorProperty))
        {
            return;
        }

        baseColor = material.GetColor(ColorProperty);
        properties = new MaterialPropertyBlock();
        cachedRenderer.GetPropertyBlock(properties);
    }

    private void Update()
    {
        elapsed += Time.deltaTime;
        float t = lifeTime > 0f ? Mathf.Clamp01(elapsed / lifeTime) : 1f;

        transform.localScale = Vector3.one * Mathf.Lerp(startScale, endScale, t);

        if (cachedRenderer != null && properties != null)
        {
            Color color = baseColor;
            color.a = Mathf.Lerp(baseColor.a, 0f, t);
            properties.SetColor(ColorProperty, color);
            cachedRenderer.SetPropertyBlock(properties);
        }

        if (t >= 1f)
        {
            Destroy(gameObject);
        }
    }
}
