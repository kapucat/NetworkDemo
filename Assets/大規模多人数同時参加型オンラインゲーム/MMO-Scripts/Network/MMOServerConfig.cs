public static class MMOServerConfig
{
#if UNITY_EDITOR
    private const string Host = "localhost";
#else
    private const string Host = "localhost";
#endif

    //private const int Port = 8080;
    private const int Port = 8081;

    public static string LoginUrl =>
        $"http://{Host}:{Port}/api/accounts/login";

    public static string RegisterUrl =>
        $"http://{Host}:{Port}/api/accounts/register";

    public static string WebSocketUrl =>
        $"ws://{Host}:{Port}/ws";   
}