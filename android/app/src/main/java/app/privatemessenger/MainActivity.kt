package app.privatemessenger

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ChatBubble
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.Tag
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

private val Background = Color(0xFF0E1621)
private val Panel = Color(0xFF17212B)
private val Accent = Color(0xFF3390EC)
private val Muted = Color(0xFF9EAFBE)
private val JsonMedia = "application/json; charset=utf-8".toMediaType()

private data class ChatItem(val id: String, val title: String, val kind: String, val createdAt: String)
private data class MessageItem(val id: String, val senderId: String, val body: String, val createdAt: String)
private data class Account(val id: String, val username: String, val displayName: String)

private object SessionCookies : CookieJar {
    private val saved = mutableMapOf<String, List<Cookie>>()
    @Synchronized override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) {
        val existing = saved[url.host].orEmpty().filter { old -> cookies.none { it.name == old.name } }
        saved[url.host] = (existing + cookies).filter { it.expiresAt >= System.currentTimeMillis() }
    }
    @Synchronized override fun loadForRequest(url: HttpUrl): List<Cookie> =
        saved[url.host].orEmpty().filter { it.expiresAt >= System.currentTimeMillis() && it.matches(url) }
    @Synchronized fun clear() { saved.clear() }
}

private object MessengerApi {
    private val client = OkHttpClient.Builder()
        .cookieJar(SessionCookies)
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(20, TimeUnit.SECONDS)
        .build()
    private val baseUrl get() = BuildConfig.API_BASE_URL.trimEnd('/') + "/"

    private suspend fun request(path: String, method: String = "GET", payload: JSONObject? = null): String =
        withContext(Dispatchers.IO) {
            val builder = Request.Builder().url(baseUrl + path.trimStart('/'))
            if (payload != null) builder.method(method, payload.toString().toRequestBody(JsonMedia))
            else if (method == "POST") builder.post("{}".toRequestBody(JsonMedia))
            else builder.get()
            client.newCall(builder.build()).execute().use { response ->
                val body = response.body?.string().orEmpty()
                if (!response.isSuccessful) {
                    val message = runCatching { JSONObject(body).optString("error") }.getOrDefault("")
                    throw IOException(message.ifBlank { "Ошибка сервера: HTTP ${response.code}" })
                }
                body
            }
        }

    suspend fun me(): Account = parseAccount(JSONObject(request("api/v1/auth/me")))
    suspend fun login(username: String, password: String): Account =
        parseAccount(JSONObject(request("api/v1/auth/login", "POST", JSONObject().put("username", username).put("password", password))))
    suspend fun register(username: String, displayName: String, password: String): Account =
        parseAccount(JSONObject(request("api/v1/auth/register", "POST", JSONObject().put("username", username).put("displayName", displayName).put("password", password))))
    suspend fun logout() {
        try { request("api/v1/auth/logout", "POST") } finally { SessionCookies.clear() }
    }
    suspend fun chats(): List<ChatItem> {
        val json = JSONArray(request("api/v1/conversations"))
        return (0 until json.length()).map { i ->
            val c = json.getJSONObject(i)
            ChatItem(c.getString("id"), c.optString("title").ifBlank { "Личная переписка" }, c.optString("kind", "direct"), c.optString("createdAt"))
        }
    }
    suspend fun messages(chatId: String): List<MessageItem> {
        val json = JSONArray(request("api/v1/conversations/${chatId}/messages"))
        return (0 until json.length()).map { i ->
            val m = json.getJSONObject(i)
            MessageItem(m.getString("id"), m.getString("senderId"), m.getString("body"), m.getString("createdAt"))
        }
    }
    suspend fun send(chatId: String, body: String): MessageItem {
        val m = JSONObject(request("api/v1/conversations/${chatId}/messages", "POST", JSONObject().put("body", body)))
        return MessageItem(m.getString("id"), m.getString("senderId"), m.getString("body"), m.getString("createdAt"))
    }
    suspend fun create(kind: String, title: String): ChatItem {
        val c = JSONObject(request("api/v1/conversations", "POST", JSONObject().put("kind", kind).put("title", title)))
        return ChatItem(c.getString("id"), c.optString("title", title), c.getString("kind"), c.optString("createdAt"))
    }
    private fun parseAccount(o: JSONObject) = Account(o.getString("id"), o.getString("username"), o.getString("displayName"))
}

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { MessengerApp() }
    }
}

