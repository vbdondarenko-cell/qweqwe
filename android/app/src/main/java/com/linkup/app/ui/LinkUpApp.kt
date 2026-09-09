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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import com.linkup.app.R
import com.linkup.app.core.capability.CapabilityCoordinator
import com.linkup.app.core.city.CityNetworkCoordinator
import com.linkup.app.core.hosting.V11HostingCoordinator
import com.linkup.app.core.city.mapViewportAround
import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.BlockedUser
import com.linkup.app.core.network.CapabilityKey
import com.linkup.app.core.network.CanonicalPlace
import com.linkup.app.core.network.LinkUpApiClient
import com.linkup.app.core.network.MapCluster
import com.linkup.app.core.network.MapViewportQuery
import com.linkup.app.core.network.MySlotsView
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
import com.linkup.app.ui.design.FrozenBottomNav
import com.linkup.app.ui.design.FrozenFlyScreen
import com.linkup.app.ui.design.FrozenMainTab
import com.linkup.app.ui.design.CapabilityMapScreen
import com.linkup.app.ui.design.FrozenMeScreen
import com.linkup.app.ui.design.FrozenPulseScreen
import com.linkup.app.ui.design.canJoin
import com.linkup.app.ui.me.EditProfileScreen
import com.linkup.app.ui.social.ChatPollingEffect
import com.linkup.app.ui.social.ChatScreen
import com.linkup.app.ui.social.CreateLinkScreen
import com.linkup.app.ui.social.EditSlotScreen
import com.linkup.app.ui.social.MySlotsScreen
import com.linkup.app.ui.social.SlotDetailScreen
import com.linkup.app.ui.theme.LinkUpBackground
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch

private enum class MainTab { PULSE, MAP, LINK, FLY, ME }
private const val MAP_DISCOVERY_WINDOW_MS = 48L * 60L * 60L * 1000L

