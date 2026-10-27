package app.privatemessenger

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChatBubble
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material.icons.filled.Tag
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

private data class DemoChat(val title: String, val preview: String, val type: String)

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { MessengerHome() }
    }
}

@Composable
private fun MessengerHome() {
    val chats = listOf(
        DemoChat("Избранное", "Личные заметки и сохранённые сообщения", "direct"),
        DemoChat("Команда", "Добро пожаловать в Private Messenger", "group"),
        DemoChat("Обновления", "Новости и объявления проекта", "channel")
    )
    MaterialTheme {
        Surface(modifier = Modifier.fillMaxSize(), color = Color(0xFF0E1621)) {
            Column(modifier = Modifier.fillMaxSize().background(Color(0xFF17212B))) {
                Text(
                    "Private Messenger",
                    style = MaterialTheme.typography.titleLarge,
                    color = Color.White,
                    modifier = Modifier.padding(horizontal = 20.dp, vertical = 22.dp)
                )
                Text(
                    "Стартовый экран · демонстрационные данные",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFF9EAFBE),
                    modifier = Modifier.padding(start = 20.dp, bottom = 12.dp)
                )
                LazyColumn {
                    items(chats) { chat ->
                        Row(
                            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(14.dp)
                        ) {
                            val icon = when (chat.type) {
                                "group" -> Icons.Default.Groups
                                "channel" -> Icons.Default.Tag
                                else -> Icons.Default.ChatBubble
                            }
                            androidx.compose.material3.Icon(icon, contentDescription = null, tint = Color(0xFF72B7F4))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(chat.title, color = Color.White, style = MaterialTheme.typography.titleMedium)
                                Text(chat.preview, color = Color(0xFF9EAFBE), style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }
                }
            }
        }
    }
}
