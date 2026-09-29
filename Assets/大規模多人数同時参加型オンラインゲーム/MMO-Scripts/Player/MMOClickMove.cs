using UnityEngine;
using UnityEngine.AI;
using UnityEngine.EventSystems;
using UnityEngine.InputSystem;

[RequireComponent(typeof(NavMeshAgent))]
public class MMOClickMove : MonoBehaviour
{
    [SerializeField]
    private Camera mainCamera;

    [SerializeField]
    private LayerMask groundMask = ~0;

    [SerializeField]
    private GameObject moveClickMarkerPrefab;

    [Header("UI")]
    [SerializeField]
    private MMOChatUI chatUI;

    private NavMeshAgent agent;

    private GameObject currentMarkerInstance;

    private void Awake()
    {
        agent = GetComponent<NavMeshAgent>();

        if (mainCamera == null)
        {
            mainCamera = Camera.main;
        }
    }

    private void Update()
    {
        if (Mouse.current == null)
        {
            return;
        }

        // チャット入力中は移動させない
        if (chatUI != null &&
            chatUI.IsInputActive)
        {
            return;
        }

        if (!Mouse.current.rightButton.wasPressedThisFrame)
        {
            return;
        }

        // ボタンや入力欄など、
        // 実際に操作するUIの上では移動させない
        if (EventSystem.current != null &&
            EventSystem.current.IsPointerOverGameObject())
        {
            return;
        }

        if (!agent.isOnNavMesh)
        {
            Debug.LogWarning(
                "PlayerがNavMesh上にいません");
            return;
        }

        Vector2 mousePosition =
            Mouse.current.position.ReadValue();

        Ray ray =
            mainCamera.ScreenPointToRay(
                mousePosition);

        if (!Physics.Raycast(
                ray,
                out RaycastHit hit,
                1000f,
                groundMask))
        {
            return;
        }

        if (!NavMesh.SamplePosition(
                hit.point,
                out NavMeshHit navMeshHit,
                2f,
                NavMesh.AllAreas))
        {
            return;
        }

        agent.SetDestination(
            navMeshHit.position);

        ShowMoveMarker(
            navMeshHit.position);
    }

    private void ShowMoveMarker(Vector3 position)
    {
        if (moveClickMarkerPrefab == null)
        {
            return;
        }

        if (currentMarkerInstance != null)
        {
            Destroy(currentMarkerInstance);
        }

        Vector3 spawnPosition =
            position + Vector3.up * 0.03f;

        currentMarkerInstance =
            Instantiate(
                moveClickMarkerPrefab,
                spawnPosition,
                Quaternion.Euler(
                    90f,
                    0f,
                    0f));
    }
}