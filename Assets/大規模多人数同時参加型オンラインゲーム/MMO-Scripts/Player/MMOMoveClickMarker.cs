using UnityEngine;

public class MMOMoveClickMarker : MonoBehaviour
{
    [SerializeField]
    private float lifeTime = 0.7f;

    [SerializeField]
    private float startScale = 0.5f;

    [SerializeField]
    private float endScale = 1.0f;

    private float elapsed;

    private Vector3 initialScale;

    private Renderer cachedRenderer;
    private Material runtimeMaterial;
    private Color baseColor;

    private void Awake()
    {
        initialScale = Vector3.one * startScale;

        transform.localScale = initialScale;

        cachedRenderer = GetComponent<Renderer>();

        if (cachedRenderer != null)
        {
            runtimeMaterial = cachedRenderer.material;
            baseColor = runtimeMaterial.color;
        }
    }

    private void Update()
    {
        elapsed += Time.deltaTime;

        float t = Mathf.Clamp01(elapsed / lifeTime);

        float scale = Mathf.Lerp(startScale, endScale, t);
        transform.localScale = Vector3.one * scale;

        if (runtimeMaterial != null)
        {
            Color color = baseColor;
            color.a = Mathf.Lerp(baseColor.a, 0f, t);
            runtimeMaterial.color = color;
        }

        if (elapsed >= lifeTime)
        {
            Destroy(gameObject);
        }
    }
}