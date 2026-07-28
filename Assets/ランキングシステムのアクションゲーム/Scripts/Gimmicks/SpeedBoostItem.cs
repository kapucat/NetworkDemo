using System.Collections;
using UnityEngine;

public class SpeedBoostItem : MonoBehaviour
{
    [SerializeField] private float boostAcceleration = 25f;
    [SerializeField] private float boostDuration = 1.5f;
    [SerializeField] private bool destroyOnUse = true;
    [SerializeField] private SpeedFovEffect fovEffect;
    [SerializeField] private SpeedLineEffect speedLineEffect;

    private bool isUsed = false;
    private Coroutine boostCoroutine;
    private RespawnablePickup respawnablePickup;

    private Renderer[] renderers;
    private Collider[] colliders;

    private void Awake()
    {
        respawnablePickup = GetComponent<RespawnablePickup>();

        renderers = GetComponentsInChildren<Renderer>(true);
        colliders = GetComponentsInChildren<Collider>(true);
    }

    private void OnEnable()
    {
        CheckpointManager.OnPlayerRespawned += ResetItem;
    }

    private void OnDisable()
    {
        CheckpointManager.OnPlayerRespawned -= ResetItem;
    }

    private void OnTriggerEnter(Collider other)
    {
        if (isUsed)
        {
            return;
        }

        if (!other.CompareTag("Player"))
        {
            return;
        }

        Rigidbody playerRb = other.GetComponent<Rigidbody>();

        if (playerRb == null)
        {
            playerRb = other.GetComponentInParent<Rigidbody>();
        }

        if (playerRb == null)
        {
            Debug.LogWarning("Player Rigidbody was not found.");
            return;
        }

        isUsed = true;

        if (fovEffect == null && Camera.main != null)
        {
            fovEffect = Camera.main.GetComponent<SpeedFovEffect>();
        }

        if (fovEffect != null)
        {
            fovEffect.PlayBoostFov(boostDuration);
        }

        if (speedLineEffect == null)
        {
            speedLineEffect = FindObjectOfType<SpeedLineEffect>();
        }

        if (speedLineEffect != null)
        {
            speedLineEffect.Play(boostDuration);
        }

        if (destroyOnUse)
        {
            HideItem();
        }

        boostCoroutine = StartCoroutine(BoostPlayer(playerRb, other.transform));
    }

    private IEnumerator BoostPlayer(Rigidbody playerRb, Transform playerTransform)
    {
        float timer = 0f;

        while (timer < boostDuration)
        {
            if (playerRb == null || playerTransform == null)
            {
                yield break;
            }

            Vector3 boostDirection = playerTransform.forward;
            boostDirection.y = 0f;
            boostDirection.Normalize();

            playerRb.AddForce(boostDirection * boostAcceleration, ForceMode.Acceleration);

            timer += Time.fixedDeltaTime;
            yield return new WaitForFixedUpdate();
        }

        boostCoroutine = null;
    }

    private void HideItem()
    {
        if (respawnablePickup != null)
        {
            respawnablePickup.HidePickup();
            return;
        }

        SetItemVisible(false);
    }

    private void ResetItem()
    {
        if (boostCoroutine != null)
        {
            StopCoroutine(boostCoroutine);
            boostCoroutine = null;
        }

        isUsed = false;

        if (respawnablePickup != null)
        {
            respawnablePickup.ResetPickup();
            return;
        }

        SetItemVisible(true);
    }

    private void SetItemVisible(bool active)
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