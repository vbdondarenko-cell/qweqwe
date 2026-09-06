package com.linkup.app.ui

import android.os.Build
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.key
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.BlockedUser
import com.linkup.app.core.network.LinkUpApiClient
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.session.SessionCoordinator
import com.linkup.app.core.session.SessionState
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.MutationState
import com.linkup.app.core.social.SocialCoordinator
import com.linkup.app.core.social.SocialError
import com.linkup.app.ui.auth.AuthScreen
import com.linkup.app.ui.me.MeScreen
import com.linkup.app.ui.me.EditProfileScreen
import com.linkup.app.ui.social.ChatScreen
import com.linkup.app.ui.social.CreateLinkScreen
import com.linkup.app.ui.social.EditSlotScreen
import com.linkup.app.ui.social.PulseScreen
import com.linkup.app.ui.social.SlotDetailScreen
import com.linkup.app.ui.theme.LinkUpBackground
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import kotlinx.coroutines.launch
import kotlinx.coroutines.CancellationException

private enum class MainTab { PULSE, MAP, LINK, FLY, ME }

@Composable
fun LinkUpApp(
    api: LinkUpApiClient,
    sessions: SessionCoordinator,
    social: SocialCoordinator,
) {
    val sessionState by sessions.state.collectAsState()
    val scope = rememberCoroutineScope()
    var authBusy by remember { mutableStateOf(false) }
    var authError by remember { mutableStateOf<String?>(null) }
    var authInfo by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) { sessions.bootstrap() }

    Box(Modifier.fillMaxSize().background(LinkUpBackground)) {
        when (val state = sessionState) {
            SessionState.Checking -> CenterLoading("Checking session…")
            SessionState.SignedOut -> AuthScreen(
                busy = authBusy,
                errorMessage = authError,
                infoMessage = authInfo,
                onLogin = { identifier, password ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try { sessions.login(identifier, password, deviceLabel()) }
                        catch (error: Exception) { authError = error.userMessage() }
                        finally { authBusy = false }
                    }
                },
                onRegister = { email, username, displayName, password ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try { sessions.register(email, username, displayName, password, "uk", deviceLabel()) }
                        catch (error: Exception) { authError = error.userMessage() }
                        finally { authBusy = false }
                    }
                },
                onRecovery = { email ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try {
                            api.requestPasswordRecovery(email)
                            authInfo = "If recovery is available for this account, a reset message has been requested."
                        } catch (error: Exception) { authError = error.userMessage() }
                        finally { authBusy = false }
                    }
                },
            )
            is SessionState.SignedIn -> key(state.user.id) {
                SignedInRoot(state.user, api, sessions, social)
            }
            is SessionState.OfflineSession -> OfflineSessionSurface(
                expiresAt = state.expiresAtEpochMillis,
                onRetry = { scope.launch { sessions.bootstrap() } },
                onSignOut = { sessions.clearLocalSession() },
            )
            is SessionState.RecoverableError -> ErrorSurface(
                title = "Session check failed",
                message = state.message,
                onRetry = { scope.launch { sessions.bootstrap() } },
                onSecondary = { sessions.clearLocalSession() },
            )
        }
    }
}

