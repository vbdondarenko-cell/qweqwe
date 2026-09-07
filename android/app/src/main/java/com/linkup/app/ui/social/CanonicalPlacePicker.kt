package com.linkup.app.ui.social

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.CanonicalPlace
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning

@Composable
internal fun CanonicalPlacePicker(
    locationText: String,
    selected: CanonicalPlace?,
    state: LoadState<List<CanonicalPlace>>,
    enabled: Boolean,
    onSearch: () -> Unit,
    onSelect: (CanonicalPlace) -> Unit,
    onUseTextOnly: () -> Unit,
) {
    val searchErrorFallback = stringResource(R.string.create_place_error)
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                stringResource(R.string.create_place_map_notice),
                color = LinkUpTextMuted,
                fontSize = 11.sp,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            TextButton(
                enabled = enabled && locationText.trim().length >= 2,
                onClick = onSearch,
            ) {
                Text(stringResource(R.string.create_place_search), color = LinkUpRed, fontSize = 11.sp)
            }
        }

        selected?.let { place ->
            Row(
                Modifier.fillMaxWidth()
                    .background(LinkUpSuccess.copy(alpha = .10f), RoundedCornerShape(12.dp))
                    .border(1.dp, LinkUpSuccess.copy(alpha = .45f), RoundedCornerShape(12.dp))
                    .padding(horizontal = 12.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(R.string.create_place_selected, place.name),
                        color = LinkUpTextPrimary,
                        fontWeight = FontWeight.SemiBold,
                        fontSize = 12.sp,
                    )
                    Text(placeMeta(place), color = LinkUpTextMuted, fontSize = 10.sp)
                }
                TextButton(enabled = enabled, onClick = onUseTextOnly) {
                    Text(stringResource(R.string.create_place_text_only), color = LinkUpTextDimmed, fontSize = 10.sp)
                }
            }
        }

        when (state) {
            LoadState.Idle -> Unit
            LoadState.Loading -> Text(stringResource(R.string.create_place_searching), color = LinkUpTextDimmed, fontSize = 11.sp)
            LoadState.Empty -> Text(stringResource(R.string.create_place_empty), color = LinkUpWarning, fontSize = 11.sp)
            is LoadState.Failure -> Text(
                state.error.message.ifBlank { searchErrorFallback },
                color = LinkUpWarning,
                fontSize = 11.sp,
            )
            is LoadState.Content -> {
                if (state.refreshing) {
                    Text(stringResource(R.string.create_place_searching), color = LinkUpTextMuted, fontSize = 10.sp)
                }
                state.value.take(8).forEach { place ->
                    Column(
                        Modifier.fillMaxWidth()
                            .background(LinkUpElevated, RoundedCornerShape(12.dp))
                            .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp))
                            .clickable(enabled = enabled) { onSelect(place) }
                            .padding(horizontal = 12.dp, vertical = 10.dp),
                        verticalArrangement = Arrangement.spacedBy(2.dp),
                    ) {
                        Text(place.name, color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                        Text(placeMeta(place), color = LinkUpTextMuted, fontSize = 10.sp)
                    }
                }
                state.refreshError?.let {
                    Text(it.message.ifBlank { searchErrorFallback }, color = LinkUpWarning, fontSize = 10.sp)
                }
            }
        }
    }
}

@Composable
private fun placeMeta(place: CanonicalPlace): String {
    val location = listOfNotNull(place.locality, place.category, place.countryCode)
        .filter { it.isNotBlank() }
        .distinct()
        .joinToString(" · ")
        .ifBlank { place.name }
    return stringResource(R.string.create_place_result_meta, location, place.precisionM)
}