@Composable
fun LinkUpApp(
    lifecycle: Lifecycle,
    api: LinkUpApiClient,
    sessions: SessionCoordinator,
    social: SocialCoordinator,
    hosting: V11HostingCoordinator,
    capabilities: CapabilityCoordinator,
    resetToken: String? = null,
    onResetTokenConsumed: () -> Unit,
) {
    val sessionState by sessions.state.collectAsState()
    val scope = rememberCoroutineScope()
    var authBusy by remember { mutableStateOf(false) }
    var authError by remember { mutableStateOf<String?>(null) }
    var authInfo by remember { mutableStateOf<String?>(null) }
    val genericError = stringResource(R.string.common_request_failed)
    val recoveryRequested = stringResource(R.string.recovery_request_info)
    val passwordChanged = stringResource(R.string.recovery_changed_info)

    LaunchedEffect(Unit) { sessions.bootstrap() }

    Box(Modifier.fillMaxSize().background(LinkUpBackground)) {
        if (resetToken != null) {
            AuthScreen(
                busy = authBusy,
                errorMessage = authError,
                infoMessage = authInfo,
                initialResetToken = resetToken,
                onLogin = { identifier, password ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try { sessions.login(identifier, password, deviceLabel()) }
                        catch (error: Exception) { authError = error.userMessage(genericError) }
                        finally { authBusy = false }
                    }
                },
                onRegister = { email, username, displayName, password ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try { sessions.register(email, username, displayName, password, defaultProfileLanguage(), deviceLabel()) }
                        catch (error: Exception) { authError = error.userMessage(genericError) }
                        finally { authBusy = false }
                    }
                },
                onRecovery = { email ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try {
                            api.requestPasswordRecovery(email)
                            authInfo = recoveryRequested
                        } catch (error: Exception) { authError = error.userMessage(genericError) }
                        finally { authBusy = false }
                    }
                },
                onResetPassword = { token, password ->
                    if (authBusy) false else {
                        authBusy = true; authError = null; authInfo = null
                        try {
                            api.resetPassword(token, password)
                            sessions.clearLocalSession()
                            onResetTokenConsumed()
                            authInfo = passwordChanged
                            true
                        } catch (error: Exception) {
                            authError = error.userMessage(genericError)
                            false
                        } finally { authBusy = false }
                    }
                },
            )
        } else when (val state = sessionState) {
            SessionState.Checking -> CenterLoading(stringResource(R.string.session_checking))
            SessionState.SignedOut -> AuthScreen(
                busy = authBusy,
                errorMessage = authError,
                infoMessage = authInfo,
                onLogin = { identifier, password ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try { sessions.login(identifier, password, deviceLabel()) }
                        catch (error: Exception) { authError = error.userMessage(genericError) }
                        finally { authBusy = false }
                    }
                },
                onRegister = { email, username, displayName, password ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try { sessions.register(email, username, displayName, password, defaultProfileLanguage(), deviceLabel()) }
                        catch (error: Exception) { authError = error.userMessage(genericError) }
                        finally { authBusy = false }
                    }
                },
                onRecovery = { email ->
                    scope.launch {
                        authBusy = true; authError = null; authInfo = null
                        try {
                            api.requestPasswordRecovery(email)
                            authInfo = recoveryRequested
                        } catch (error: Exception) { authError = error.userMessage(genericError) }
                        finally { authBusy = false }
                    }
                },
                onResetPassword = { token, password ->
                    if (authBusy) false else {
                        authBusy = true; authError = null; authInfo = null
                        try {
                            api.resetPassword(token, password)
                            authInfo = passwordChanged
                            true
                        } catch (error: Exception) {
                            authError = error.userMessage(genericError)
                            false
                        } finally { authBusy = false }
                    }
                },
            )
            is SessionState.SignedIn -> key(state.user.id) {
                SignedInRoot(state.user, api, sessions, social, hosting, capabilities, lifecycle)
            }
            is SessionState.OfflineSession -> OfflineSessionSurface(
                expiresAt = state.expiresAtEpochMillis,
                onRetry = { scope.launch { sessions.bootstrap() } },
                onSignOut = { sessions.clearLocalSession() },
            )
            is SessionState.RecoverableError -> ErrorSurface(
                title = stringResource(R.string.session_check_failed),
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
    hosting: V11HostingCoordinator,
    capabilities: CapabilityCoordinator,
    lifecycle: Lifecycle,
) {
    val scope = rememberCoroutineScope()
    val city = remember(api) { CityNetworkCoordinator(api) }
    val capabilitySnapshot by capabilities.snapshot.collectAsState()
    val mapEnabled = capabilitySnapshot.enabled(CapabilityKey.MAP)
    val pulse by social.pulse.collectAsState()
    val mySlots by social.mySlots.collectAsState()
    val selected by social.selectedSlot.collectAsState()
    val pending by social.pending.collectAsState()
    val accepted by social.accepted.collectAsState()
    val chat by social.chat.collectAsState()
    val mutation by social.mutation.collectAsState()
    val hostingDraft by hosting.draft.collectAsState()
    val placeSearch by city.places.collectAsState()
    val mapClusters by city.map.collectAsState()
    val mapPlaceSlots by city.mapSlots.collectAsState()
    val genericError = stringResource(R.string.common_request_failed)
    val hostingBusy = when (val state = hostingDraft) {
        is LoadState.Loading -> true
        is LoadState.Content -> state.refreshing
        else -> false
    }
    val hostingError = when (val state = hostingDraft) {
        is LoadState.Failure -> state.error.message
        is LoadState.Content -> state.refreshError?.message
        else -> null
    }

    var tab by remember { mutableStateOf(MainTab.PULSE) }
    var detailOpen by remember { mutableStateOf(false) }
    var mySlotsOpen by remember { mutableStateOf(false) }
    var myView by remember { mutableStateOf(MySlotsView.HOSTING) }
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
    var mapCenter by remember { mutableStateOf<CanonicalPlace?>(null) }
    var mapViewport by remember { mutableStateOf<MapViewportQuery?>(null) }
    var selectedMapCluster by remember { mutableStateOf<MapCluster?>(null) }

    fun openDesignedSlot(slot: SlotModel) {
        detailOpen = true
        scope.launch { social.openSlot(slot.id) }
    }

    fun handleDesignedPrimaryAction(slot: SlotModel) {
        if (slot.viewerState == SlotViewerState.NONE) {
            if (!slot.canJoin()) return
            scope.launch {
                if (social.requestSlot(slot.id)) detailOpen = true
            }
        } else {
            openDesignedSlot(slot)
        }
    }

    fun selectMapCenter(place: CanonicalPlace) {
        if (!mapEnabled) return
        val now = System.currentTimeMillis()
        val query = mapViewportAround(
            center = place,
            zoom = 13,
            fromEpochMillis = now,
            toEpochMillis = now + MAP_DISCOVERY_WINDOW_MS,
        )
        mapCenter = place
        mapViewport = query
        selectedMapCluster = null
        city.clearPlaces()
        city.clearMapSlots()
        scope.launch { city.refreshMap(query) }
    }

    fun refreshCurrentMap() {
        if (!mapEnabled) return
        val query = mapViewport ?: return
        selectedMapCluster = null
        scope.launch { city.refreshMap(query) }
    }

    fun changeMapZoom(delta: Int) {
        if (!mapEnabled) return
        val center = mapCenter ?: return
        val current = mapViewport ?: return
        val nextZoom = (current.zoom + delta).coerceIn(1, 20)
        if (nextZoom == current.zoom) return
        val query = mapViewportAround(
            center = center,
            zoom = nextZoom,
            fromEpochMillis = current.fromEpochMillis,
            toEpochMillis = current.toEpochMillis,
            limit = current.limit,
        )
        mapViewport = query
        selectedMapCluster = null
        scope.launch { city.refreshMap(query) }
    }

    fun openMapCluster(cluster: MapCluster) {
        if (!mapEnabled) return
        val placeId = cluster.placeId ?: return
        val query = mapViewport ?: return
        selectedMapCluster = cluster
        scope.launch {
            city.refreshMapPlaceSlots(placeId, query.fromEpochMillis, query.toEpochMillis)
        }
    }

    fun refreshBlocks() {
        scope.launch {
            blockedState = LoadState.Loading
            try {
                val items = api.blockedUsers()
                blockedState = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
            } catch (error: Exception) {
                blockedState = LoadState.Failure(SocialError("blocks_error", error.userMessage(genericError)))
            }
        }
    }

    fun unblockUser(userId: String) {
        scope.launch {
            meError = null
            try {
                api.unblockUser(userId)
                val items = api.blockedUsers()
                blockedState = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
                social.refreshPulse()
            } catch (error: Exception) {
                meError = error.userMessage(genericError)
            }
        }
    }

    DisposableEffect(user.id) {
        onDispose {
            social.clearAll()
            hosting.clear()
            city.clearAll()
        }
    }
    LaunchedEffect(mySlotsOpen, detailOpen, myView) {
        if (mySlotsOpen && !detailOpen) social.refreshMySlots(myView)
    }
    LaunchedEffect(user.id) { social.refreshPulse() }

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
                        } catch (error: Exception) { profileError = error.userMessage(genericError) }
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
                    onSave = { input -> scope.launch { if (social.editSlot(target.id, input)) editTarget = null } },
                )
            }
            chatSlotId != null -> {
                ChatPollingEffect(lifecycle, chatSlotId!!, social)
                ChatScreen(
                    state = chat,
                    mutation = mutation,
                    onBack = { chatSlotId = null },
                    onRefresh = { scope.launch { social.refreshChat(chatSlotId!!, 100) } },
                    onSend = { text -> scope.launch { social.sendChatMessage(chatSlotId!!, text) } },
                )
            }
            detailOpen -> SlotDetailScreen(
                state = selected,
                pending = pending,
                accepted = accepted,
                onRefreshAccepted = { id -> scope.launch { social.refreshAccepted(id) } },
                onRemoveParticipant = { id, userId, version -> scope.launch { social.removeParticipant(id, userId, version) } },
                mutation = mutation,
                hostingBusy = hostingBusy,
                hostingError = hostingError,
                onPublishDraft = { draft ->
                    scope.launch {
                        val published = hosting.publishDraft(draft)
                        if (published != null) {
                            social.openSlot(published.id)
                            social.refreshPulse()
                        }
                    }
                },
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
                onOpenChat = { id -> chatSlotId = id },
            )
            mySlotsOpen -> MySlotsScreen(
                state = mySlots,
                view = myView,
                onViewChange = { myView = it },
                onBack = { mySlotsOpen = false },
                onRefresh = { scope.launch { social.refreshMySlots(myView) } },
                onSlotClick = { item -> detailOpen = true; scope.launch { social.openSlot(item.id) } },
            )
            else -> {
                Column(Modifier.fillMaxSize()) {
                    Box(Modifier.weight(1f).fillMaxWidth()) {
                        when (tab) {
                            MainTab.PULSE -> FrozenPulseScreen(
                                state = pulse,
                                onRefresh = { scope.launch { social.refreshPulse() } },
                                onSlotClick = ::openDesignedSlot,
                                onPrimaryAction = ::handleDesignedPrimaryAction,
                            )
                            MainTab.LINK -> CreateLinkScreen(
                                submitting = hostingBusy,
                                errorMessage = hostingError,
                                placeSearch = placeSearch,
                                onPlaceSearch = { query -> scope.launch { city.searchPlaces(query) } },
                                onClearPlaceSearch = city::clearPlaces,
                                onClose = {
                                    city.clearPlaces()
                                    tab = MainTab.PULSE
                                },
                                onSaveDraft = { input ->
                                    scope.launch {
                                        if (hosting.createDraft(input)) {
                                            val draft = (hosting.draft.value as? LoadState.Content<SlotModel>)?.value
                                            if (draft != null) {
                                                city.clearPlaces()
                                                tab = MainTab.PULSE
                                                detailOpen = true
                                                social.openSlot(draft.id)
                                            }
                                        }
                                    }
                                },
                            )
                            MainTab.ME -> FrozenMeScreen(
                                user = user,
                                blocked = blockedState,
                                actionError = meError,
                                onRefreshBlocks = ::refreshBlocks,
                                onUnblock = ::unblockUser,
                                onEditProfile = { profileError = null; profileOpen = true },
                                onMySlots = { myView = MySlotsView.HOSTING; mySlotsOpen = true },
                                onLogout = {
                                    scope.launch {
                                        try { sessions.logout() }
                                        finally {
                                            social.clearAll()
                                            city.clearAll()
                                        }
                                    }
                                },
                            )
                            MainTab.MAP -> CapabilityMapScreen(
                                enabled = mapEnabled,
                                placeSearch = placeSearch,
                                center = mapCenter,
                                viewport = mapViewport,
                                clusters = mapClusters,
                                selectedCluster = selectedMapCluster,
                                placeSlots = mapPlaceSlots,
                                onSearchPlaces = { query -> if (mapEnabled) scope.launch { city.searchPlaces(query) } },
                                onSelectCenter = ::selectMapCenter,
                                onRefreshMap = ::refreshCurrentMap,
                                onZoomIn = { changeMapZoom(1) },
                                onZoomOut = { changeMapZoom(-1) },
                                onClusterClick = ::openMapCluster,
                                onSlotClick = ::openDesignedSlot,
                                onOpenPulse = {
                                    tab = MainTab.PULSE
                                    scope.launch { social.refreshPulse() }
                                },
                            )
                            MainTab.FLY -> FrozenFlyScreen(
                                onOpenPulse = {
                                    tab = MainTab.PULSE
                                    scope.launch { social.refreshPulse() }
                                },
                            )
                        }
                    }
                    BottomNav(tab) { selectedTab ->
                        detailOpen = false; chatSlotId = null; editTarget = null; social.clearSelected(); tab = selectedTab
                        if (selectedTab != MainTab.LINK) city.clearPlaces()
                        if (selectedTab == MainTab.PULSE) scope.launch { social.refreshPulse() }
                        if (selectedTab == MainTab.ME && blockedState is LoadState.Idle) refreshBlocks()
                    }
                }
            }
        }
        blockTarget?.let { target ->
            AlertDialog(
                onDismissRequest = { if (!blockBusy) blockTarget = null },
                title = { Text(stringResource(R.string.block_confirm_title, target.username)) },
                text = {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(stringResource(R.string.block_relationship_body))
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
                            } catch (error: Exception) { blockError = error.userMessage(genericError) }
                            finally { blockBusy = false }
                        }
                    }) {
                        Text(if (blockBusy) stringResource(R.string.block_busy) else stringResource(R.string.common_block), color = LinkUpRed)
                    }
                },
                dismissButton = {
                    TextButton(enabled = !blockBusy, onClick = { blockTarget = null }) { Text(stringResource(R.string.common_cancel)) }
                },
            )
        }
    }
}