@Composable
private fun SignedInRoot(
    user: com.linkup.app.core.network.UserProfile,
    api: LinkUpApiClient,
    sessions: SessionCoordinator,
    social: SocialCoordinator,
) {
    val scope = rememberCoroutineScope()
    val pulse by social.pulse.collectAsState()
    val selected by social.selectedSlot.collectAsState()
    val pending by social.pending.collectAsState()
    val chat by social.chat.collectAsState()
    val mutation by social.mutation.collectAsState()

    var tab by remember { mutableStateOf(MainTab.PULSE) }
    var detailOpen by remember { mutableStateOf(false) }
    var chatSlotId by remember { mutableStateOf<String?>(null) }
    var editTarget by remember { mutableStateOf<SlotModel?>(null) }
    var blockedState by remember { mutableStateOf<LoadState<List<BlockedUser>>>(LoadState.Idle) }
    var meError by remember { mutableStateOf<String?>(null) }
    var profileOpen by remember { mutableStateOf(false) }
    var profileBusy by remember { mutableStateOf(false) }
    var profileError by remember { mutableStateOf<String?>(null) }
    var blockTarget by remember { mutableStateOf<SlotOrganizer?>(null) }
    var blockBusy by remember { mutableStateOf(false) }
    var blockError by remember { mutableStateOf<String?>(null) }

    fun refreshBlocks() {
        scope.launch {
            blockedState = LoadState.Loading
            try {
                val items = api.blockedUsers()
                blockedState = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
            } catch (error: Exception) {
                blockedState = LoadState.Failure(SocialError("blocks_error", error.userMessage()))
            }
        }
    }

    DisposableEffect(user.id) {
        onDispose { social.clearAll() }
    }
    LaunchedEffect(user.id) {
        social.refreshPulse()
    }

    Box(Modifier.fillMaxSize()) {
        when {
            profileOpen -> EditProfileScreen(
                user = user,
                busy = profileBusy,
                error = profileError,
                onBack = { profileOpen = false },
                onSave = { name, avatar, visibility, language ->
                    if (!profileBusy) scope.launch {
                        profileBusy = true; profileError = null
                        try {
                            if (sessions.updateProfile(name, avatar, visibility, language)) {
                                profileOpen = false
                                social.refreshPulse()
                            }
                        } catch (error: Exception) { profileError = error.userMessage() }
                        finally { profileBusy = false }
                    }
                },
            )
            editTarget != null -> {
                val target = editTarget!!
                EditSlotScreen(
                    slot = target,
                    submitting = mutation is MutationState.Running,
                    errorMessage = (mutation as? MutationState.Failed)?.error?.message,
                    onClose = { editTarget = null },
                    onSave = { input ->
                        scope.launch {
                            if (social.editSlot(target.id, input)) editTarget = null
                        }
                    },
                )
            }
            chatSlotId != null -> ChatScreen(
                state = chat,
                mutation = mutation,
                onBack = { chatSlotId = null },
                onRefresh = { scope.launch { social.refreshChat(chatSlotId!!, 100) } },
                onSend = { text -> scope.launch { social.sendChatMessage(chatSlotId!!, text) } },
            )
            detailOpen -> SlotDetailScreen(
                state = selected,
                pending = pending,
                mutation = mutation,
                onBack = { detailOpen = false; social.clearSelected() },
                onRefresh = { id -> scope.launch { social.openSlot(id) } },
                onRequest = { id -> scope.launch { social.requestSlot(id) } },
                onLeave = { id -> scope.launch { social.leaveSlot(id) } },
                onRefreshPending = { id -> scope.launch { social.refreshPending(id) } },
                onApprove = { slotId, userId -> scope.launch { social.approveRequest(slotId, userId) } },
                onReject = { slotId, userId -> scope.launch { social.rejectRequest(slotId, userId) } },
                onStart = { id -> scope.launch { social.startSlot(id) } },
                onComplete = { id -> scope.launch { social.completeSlot(id) } },
                onCancel = { id, version -> scope.launch { social.cancelSlot(id, version) } },
                onEdit = { editTarget = it },
                onBlockUser = { target -> blockError = null; blockTarget = target },
                onOpenChat = { id ->
                    chatSlotId = id
                    scope.launch { social.refreshChat(id, 100) }
                },
            )
            else -> {
                Column(Modifier.fillMaxSize()) {
                    Box(Modifier.weight(1f).fillMaxWidth()) {
                        when (tab) {
                            MainTab.PULSE -> PulseScreen(
                                state = pulse,
                                onRefresh = { scope.launch { social.refreshPulse() } },
                                onSlotClick = { slot ->
                                    detailOpen = true
                                    scope.launch { social.openSlot(slot.id) }
                                },
                                onPrimaryAction = { slot ->
                                    if (slot.viewerState == SlotViewerState.NONE) scope.launch { social.requestSlot(slot.id) }
                                    else {
                                        detailOpen = true
                                        scope.launch { social.openSlot(slot.id) }
                                    }
                                },
                            )
                            MainTab.LINK -> CreateLinkScreen(
                                submitting = mutation is MutationState.Running,
                                errorMessage = (mutation as? MutationState.Failed)?.error?.message,
                                onClose = { tab = MainTab.PULSE },
                                onPublish = { input ->
                                    scope.launch {
                                        if (social.createSlot(input)) {
                                            tab = MainTab.PULSE
                                            detailOpen = true
                                        }
                                    }
                                },
                            )
                            MainTab.ME -> MeScreen(
                                user = user,
                                blocked = blockedState,
                                actionError = meError,
                                onRefreshBlocks = ::refreshBlocks,
                                onEditProfile = { profileError = null; profileOpen = true },
                                onUnblock = { userId ->
                                    scope.launch {
                                        try { api.unblockUser(userId); meError = null; refreshBlocks() }
                                        catch (error: Exception) { meError = error.userMessage() }
                                    }
                                },
                                onLogout = {
                                    scope.launch {
                                        try { sessions.logout() }
                                        catch (error: Exception) { meError = error.userMessage() }
                                    }
                                },
                            )
                            MainTab.MAP -> CapabilitySurface("Map", "Real City Context and map data are not active in this capability block yet.")
                            MainTab.FLY -> CapabilitySurface("Fly", "Fly production behavior is scheduled after its required city/realtime foundations.")
                        }
                    }
                    BottomNav(tab) { selectedTab ->
                        detailOpen = false; chatSlotId = null; editTarget = null; social.clearSelected(); tab = selectedTab
                        if (selectedTab == MainTab.PULSE) scope.launch { social.refreshPulse() }
                        if (selectedTab == MainTab.ME && blockedState is LoadState.Idle) refreshBlocks()
                    }
                }
            }
        }
        blockTarget?.let { target ->
            AlertDialog(
                onDismissRequest = { if (!blockBusy) blockTarget = null },
                title = { Text("Block @${target.username}?") },
                text = {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Your requests or participation in each other's LINKs will be removed. You can unblock this person in Me.")
                        blockError?.let { Text(it, color = LinkUpRed) }
                    }
                },
                confirmButton = {
                    TextButton(enabled = !blockBusy, onClick = {
                        if (!blockBusy) scope.launch {
                            blockBusy = true; blockError = null
                            try {
                                api.blockUser(target.id)
                                social.clearAll()
                                detailOpen = false; chatSlotId = null; editTarget = null
                                blockTarget = null; tab = MainTab.PULSE
                                blockedState = LoadState.Idle
                                social.refreshPulse()
                            } catch (error: Exception) { blockError = error.userMessage() }
                            finally { blockBusy = false }
                        }
                    }) { Text(if (blockBusy) "Blocking…" else "Block", color = LinkUpRed) }
                },
                dismissButton = {
                    TextButton(enabled = !blockBusy, onClick = { blockTarget = null }) { Text("Cancel") }
                },
            )
        }
    }
}

