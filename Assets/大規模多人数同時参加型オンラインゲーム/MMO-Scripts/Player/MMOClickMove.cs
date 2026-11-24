using System.Collections.Generic;
using UnityEngine;
using UnityEngine.AI;
using UnityEngine.EventSystems;
using UnityEngine.InputSystem;
using UnityEngine.UI;

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

    private readonly List<RaycastResult> uiRaycastResults = new();
    private EventSystem pointerEventSystem;
    private PointerEventData pointerEventData;

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
        if (!MMORoomLobby.MovementAllowed) return;
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

        Vector2 mousePosition =
            Mouse.current.position.ReadValue();

        if (IsPointerOverInteractiveUI(mousePosition))
        {
            return;
        }

        if (!agent.isOnNavMesh)
        {
            Debug.LogWarning(
                "PlayerがNavMesh上にいません");
            return;
        }

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

    private bool IsPointerOverInteractiveUI(Vector2 mousePosition)
    {
        EventSystem eventSystem = EventSystem.current;
        if (eventSystem == null)
        {
            return false;
        }

        if (pointerEventData == null || pointerEventSystem != eventSystem)
        {
            pointerEventSystem = eventSystem;
            pointerEventData = new PointerEventData(eventSystem);
        }

        pointerEventData.Reset();
        pointerEventData.position = mousePosition;
        uiRaycastResults.Clear();
        eventSystem.RaycastAll(pointerEventData, uiRaycastResults);

        // 履歴や背景の上は地面へ通し、操作するUIだけ移動を止める。
        foreach (RaycastResult result in uiRaycastResults)
        {
            Selectable control = result.gameObject.GetComponentInParent<Selectable>();
            if (control != null && control.IsActive())
            {
                return true;
            }
        }

        return false;
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
