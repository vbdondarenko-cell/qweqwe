package com.linkup.app.ui.design

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.city.latitudeFraction
import com.linkup.app.core.city.longitudeFraction
import com.linkup.app.core.network.CanonicalPlace
import com.linkup.app.core.network.MapCluster
import com.linkup.app.core.network.MapViewportQuery
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone

@Composable
fun FrozenMapScreen(
    placeSearch: LoadState<List<CanonicalPlace>>,
    center: CanonicalPlace?,
    viewport: MapViewportQuery?,
    clusters: LoadState<List<MapCluster>>,
    selectedCluster: MapCluster?,
    placeSlots: LoadState<List<SlotModel>>,
    onSearchPlaces: (String) -> Unit,
    onSelectCenter: (CanonicalPlace) -> Unit,
    onRefreshMap: () -> Unit,
    onZoomIn: () -> Unit,
    onZoomOut: () -> Unit,
    onClusterClick: (MapCluster) -> Unit,
    onSlotClick: (SlotModel) -> Unit,
    onOpenPulse: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var search by remember { mutableStateOf("") }

    LazyColumn(
        modifier = modifier.fillMaxSize().background(Color(0xFF08090B)),
        contentPadding = PaddingValues(bottom = 112.dp),
    ) {
        item(key = "map-header") {
            MapHeader()
        }
        item(key = "map-search") {
            MapCenterSearch(
                query = search,
                onQueryChange = { search = it.take(80) },
                enabled = placeSearch !is LoadState.Loading,
                onSearch = { onSearchPlaces(search.trim()) },
            )
        }
        item(key = "map-search-results") {
            MapSearchResults(
                state = placeSearch,
                onSelect = { place ->
                    search = place.name
                    onSelectCenter(place)
                },
            )
        }
        item(key = "map-controls") {
            MapControls(center, viewport, onRefreshMap, onZoomIn, onZoomOut)
        }
        item(key = "map-scope") {
            Column(
                Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 6.dp)
                    .background(LinkUpZone.copy(alpha = .65f), RoundedCornerShape(12.dp))
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(12.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Text(stringResource(R.string.map_scope_notice), color = LinkUpTextDimmed, fontSize = 11.sp)
                Text(stringResource(R.string.map_partial_provider_notice), color = LinkUpTextMuted, fontSize = 10.sp)
            }
        }
        item(key = "map-canvas") {
            ServerClusterCanvas(
                viewport = viewport,
                state = clusters,
                onRetry = onRefreshMap,
                onClusterClick = onClusterClick,
                onRefine = onZoomIn,
            )
        }
        selectedCluster?.let { cluster ->
            item(key = "cluster-${cluster.key}") {
                SelectedClusterSummary(cluster)
            }
            item(key = "map-place-slots") {
                MapPlaceSlots(
                    cluster = cluster,
                    state = placeSlots,
                    onSlotClick = onSlotClick,
                )
            }
        }
        item(key = "map-pulse-fallback") {
            LinkUpButton(
                stringResource(R.string.open_pulse),
                onOpenPulse,
                Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 12.dp),
                LinkUpButtonVariant.SECONDARY,
                LinkUpButtonSize.MD,
            )
        }
    }
}

@Composable
private fun MapHeader() {
    Column(
        Modifier.fillMaxWidth().background(LinkUpElevated.copy(alpha = .72f)).border(1.dp, LinkUpBorder)
            .padding(start = 20.dp, end = 20.dp, top = 56.dp, bottom = 12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            FrozenLineIcon(FrozenIconKind.MAP, LinkUpRed, Modifier.size(20.dp))
            Spacer(Modifier.width(8.dp))
            Text(
                stringResource(R.string.map_title),
                color = LinkUpTextPrimary,
                fontFamily = LinkUpDesign.displayFont,
                fontWeight = FontWeight.Black,
                fontSize = 24.sp,
            )
            Spacer(Modifier.weight(1f))
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Box(Modifier.size(7.dp).clip(CircleShape).background(LinkUpSuccess))
                Text(stringResource(R.string.map_server_live), color = LinkUpSuccess, fontFamily = LinkUpDesign.monoFont, fontSize = 9.sp)
            }
        }
        Spacer(Modifier.height(6.dp))
        Text(stringResource(R.string.map_subtitle), color = LinkUpTextDimmed, fontSize = 13.sp)
    }
}

