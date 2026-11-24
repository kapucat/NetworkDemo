using System;

[Serializable]
public class MMORoomMember {
    public long account_id;
    public string login_name;
    public bool ready;
}
[Serializable]
public class MMORoomSummary {
    public string id, name, phase;
    public int count, capacity;
}
[Serializable]
public class MMORoomState {
    public string id, name, phase;
    public long start_at_unix_ms;
    public MMORoomMember[] members;
}
[Serializable]
public class MMORoomPacket {
    public string type, room_id, message;
    public bool ready;
    public long server_unix_ms, client_unix_ms;
    public MMORoomState room;
    public MMORoomSummary[] rooms;
}

// Server clock estimate based on the best observed round trip; independent of PC wall-clock settings.
public sealed class MMORoomClock {
    private double offset, bestRoundTrip = double.MaxValue;
    private bool initialized;
    public void Observe(long serverMilliseconds, double localMilliseconds, long sentMilliseconds = 0) {
        if (serverMilliseconds <= 0) return;
        if (sentMilliseconds > 0) {
            double roundTrip = localMilliseconds - sentMilliseconds;
            if (roundTrip < 0 || roundTrip > bestRoundTrip) return;
            bestRoundTrip = roundTrip;
            offset = serverMilliseconds - (localMilliseconds + sentMilliseconds) / 2;
            initialized = true;
        } else if (!initialized) {
            offset = serverMilliseconds - localMilliseconds;
            initialized = true;
        }
    }
    public double ServerNow(double localMilliseconds) => localMilliseconds + offset;
    public void Reset() { initialized = false; offset = 0; bestRoundTrip = double.MaxValue; }
}
