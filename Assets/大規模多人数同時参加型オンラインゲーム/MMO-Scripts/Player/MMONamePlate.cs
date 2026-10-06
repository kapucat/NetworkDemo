using TMPro;
using UnityEngine;

public class MMONamePlate : MonoBehaviour
{
    [SerializeField] private TMP_Text nameText;

    private Camera targetCamera;

    private void Awake()
    {
        targetCamera = Camera.main;
    }

    private void LateUpdate()
    {
        if (targetCamera == null)
        {
            targetCamera = Camera.main;

            if (targetCamera == null)
            {
                return;
            }
        }

        transform.rotation = targetCamera.transform.rotation;
    }

    public void SetLoginName(string loginName)
    {
        if (nameText == null)
        {
            return;
        }

        nameText.text = loginName;
    }
}