@Composable
private fun MapCenterSearch(
    query: String,
    onQueryChange: (String) -> Unit,
    enabled: Boolean,
    onSearch: () -> Unit,
) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(
            Modifier.weight(1f).height(44.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
                .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(horizontal = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            FrozenLineIcon(FrozenIconKind.SEARCH, LinkUpTextMuted, Modifier.size(16.dp))
            Spacer(Modifier.width(8.dp))
            BasicTextField(
                value = query,
                onValueChange = onQueryChange,
                enabled = enabled,
                singleLine = true,
                modifier = Modifier.weight(1f),
                textStyle = TextStyle(color = LinkUpTextPrimary, fontFamily = LinkUpDesign.bodyFont, fontSize = 13.sp),
                decorationBox = { inner ->
                    if (query.isBlank()) {
                        Text(stringResource(R.string.map_center_search_hint), color = LinkUpTextMuted, fontSize = 12.sp)
                    } else inner()
                },
            )
        }
        LinkUpButton(
            stringResource(R.string.map_search),
            onSearch,
            enabled = enabled && query.trim().length >= 2,
            variant = LinkUpButtonVariant.PRIMARY,
            size = LinkUpButtonSize.MD,
        )
    }
}

@Composable
private fun MapSearchResults(
    state: LoadState<List<CanonicalPlace>>,
    onSelect: (CanonicalPlace) -> Unit,
) {
    when (state) {
        LoadState.Idle -> Unit
        LoadState.Loading -> Row(
            Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            Text(stringResource(R.string.create_place_searching), color = LinkUpTextDimmed, fontSize = 11.sp)
        }
        LoadState.Empty -> Text(
            stringResource(R.string.create_place_empty),
            color = LinkUpTextMuted,
            fontSize = 11.sp,
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 4.dp),
        )
        is LoadState.Failure -> Text(
            state.error.message,
            color = LinkUpWarning,
            fontSize = 11.sp,
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 4.dp),
        )
        is LoadState.Content -> LazyRow(
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(state.value, key = { it.id }) { place ->
                Column(
                    Modifier.width(190.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
                        .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp))
                        .clickable { onSelect(place) }.padding(10.dp),
                ) {
                    Text(place.name, color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    val meta = listOfNotNull(place.locality, place.category).joinToString(" · ").ifBlank { place.countryCode.orEmpty() }
                    Text(
                        stringResource(R.string.map_center_result_meta, meta.ifBlank { place.name }, place.precisionM),
                        color = LinkUpTextMuted,
                        fontSize = 9.sp,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun MapControls(
    center: CanonicalPlace?,
    viewport: MapViewportQuery?,
    onRefresh: () -> Unit,
    onZoomIn: () -> Unit,
    onZoomOut: () -> Unit,
) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 6.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(
            if (center == null) stringResource(R.string.map_choose_center)
            else stringResource(R.string.map_selected_center, center.name),
            color = if (center == null) LinkUpTextMuted else LinkUpTextPrimary,
            fontSize = 12.sp,
            fontWeight = FontWeight.SemiBold,
        )
        if (center != null && viewport != null) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                LinkUpButton("−", onZoomOut, enabled = viewport.zoom > 1, variant = LinkUpButtonVariant.SECONDARY, size = LinkUpButtonSize.SM)
                Text("Z${viewport.zoom}", color = LinkUpTextDimmed, fontFamily = LinkUpDesign.monoFont, fontSize = 11.sp)
                LinkUpButton("+", onZoomIn, enabled = viewport.zoom < 20, variant = LinkUpButtonVariant.SECONDARY, size = LinkUpButtonSize.SM)
                Spacer(Modifier.weight(1f))
                LinkUpButton(stringResource(R.string.common_refresh), onRefresh, variant = LinkUpButtonVariant.SECONDARY, size = LinkUpButtonSize.SM)
            }
        }
    }
}