@Composable
private fun BottomNav(selected: MainTab, onSelect: (MainTab) -> Unit) {
    Row(
        Modifier.fillMaxWidth().height(72.dp).background(LinkUpElevated).border(1.dp, LinkUpBorder).padding(horizontal = 6.dp, vertical = 7.dp),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        listOf(
            MainTab.PULSE to "Pulse",
            MainTab.MAP to "Map",
            MainTab.LINK to "LINK",
            MainTab.FLY to "Fly",
            MainTab.ME to "Me",
        ).forEach { (tab, label) ->
            val active = selected == tab
            Column(
                Modifier.weight(1f).clip(RoundedCornerShape(12.dp)).clickable { onSelect(tab) }.padding(vertical = 7.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(if (tab == MainTab.LINK) "+" else "•", color = if (active || tab == MainTab.LINK) LinkUpRed else LinkUpTextMuted, fontSize = if (tab == MainTab.LINK) 21.sp else 13.sp, fontWeight = FontWeight.Black)
                Text(label, color = if (active) LinkUpTextPrimary else LinkUpTextMuted, fontSize = 10.sp, fontWeight = if (active) FontWeight.Bold else FontWeight.Normal)
            }
        }
    }
}

@Composable
private fun CenterLoading(label: String) {
    Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
        CircularProgressIndicator(color = LinkUpRed)
        Text(label, color = LinkUpTextDimmed, modifier = Modifier.padding(top = 12.dp))
    }
}

@Composable
private fun OfflineSessionSurface(expiresAt: Long, onRetry: () -> Unit, onSignOut: () -> Unit) {
    ErrorSurface("Offline", "Your encrypted local session is still valid until $expiresAt, but foundation social data requires a server connection.", onRetry, onSignOut)
}

@Composable
private fun ErrorSurface(title: String, message: String, onRetry: () -> Unit, onSecondary: () -> Unit) {
    Column(Modifier.fillMaxSize().padding(28.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
        Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 20.sp)
        Text(message, color = LinkUpTextDimmed, fontSize = 13.sp, modifier = Modifier.padding(vertical = 10.dp))
        TextButton(onClick = onRetry) { Text("Retry", color = LinkUpRed) }
        TextButton(onClick = onSecondary) { Text("Sign out locally", color = LinkUpTextMuted) }
    }
}

@Composable
private fun CapabilitySurface(title: String, message: String) {
    Column(Modifier.fillMaxSize().padding(28.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 28.sp)
        Text(message, color = LinkUpTextDimmed, fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
    }
}

private fun deviceLabel(): String = "${Build.MANUFACTURER} ${Build.MODEL}".trim().take(120)

private fun Exception.userMessage(): String = when (this) {
    is CancellationException -> throw this
    is ApiException -> message
    else -> message ?: "Request failed"
}
