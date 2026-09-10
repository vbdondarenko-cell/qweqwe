package com.linkup.app.ui.design

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.CanonicalPlace
import com.linkup.app.core.network.CityContextModel
import com.linkup.app.core.network.MapCluster
import com.linkup.app.core.network.MapViewportQuery
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning

@Composable
fun CapabilityMapScreen(
    enabled: Boolean,
    cityContextEnabled: Boolean,
    cityContext: LoadState<CityContextModel>,
    cityPermissionGranted: Boolean,
    cityPermissionDenied: Boolean,
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
    onRequestCityContext: () -> Unit,
    onOpenPulse: () -> Unit,
    modifier: Modifier = Modifier,
) {
    if (!enabled) {
        MapCapabilityInactive(
            cityContextEnabled = cityContextEnabled,
            cityContext = cityContext,
            cityPermissionGranted = cityPermissionGranted,
            cityPermissionDenied = cityPermissionDenied,
            onRequestCityContext = onRequestCityContext,
            onOpenPulse = onOpenPulse,
            modifier = modifier,
        )
        return
    }
    FrozenMapScreen(
        cityContextEnabled = cityContextEnabled,
        cityContext = cityContext,
        cityPermissionGranted = cityPermissionGranted,
        cityPermissionDenied = cityPermissionDenied,
        onRequestCityContext = onRequestCityContext,
        placeSearch = placeSearch,
        center = center,
        viewport = viewport,
        clusters = clusters,
        selectedCluster = selectedCluster,
        placeSlots = placeSlots,
        onSearchPlaces = onSearchPlaces,
        onSelectCenter = onSelectCenter,
        onRefreshMap = onRefreshMap,
        onZoomIn = onZoomIn,
        onZoomOut = onZoomOut,
        onClusterClick = onClusterClick,
        onSlotClick = onSlotClick,
        onOpenPulse = onOpenPulse,
        modifier = modifier,
    )
}

@Composable
internal fun MapCityContextCard(
    cityContextEnabled: Boolean,
    cityContext: LoadState<CityContextModel>,
    cityPermissionGranted: Boolean,
    cityPermissionDenied: Boolean,
    onRequestCityContext: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val contentState = cityContext as? LoadState.Content<CityContextModel>
    val content = contentState?.value
    val busy = cityContext is LoadState.Loading || contentState?.refreshing == true
    val statusColor = when {
        !cityContextEnabled -> LinkUpTextDimmed
        content != null -> LinkUpSuccess
        cityPermissionDenied -> LinkUpWarning
        cityContext is LoadState.Failure -> LinkUpWarning
        else -> LinkUpTextDimmed
    }
    val statusText = when {
        !cityContextEnabled -> stringResource(R.string.map_city_context_disabled)
        content != null -> stringResource(R.string.map_city_context_active, content.locality.name)
        busy -> stringResource(R.string.map_city_context_resolving)
        cityPermissionDenied -> stringResource(R.string.map_city_context_denied)
        else -> stringResource(R.string.map_city_context_unavailable)
    }

    Column(
        modifier.fillMaxWidth()
            .background(LinkUpElevated, RoundedCornerShape(18.dp))
            .border(1.dp, LinkUpBorder, RoundedCornerShape(18.dp))
            .padding(18.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            FrozenLineIcon(FrozenIconKind.MAP, statusColor, Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            Text(
                stringResource(R.string.map_city_context_title),
                color = LinkUpTextPrimary,
                fontFamily = LinkUpDesign.displayFont,
                fontWeight = FontWeight.Bold,
                fontSize = 18.sp,
            )
        }
        Text(stringResource(R.string.map_city_context_body), color = LinkUpTextDimmed, fontSize = 12.sp)
        Text(statusText, color = statusColor, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
        if (content?.switchPending == true) {
            Text(stringResource(R.string.map_city_context_switch_pending), color = LinkUpWarning, fontSize = 11.sp)
        }
        if (contentState?.refreshError != null) {
            Text(stringResource(R.string.map_city_context_unavailable), color = LinkUpWarning, fontSize = 11.sp)
        }
        LinkUpButton(
            label = stringResource(
                if (cityPermissionGranted || content != null) R.string.map_city_context_refresh
                else R.string.map_city_context_use,
            ),
            onClick = onRequestCityContext,
            modifier = Modifier.fillMaxWidth(),
            variant = LinkUpButtonVariant.SECONDARY,
            size = LinkUpButtonSize.MD,
            enabled = cityContextEnabled && !busy,
        )
    }
}

@Composable
private fun MapCapabilityInactive(
    cityContextEnabled: Boolean,
    cityContext: LoadState<CityContextModel>,
    cityPermissionGranted: Boolean,
    cityPermissionDenied: Boolean,
    onRequestCityContext: () -> Unit,
    onOpenPulse: () -> Unit,
    modifier: Modifier = Modifier,
) {
    LazyColumn(
        modifier.fillMaxSize().background(Color(0xFF08090B)),
        contentPadding = PaddingValues(bottom = 112.dp),
    ) {
        item(key = "map-inactive-header") {
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
                }
                Spacer(Modifier.height(6.dp))
                Text(stringResource(R.string.map_subtitle), color = LinkUpTextDimmed, fontSize = 13.sp)
            }
        }
        item(key = "map-inactive-city-context") {
            MapCityContextCard(
                cityContextEnabled = cityContextEnabled,
                cityContext = cityContext,
                cityPermissionGranted = cityPermissionGranted,
                cityPermissionDenied = cityPermissionDenied,
                onRequestCityContext = onRequestCityContext,
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 16.dp),
            )
        }
        item(key = "map-inactive-capability") {
            Column(
                Modifier.padding(horizontal = 20.dp, vertical = 8.dp).fillMaxWidth()
                    .background(LinkUpElevated, RoundedCornerShape(18.dp))
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(18.dp)).padding(20.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    stringResource(R.string.capability_not_active),
                    color = LinkUpTextPrimary,
                    fontFamily = LinkUpDesign.displayFont,
                    fontWeight = FontWeight.Bold,
                    fontSize = 20.sp,
                )
                Text(stringResource(R.string.map_v1_inactive), color = LinkUpTextDimmed, fontSize = 13.sp)
                LinkUpButton(
                    stringResource(R.string.open_pulse),
                    onOpenPulse,
                    Modifier.fillMaxWidth(),
                    LinkUpButtonVariant.PRIMARY,
                    LinkUpButtonSize.LG,
                )
            }
        }
    }
}
