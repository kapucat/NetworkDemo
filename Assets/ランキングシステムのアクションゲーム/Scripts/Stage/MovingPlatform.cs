using UnityEngine;

[RequireComponent(typeof(Rigidbody))]
public class MovingPlatform : MonoBehaviour
{
    [Header("Movement")]
    [SerializeField] private Vector3 moveOffset = new Vector3(0f, 3f, 0f);
    [SerializeField] private float moveDuration = 2f;
    [SerializeField] private AnimationCurve moveCurve = AnimationCurve.EaseInOut(0f, 0f, 1f, 1f);
    [SerializeField] private bool randomStartTime = false;

    private Rigidbody rb;
    private Vector3 startPosition;
    private float timeOffset;

    private void Awake()
    {
        rb = GetComponent<Rigidbody>();
        rb.isKinematic = true;
        rb.useGravity = false;

        startPosition = transform.position;

        if (randomStartTime)
        {
            timeOffset = Random.Range(0f, moveDuration);
        }
    }

    private void FixedUpdate()
    {
        if (moveDuration <= 0f)
        {
            return;
        }

        float rawT = Mathf.PingPong(Time.time + timeOffset, moveDuration) / moveDuration;
        float easedT = moveCurve.Evaluate(rawT);

        Vector3 targetPosition = startPosition + moveOffset * easedT;
        rb.MovePosition(targetPosition);
    }

    public void Configure(Vector3 newMoveOffset, float newMoveDuration, bool newRandomStartTime)
    {
        moveOffset = newMoveOffset;
        moveDuration = newMoveDuration;
        randomStartTime = newRandomStartTime;
    }

}