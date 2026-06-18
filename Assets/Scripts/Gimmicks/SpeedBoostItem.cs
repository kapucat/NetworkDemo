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

        Collider itemCollider = GetComponent<Collider>();
        if (itemCollider != null)
        {
            itemCollider.enabled = false;
        }

        Renderer itemRenderer = GetComponent<Renderer>();
        if (itemRenderer != null)
        {
            itemRenderer.enabled = false;
        }

        StartCoroutine(BoostPlayer(playerRb, other.transform));
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

        if (destroyOnUse)
        {
            Destroy(gameObject);
        }
    }
}