using System;
using System.Reflection;
using System.Threading.Tasks;
using UnityEngine;
using UnityEngine.SceneManagement;
using NativeWebSocket;
internal static class ClientTests {
 private static int passed;
 private static void Check(bool ok,string name){if(!ok)throw new Exception(name);Console.WriteLine("PASS: "+name);passed++;}
 private static object Invoke(object value,string method){return value.GetType().GetMethod(method,BindingFlags.Instance|BindingFlags.NonPublic).Invoke(value,null);}
 private static void Reset(MMOSession session){WebSocket.Reset();SceneManager.Loads=0;SceneManager.LastScene=null;session.SetAccount(42,"test_user","dummy_token");}
 private static async Task WaitFor(Func<bool> condition){var until=DateTime.UtcNow.AddSeconds(5);while(!condition()&&DateTime.UtcNow<until)await Task.Delay(20);if(!condition())throw new Exception("timed out");}
 private static async Task Run(){
  var session=new MMOSession();Invoke(session,"Awake");Reset(session);
  var client=new MMOWebSocketClient();Invoke(client,"Start");var socket=WebSocket.Last;
  Check(client.IsConnected,"initial connection opens");
  client.SendPlayerMove(new Vector3(1,0,1),0);Check(socket.Sends==1,"movement sends before disconnect");
  socket.RaiseClose(WebSocketCloseCode.Abnormal); // A reconnect is already scheduled.
  socket.RaiseClose(WebSocketCloseCode.PolicyViolation);
  Check(SceneManager.Loads==0,"close callback does not change Unity scene directly");
  socket.State=WebSocketState.Open;
  Check(!client.IsConnected,"admin disconnect suppresses connection immediately");
  client.SendPlayerMove(new Vector3(2,0,2),0);Check(socket.Sends==1,"movement cannot send after admin close");
  Invoke(client,"Update");
  Check(SceneManager.Loads==1&&SceneManager.LastScene=="LoginScene","main-thread update returns to login");
  Check(!session.IsLoggedIn&&session.AccountId==0&&session.SessionToken==""&&session.LoginName=="","local login credentials are cleared");
  socket.RaiseClose(WebSocketCloseCode.PolicyViolation);socket.RaiseClose(WebSocketCloseCode.Abnormal);Invoke(client,"Update");
  Check(SceneManager.Loads==1,"duplicate close events do not reload login twice");
  await Task.Delay(3250);
  Check(WebSocket.Count==1,"scheduled reconnect is cancelled by admin disconnect");
  await (Task)Invoke(client,"ConnectWebSocket");Check(WebSocket.Count==1,"explicit retry is blocked for the stopped client");
  Invoke(client,"OnDestroy");

  Reset(session);client=new MMOWebSocketClient();Invoke(client,"Start");socket=WebSocket.Last;
  socket.RaiseClose(WebSocketCloseCode.Abnormal);
  await WaitFor(()=>WebSocket.Count==2);
  Check(client.IsConnected&&session.IsLoggedIn&&SceneManager.Loads==0,"temporary network loss reconnects without logging out");
  socket.RaiseClose(WebSocketCloseCode.PolicyViolation);Invoke(client,"Update");
  Check(client.IsConnected&&session.IsLoggedIn&&SceneManager.Loads==0,"late close event from old socket cannot log out new connection");
  Invoke(client,"OnDestroy");

  Reset(session);client=new MMOWebSocketClient();Invoke(client,"Start");socket=WebSocket.Last;
  socket.RaiseClose(WebSocketCloseCode.Abnormal);Invoke(client,"OnDestroy");
  await Task.Delay(3250);Check(WebSocket.Count==1,"destroyed client does not reconnect after delay");

  Reset(session);client=new MMOWebSocketClient();Invoke(client,"Start");
  Check(client.IsConnected,"new login can establish a fresh connection");Invoke(client,"OnDestroy");
  Console.WriteLine(passed+" Unity client checks passed.");
 }
 private static int Main(){try{Run().GetAwaiter().GetResult();return 0;}catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
