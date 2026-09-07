package com.linkup.app.ui.design

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary

enum class FrozenMainTab { PULSE, MAP, CREATE, FLY, ME }

@Composable
fun FrozenBottomNav(
    selected: FrozenMainTab,
    onSelect: (FrozenMainTab) -> Unit,
    modifier: Modifier = Modifier,
) {
    Box(modifier.fillMaxWidth().height(92.dp), contentAlignment = Alignment.BottomCenter) {
        Row(
            Modifier.fillMaxWidth().height(76.dp).background(LinkUpElevated.copy(alpha = .92f))
                .border(1.dp, LinkUpBorder).padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.Bottom,
        ) {
            FrozenNavItem(FrozenMainTab.PULSE, "Pulse", FrozenIconKind.PULSE, selected, onSelect, Modifier.weight(1f))
            FrozenNavItem(FrozenMainTab.MAP, "Map", FrozenIconKind.MAP, selected, onSelect, Modifier.weight(1f))
            Box(Modifier.weight(1f).height(60.dp))
            FrozenNavItem(FrozenMainTab.FLY, "Fly", FrozenIconKind.FLY, selected, onSelect, Modifier.weight(1f))
            FrozenNavItem(FrozenMainTab.ME, "Me", FrozenIconKind.ME, selected, onSelect, Modifier.weight(1f))
        }
        Column(
            modifier = Modifier.offset(y = (-14).dp).clickable { onSelect(FrozenMainTab.CREATE) },
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                Modifier.size(56.dp).clip(CircleShape).background(LinkUpRed),
                contentAlignment = Alignment.Center,
            ) {
                Text("LINK", color = Color.White, fontFamily = LinkUpDesign.displayFont, fontSize = 14.sp, fontWeight = FontWeight.Black)
            }
            Text("Create", color = LinkUpTextMuted, fontFamily = LinkUpDesign.bodyFont, fontSize = 10.sp, fontWeight = FontWeight.SemiBold)
        }
    }
}

@Composable
private fun FrozenNavItem(
    tab: FrozenMainTab,
    label: String,
    icon: FrozenIconKind,
    selected: FrozenMainTab,
    onSelect: (FrozenMainTab) -> Unit,
    modifier: Modifier = Modifier,
) {
    val active = selected == tab
    Column(
        modifier = modifier.clickable { onSelect(tab) }.padding(vertical = 2.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(3.dp),
    ) {
        Box(
            Modifier.size(34.dp).clip(RoundedCornerShape(12.dp))
                .background(if (active) LinkUpRed.copy(alpha = .15f) else Color.Transparent),
            contentAlignment = Alignment.Center,
        ) {
            FrozenLineIcon(icon, if (active) LinkUpRed else LinkUpTextMuted, Modifier.size(22.dp))
        }
        Text(label, color = if (active) LinkUpRed else LinkUpTextMuted, fontSize = 10.sp, fontWeight = FontWeight.SemiBold)
        Box(Modifier.size(4.dp).clip(CircleShape).background(if (active) LinkUpRed else Color.Transparent))
    }
}
