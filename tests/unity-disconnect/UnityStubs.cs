using System;
using System.Collections.Generic;
using System.Threading.Tasks;

// Console regression tests only. These types are not imported into the Unity game.
namespace UnityEngine {
 public class MonoBehaviour {
  public object gameObject=new object();
  protected static void Destroy(object value){}
  protected static void DontDestroyOnLoad(object value){}
 }
 public struct Vector3 {public float x,y,z;public Vector3(float x,float y,float z){this.x=x;this.y=y;this.z=z;}}
 public static class Application {public static bool runInBackground;}
 public static class Debug {public static void LogError(object text){}public static void LogWarning(object text){}}
 public static class JsonUtility {public static T FromJson<T>(string json){return default(T);}public static string ToJson(object value){return "{}";}}
}
namespace UnityEngine.SceneManagement {
 public static class SceneManager {public static int Loads;public static string LastScene;public static void LoadScene(string name){Loads++;LastScene=name;}}
}
namespace NativeWebSocket {
 public enum WebSocketCloseCode {Normal=1000,Abnormal=1006,PolicyViolation=1008}
 public enum WebSocketState {Connecting,Open,Closing,Closed}
 public sealed class WebSocket {
  private static readonly object gate=new object();private static readonly List<WebSocket> created=new List<WebSocket>();
  public static int Count {get{lock(gate)return created.Count;}}
  public static WebSocket Last {get{lock(gate)return created[created.Count-1];}}
  public static void Reset(){lock(gate)created.Clear();}
  public event Action OnOpen;public event Action<string> OnError;public event Action<WebSocketCloseCode> OnClose;public event Action<byte[]> OnMessage;
  private readonly TaskCompletionSource<bool> closed=new TaskCompletionSource<bool>();
  public WebSocketState State {get;set;}public int Sends;
  public WebSocket(string url){State=WebSocketState.Closed;lock(gate)created.Add(this);}
  public Task Connect(){State=WebSocketState.Open;OnOpen?.Invoke();return closed.Task;}
  public void RaiseClose(WebSocketCloseCode code){State=WebSocketState.Closed;OnClose?.Invoke(code);closed.TrySetResult(true);}
  public void RaiseOpen(){State=WebSocketState.Open;OnOpen?.Invoke();}
  public Task Close(){RaiseClose(WebSocketCloseCode.Normal);return Task.CompletedTask;}
  public Task SendText(string text){Sends++;return Task.CompletedTask;}
 }
}
public static class MMOServerConfig {public static string WebSocketUrl=>"wss://test.invalid/ws";}
