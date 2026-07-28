using System.Collections.Generic;
using UnityEditor;
using UnityEngine;

public class MovingPlatformSetupTool : EditorWindow
{
    private Vector3 moveOffset = new Vector3(0f, 3f, 0f);
    private float moveDuration = 2f;
    private bool randomStartTime = true;

    private float colliderHeight = 0.3f;
    private float sizeMultiplierX = 0.85f;
    private float sizeMultiplierZ = 0.85f;
    private float yOffsetFromTop = -0.1f;

    private bool disableChildColliders = true;
    private bool disableChildAnimators = true;

    [MenuItem("Tools/Hook Runner/Moving Platform Setup Tool")]
    public static void Open()
    {
        GetWindow<MovingPlatformSetupTool>("Moving Platform Setup");
    }

    private void OnGUI()
    {
        EditorGUILayout.LabelField("Moving Platform Setup", EditorStyles.boldLabel);

        EditorGUILayout.Space();

        moveOffset = EditorGUILayout.Vector3Field("Move Offset", moveOffset);
        moveDuration = EditorGUILayout.FloatField("Move Duration", moveDuration);
        randomStartTime = EditorGUILayout.Toggle("Random Start Time", randomStartTime);

        EditorGUILayout.Space();

        colliderHeight = EditorGUILayout.FloatField("Collider Height", colliderHeight);
        sizeMultiplierX = EditorGUILayout.Slider("Size Multiplier X", sizeMultiplierX, 0.1f, 1.5f);
        sizeMultiplierZ = EditorGUILayout.Slider("Size Multiplier Z", sizeMultiplierZ, 0.1f, 1.5f);
        yOffsetFromTop = EditorGUILayout.FloatField("Y Offset From Top", yOffsetFromTop);

        EditorGUILayout.Space();

        disableChildColliders = EditorGUILayout.Toggle("Disable Child Colliders", disableChildColliders);
        disableChildAnimators = EditorGUILayout.Toggle("Disable Child Animators", disableChildAnimators);

        EditorGUILayout.Space();

        if (GUILayout.Button("Create Moving Platforms From Selection"))
        {
            CreateMovingPlatformsFromSelection();
        }
    }

    private void CreateMovingPlatformsFromSelection()
    {
        GameObject[] selectedObjects = Selection.gameObjects;

        if (selectedObjects == null || selectedObjects.Length == 0)
        {
            Debug.LogWarning("No objects selected.");
            return;
        }

        List<GameObject> createdParents = new List<GameObject>();

        foreach (GameObject target in selectedObjects)
        {
            if (target == null)
            {
                continue;
            }

            if (target.scene.name == null)
            {
                Debug.LogWarning("Please select scene objects, not project assets.", target);
                continue;
            }

            Renderer[] renderers = target.GetComponentsInChildren<Renderer>(true);

            if (renderers.Length == 0)
            {
                Debug.LogWarning("No renderers found in selected object.", target);
                continue;
            }

            Transform oldParent = target.transform.parent;
            int oldSiblingIndex = target.transform.GetSiblingIndex();

            GameObject platformParent = new GameObject("MovingPlatform_" + target.name);
            Undo.RegisterCreatedObjectUndo(platformParent, "Create Moving Platform Parent");

            platformParent.transform.SetParent(oldParent, false);
            platformParent.transform.SetSiblingIndex(oldSiblingIndex);
            platformParent.transform.position = target.transform.position;
            platformParent.transform.rotation = Quaternion.identity;
            platformParent.transform.localScale = Vector3.one;

            Undo.SetTransformParent(target.transform, platformParent.transform, "Parent Object To Moving Platform");

            Rigidbody rb = Undo.AddComponent<Rigidbody>(platformParent);
            rb.isKinematic = true;
            rb.useGravity = false;
            rb.interpolation = RigidbodyInterpolation.Interpolate;
            rb.collisionDetectionMode = CollisionDetectionMode.ContinuousSpeculative;

            BoxCollider boxCollider = Undo.AddComponent<BoxCollider>(platformParent);
            FitBoxColliderToRendererTop(platformParent.transform, renderers, boxCollider);

            MovingPlatform movingPlatform = Undo.AddComponent<MovingPlatform>(platformParent);
            movingPlatform.Configure(moveOffset, moveDuration, randomStartTime);

            if (disableChildColliders)
            {
                DisableChildColliders(target);
            }

            if (disableChildAnimators)
            {
                DisableChildAnimators(target);
            }

            createdParents.Add(platformParent);

            EditorUtility.SetDirty(platformParent);
            Debug.Log("Created moving platform: " + platformParent.name, platformParent);
        }

        Selection.objects = createdParents.ToArray();
    }

    private void FitBoxColliderToRendererTop(Transform platformTransform, Renderer[] renderers, BoxCollider boxCollider)
    {
        Bounds worldBounds = renderers[0].bounds;

        for (int i = 1; i < renderers.Length; i++)
        {
            worldBounds.Encapsulate(renderers[i].bounds);
        }

        Vector3 worldCenter = worldBounds.center;
        Vector3 worldTopCenter = new Vector3(
            worldCenter.x,
            worldBounds.max.y + yOffsetFromTop,
            worldCenter.z
        );

        Vector3 localCenter = platformTransform.InverseTransformPoint(worldTopCenter);

        float scaleX = Mathf.Abs(platformTransform.lossyScale.x);
        float scaleZ = Mathf.Abs(platformTransform.lossyScale.z);

        if (scaleX <= 0f)
        {
            scaleX = 1f;
        }

        if (scaleZ <= 0f)
        {
            scaleZ = 1f;
        }

        Vector3 localSize = new Vector3(
            worldBounds.size.x / scaleX * sizeMultiplierX,
            colliderHeight,
            worldBounds.size.z / scaleZ * sizeMultiplierZ
        );

        boxCollider.center = localCenter;
        boxCollider.size = localSize;
    }

    private void DisableChildColliders(GameObject target)
    {
        Collider[] colliders = target.GetComponentsInChildren<Collider>(true);

        foreach (Collider collider in colliders)
        {
            Undo.RecordObject(collider, "Disable Child Collider");
            collider.enabled = false;
            EditorUtility.SetDirty(collider);
        }
    }

    private void DisableChildAnimators(GameObject target)
    {
        Animator[] animators = target.GetComponentsInChildren<Animator>(true);

        foreach (Animator animator in animators)
        {
            Undo.RecordObject(animator, "Disable Child Animator");
            animator.enabled = false;
            EditorUtility.SetDirty(animator);
        }
    }


}