@Composable
private fun MessengerApp() {
    val scope = rememberCoroutineScope()
    var account by remember { mutableStateOf<Account?>(null) }
    var authMode by remember { mutableStateOf(false) }
    var username by remember { mutableStateOf("") }
    var displayName by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var chats by remember { mutableStateOf<List<ChatItem>>(emptyList()) }
    var active by remember { mutableStateOf<ChatItem?>(null) }
    var messages by remember { mutableStateOf<List<MessageItem>>(emptyList()) }
    var draft by remember { mutableStateOf("") }
    var createKind by remember { mutableStateOf<String?>(null) }
    var createTitle by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }

    suspend fun refreshChats(selectId: String? = null) {
        chats = MessengerApi.chats()
        active = chats.firstOrNull { it.id == selectId } ?: active?.let { a -> chats.firstOrNull { it.id == a.id } } ?: chats.firstOrNull()
        messages = active?.let { MessengerApi.messages(it.id) }.orEmpty()
    }
    fun runTask(block: suspend () -> Unit) {
        scope.launch {
            busy = true
            error = ""
            try { block() } catch (e: Exception) { error = e.message ?: "Не удалось выполнить запрос" }
            finally { busy = false }
        }
    }

    LaunchedEffect(Unit) {
        try { account = MessengerApi.me(); refreshChats() } catch (_: Exception) { /* show sign-in when no valid session exists */ }
    }
    LaunchedEffect(account?.id, active?.id) {
        val chatId = active?.id
        if (account != null && chatId != null) {
            while (true) {
                delay(4000)
                try { messages = MessengerApi.messages(chatId) }
                catch (_: Exception) { /* keep the last history during a temporary network interruption */ }
            }
        }
    }

    MaterialTheme {
        Surface(modifier = Modifier.fillMaxSize(), color = Background) {
            if (account == null) {
                Column(
                    modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(24.dp),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Box(Modifier.size(64.dp).background(Accent, CircleShape), contentAlignment = Alignment.Center) {
                        Icon(Icons.Default.ChatBubble, null, tint = Color.White, modifier = Modifier.size(32.dp))
                    }
                    Spacer(Modifier.height(18.dp))
                    Text("Private Messenger", color = Color.White, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                    Text("Общение на вашем сервере", color = Muted, modifier = Modifier.padding(top = 6.dp, bottom = 24.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        if (authMode) OutlinedButton(onClick = { authMode = false; error = "" }) { Text("Войти") }
                        else Button(onClick = { authMode = false; error = "" }) { Text("Войти") }
                        if (!authMode) OutlinedButton(onClick = { authMode = true; error = "" }) { Text("Регистрация") }
                        else Button(onClick = { authMode = true; error = "" }) { Text("Регистрация") }
                    }
                    if (authMode) {
                        OutlinedTextField(displayName, { displayName = it }, label = { Text("Ваше имя") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                        Spacer(Modifier.height(8.dp))
                    }
                    OutlinedTextField(username, { username = it }, label = { Text("Имя пользователя") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                    Spacer(Modifier.height(8.dp))
                    OutlinedTextField(password, { password = it }, label = { Text("Пароль (от 10 символов)") }, singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth())
                    if (error.isNotBlank()) Text(error, color = Color(0xFFFF929B), modifier = Modifier.padding(top = 10.dp))
                    Spacer(Modifier.height(14.dp))
                    Button(
                        enabled = !busy && username.isNotBlank() && password.length >= 10 && (!authMode || displayName.isNotBlank()),
                        onClick = { runTask {
                            account = if (authMode) MessengerApi.register(username.trim(), displayName.trim(), password) else MessengerApi.login(username.trim(), password)
                            password = ""
                            refreshChats()
                        } },
                        modifier = Modifier.fillMaxWidth()
                    ) { if (busy) CircularProgressIndicator(Modifier.size(18.dp), color = Color.White) else Text(if (authMode) "Создать аккаунт" else "Войти") }
                }
            } else {
                Column(Modifier.fillMaxSize().background(Panel)) {
                    Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 14.dp), verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text("Private Messenger", color = Color.White, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                            Text("@${account!!.username}", color = Muted, style = MaterialTheme.typography.bodySmall)
                        }
                        TextButton(onClick = { createKind = "group"; createTitle = "" }) { Icon(Icons.Default.Groups, null); Spacer(Modifier.width(3.dp)); Text("Группа") }
                        TextButton(onClick = { createKind = "channel"; createTitle = "" }) { Icon(Icons.Default.Add, null); Spacer(Modifier.width(3.dp)); Text("Канал") }
                        TextButton(onClick = { runTask { MessengerApi.logout(); account = null; active = null; chats = emptyList(); messages = emptyList() } }) { Text("Выйти") }
                    }
                    if (error.isNotBlank()) Text(error, color = Color(0xFFFF929B), modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp))
                    if (createKind != null) {
                        Column(Modifier.fillMaxWidth().padding(12.dp).background(Background, RoundedCornerShape(14.dp)).padding(14.dp)) {
                            Text(if (createKind == "group") "Новая группа" else "Новый канал", color = Color.White, style = MaterialTheme.typography.titleMedium)
                            OutlinedTextField(createTitle, { createTitle = it }, label = { Text("Название") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Button(enabled = !busy && createTitle.isNotBlank(), onClick = { runTask { val created = MessengerApi.create(createKind!!, createTitle.trim()); createKind = null; createTitle = ""; refreshChats(created.id) } }) { Text("Создать") }
                                TextButton(onClick = { createKind = null }) { Text("Отмена") }
                            }
                        }
                    }
                    Row(Modifier.weight(1f).fillMaxWidth()) {
                        Column(Modifier.width(132.dp).fillMaxSize().background(Background)) {
                            if (chats.isEmpty()) Text("Пока нет чатов", color = Muted, modifier = Modifier.padding(12.dp))
                            LazyColumn {
                                items(chats, key = { it.id }) { chat ->
                                    val selected = active?.id == chat.id
                                    Row(
                                        Modifier.fillMaxWidth().background(if (selected) Color(0xFF263B4D) else Background).clickable {
                                            active = chat
                                            runTask { messages = MessengerApi.messages(chat.id) }
                                        }.padding(horizontal = 9.dp, vertical = 13.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Icon(when (chat.kind) { "group" -> Icons.Default.Groups; "channel" -> Icons.Default.Tag; else -> Icons.Default.ChatBubble }, null, tint = Accent, modifier = Modifier.size(17.dp))
                                        Spacer(Modifier.width(7.dp))
                                        Text(chat.title, color = Color.White, maxLines = 2, style = MaterialTheme.typography.bodySmall)
                                    }
                                }
                            }
                        }
                        Column(Modifier.weight(1f).fillMaxSize().background(Panel)) {
                            if (active == null) {
                                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { Text("Выберите чат", color = Muted) }
                            } else {
                                Text(active!!.title, color = Color.White, style = MaterialTheme.typography.titleMedium, modifier = Modifier.fillMaxWidth().padding(14.dp))
                                LazyColumn(Modifier.weight(1f).fillMaxWidth().padding(horizontal = 10.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                    items(messages, key = { it.id }) { message ->
                                        val own = message.senderId == account!!.id
                                        Row(Modifier.fillMaxWidth(), horizontalArrangement = if (own) Arrangement.End else Arrangement.Start) {
                                            Column(Modifier.widthIn(max = 270.dp).background(if (own) Color(0xFF234B69) else Background, RoundedCornerShape(13.dp)).padding(horizontal = 11.dp, vertical = 8.dp)) {
                                                Text(message.body, color = Color.White, style = MaterialTheme.typography.bodyMedium)
                                                Text(message.createdAt.replace('T', ' ').take(16), color = Muted, style = MaterialTheme.typography.labelSmall, modifier = Modifier.padding(top = 4.dp))
                                            }
                                        }
                                    }
                                }
                                Row(Modifier.fillMaxWidth().padding(8.dp), verticalAlignment = Alignment.CenterVertically) {
                                    OutlinedTextField(draft, { draft = it }, modifier = Modifier.weight(1f), placeholder = { Text("Сообщение") }, maxLines = 3)
                                    Spacer(Modifier.width(6.dp))
                                    Button(enabled = !busy && draft.isNotBlank(), onClick = { val body = draft.trim(); runTask { val sent = MessengerApi.send(active!!.id, body); messages = messages + sent; draft = "" } }) { Icon(Icons.Default.Send, contentDescription = "Отправить") }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
