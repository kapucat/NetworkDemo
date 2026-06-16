using System.Collections;
using UnityEngine;

public class SpeedFovEffect : MonoBehaviour
{
    [SerializeField] private Camera targetCamera;
    [SerializeField] private float normalFov = 60f;
    [SerializeField] private float boostFov = 75f;
    [SerializeField] private float changeSpeed = 8f;

    private Coroutine fovCoroutine;

    private void Awake()
    {
        if (targetCamera == null)
        {
            targetCamera = GetComponent<Camera>();
        }

        if (targetCamera != null)
        {
            normalFov = targetCamera.fieldOfView;
        }
    }

    public void PlayBoostFov(float duration)
    {
        if (targetCamera == null)
        {
            return;
        }

        if (fovCoroutine != null)
        {
            StopCoroutine(fovCoroutine);
        }

        fovCoroutine = StartCoroutine(BoostFovRoutine(duration));
    }

    private IEnumerator BoostFovRoutine(float duration)
    {
        while (Mathf.Abs(targetCamera.fieldOfView - boostFov) > 0.1f)
        {
            targetCamera.fieldOfView = Mathf.Lerp(
                targetCamera.fieldOfView,
                boostFov,
                Time.deltaTime * changeSpeed
            );

            yield return null;
        }

        float timer = 0f;

        while (timer < duration)
        {
            timer += Time.deltaTime;
            yield return null;
        }

        while (Mathf.Abs(targetCamera.fieldOfView - normalFov) > 0.1f)
        {
            targetCamera.fieldOfView = Mathf.Lerp(
                targetCamera.fieldOfView,
                normalFov,
                Time.deltaTime * changeSpeed
            );

            yield return null;
        }

        targetCamera.fieldOfView = normalFov;
        fovCoroutine = null;
    }
}