@Composable
private fun ServerClusterCanvas(
    viewport: MapViewportQuery?,
    state: LoadState<List<MapCluster>>,
    onRetry: () -> Unit,
    onClusterClick: (MapCluster) -> Unit,
    onRefine: () -> Unit,
) {
    BoxWithConstraints(
        Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp).height(320.dp)
            .clip(RoundedCornerShape(18.dp)).background(Color(0xFF0B0C0F))
            .border(1.dp, LinkUpBorder, RoundedCornerShape(18.dp)),
    ) {
        Canvas(Modifier.fillMaxSize()) {
            val grid = 42.dp.toPx()
            var x = 0f
            while (x <= size.width) {
                drawLine(Color(0x12F7F8FA), Offset(x, 0f), Offset(x, size.height), strokeWidth = 1f)
                x += grid
            }
            var y = 0f
            while (y <= size.height) {
                drawLine(Color(0x12F7F8FA), Offset(0f, y), Offset(size.width, y), strokeWidth = 1f)
                y += grid
            }
        }

        when {
            viewport == null -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(stringResource(R.string.map_choose_center), color = LinkUpTextMuted, fontSize = 12.sp, modifier = Modifier.padding(24.dp))
            }
            state is LoadState.Idle -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                LinkUpButton(stringResource(R.string.map_refresh), onRetry, variant = LinkUpButtonVariant.PRIMARY, size = LinkUpButtonSize.MD)
            }
            state is LoadState.Loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    CircularProgressIndicator(color = LinkUpRed)
                    Text(stringResource(R.string.map_loading), color = LinkUpTextDimmed, fontSize = 11.sp)
                }
            }
            state is LoadState.Empty -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                LinkUpEmptyState(stringResource(R.string.map_empty_title), stringResource(R.string.map_empty_body))
            }
            state is LoadState.Failure -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                LinkUpErrorState(state.error.message, onRetry, stringResource(R.string.map_error_title))
            }
            state is LoadState.Content -> {
                state.value.forEach { cluster ->
                    val x = longitudeFraction(cluster.longitudeE6, viewport.westE6, viewport.eastE6)
                    val y = 1f - latitudeFraction(cluster.latitudeE6, viewport.southE6, viewport.northE6)
                    val markerSize = if (cluster.placeCount > 1) 54.dp else 46.dp
                    Box(
                        Modifier.offset(
                            x = maxWidth * x - markerSize / 2,
                            y = maxHeight * y - markerSize / 2,
                        ).size(markerSize).clip(CircleShape)
                            .background(if (cluster.placeCount > 1) LinkUpWarning else LinkUpRed)
                            .border(2.dp, Color(0xAAFFFFFF), CircleShape)
                            .clickable {
                                if (cluster.placeId != null) onClusterClick(cluster) else onRefine()
                            },
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(cluster.slotCount.toString(), color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 13.sp)
                    }
                }
                if (state.refreshing) {
                    CircularProgressIndicator(
                        color = LinkUpRed,
                        modifier = Modifier.align(Alignment.TopEnd).padding(10.dp).size(20.dp),
                        strokeWidth = 2.dp,
                    )
                }
            }
        }
    }
}

@Composable
private fun SelectedClusterSummary(cluster: MapCluster) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 6.dp)
            .background(LinkUpElevated, RoundedCornerShape(14.dp))
            .border(1.dp, LinkUpBorder, RoundedCornerShape(14.dp)).padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text(
            if (cluster.placeName != null) stringResource(R.string.map_cluster_single, cluster.placeName, cluster.slotCount)
            else stringResource(R.string.map_cluster_places, cluster.placeCount),
            color = LinkUpTextPrimary,
            fontWeight = FontWeight.Bold,
            fontSize = 13.sp,
        )
        Text(stringResource(R.string.map_cluster_slots, cluster.slotCount), color = LinkUpTextDimmed, fontSize = 11.sp)
    }
}

@Composable
private fun MapPlaceSlots(
    cluster: MapCluster,
    state: LoadState<List<SlotModel>>,
    onSlotClick: (SlotModel) -> Unit,
) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 6.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Text(
            stringResource(R.string.map_place_links, cluster.placeName ?: cluster.key),
            color = LinkUpTextPrimary,
            fontFamily = LinkUpDesign.displayFont,
            fontWeight = FontWeight.Bold,
            fontSize = 16.sp,
        )
        when (state) {
            LoadState.Idle -> Unit
            LoadState.Loading -> Text(stringResource(R.string.map_place_links_loading), color = LinkUpTextDimmed, fontSize = 11.sp)
            LoadState.Empty -> Text(stringResource(R.string.map_place_links_empty), color = LinkUpTextMuted, fontSize = 11.sp)
            is LoadState.Failure -> Text(state.error.message, color = LinkUpWarning, fontSize = 11.sp)
            is LoadState.Content -> {
                state.value.forEach { slot ->
                    FrozenSlotCard(
                        slot = slot.toFrozenSlot(),
                        onClick = { onSlotClick(slot) },
                        actionLabel = stringResource(R.string.common_open),
                        actionEnabled = true,
                        onPrimaryAction = { onSlotClick(slot) },
                    )
                }
                state.refreshError?.let { Text(it.message, color = LinkUpWarning, fontSize = 10.sp) }
            }
        }
    }
}