@Composable
private fun BottomNav(selected: MainTab, onSelect: (MainTab) -> Unit) {
    val frozen = when (selected) {
        MainTab.PULSE -> FrozenMainTab.PULSE
        MainTab.MAP -> FrozenMainTab.MAP
        MainTab.LINK -> FrozenMainTab.CREATE
        MainTab.FLY -> FrozenMainTab.FLY
        MainTab.ME -> FrozenMainTab.ME
    }
    FrozenBottomNav(selected = frozen, onSelect = { tab ->
        onSelect(
            when (tab) {
                FrozenMainTab.PULSE -> MainTab.PULSE
                FrozenMainTab.MAP -> MainTab.MAP
                FrozenMainTab.CREATE -> MainTab.LINK
                FrozenMainTab.FLY -> MainTab.FLY
                FrozenMainTab.ME -> MainTab.ME
            },
        )
    })
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
    ErrorSurface(
        stringResource(R.string.session_offline_title),
        stringResource(R.string.session_offline_body_format, sessionExpiryLabel(expiresAt)),
        onRetry,
        onSignOut,
    )
}

@Composable
private fun ErrorSurface(title: String, message: String, onRetry: () -> Unit, onSecondary: () -> Unit) {
    Column(Modifier.fillMaxSize().padding(28.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
        Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 20.sp)
        Text(message, color = LinkUpTextDimmed, fontSize = 13.sp, modifier = Modifier.padding(vertical = 10.dp))
        TextButton(onClick = onRetry) { Text(stringResource(R.string.common_retry), color = LinkUpRed) }
        TextButton(onClick = onSecondary) { Text(stringResource(R.string.session_sign_out_local), color = LinkUpTextMuted) }
    }
}

@Composable
private fun CapabilitySurface(title: String, message: String) {
    Column(Modifier.fillMaxSize().padding(28.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 28.sp)
        Text(message, color = LinkUpTextDimmed, fontSize = 13.sp, modifier = Modifier.padding(top = 8.dp))
    }
}

private fun defaultProfileLanguage(): String = if (Locale.getDefault().language.equals("uk", ignoreCase = true)) "uk" else "en"

private fun sessionExpiryLabel(epochMillis: Long): String = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")
    .withZone(ZoneId.systemDefault())
    .format(Instant.ofEpochMilli(epochMillis))

private fun deviceLabel(): String = "${Build.MANUFACTURER} ${Build.MODEL}".trim().take(120)

private fun Exception.userMessage(fallback: String): String = when (this) {
    is CancellationException -> throw this
    is ApiException -> message
    else -> message ?: fallback
}
