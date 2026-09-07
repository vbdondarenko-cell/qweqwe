package com.linkup.app.ui.design

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary

@Composable
fun FrozenMapScreen(
    onOpenPulse: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Box(modifier.fillMaxSize().background(Color(0xFF08090B))) {
        Canvas(Modifier.fillMaxSize()) {
            val grid = 40.dp.toPx()
            var x = 0f
            while (x < size.width) {
                drawLine(Color(0x10F7F8FA), Offset(x, 0f), Offset(x, size.height), strokeWidth = 1f)
                x += grid
            }
            var y = 0f
            while (y < size.height) {
                drawLine(Color(0x10F7F8FA), Offset(0f, y), Offset(size.width, y), strokeWidth = 1f)
                y += grid
            }
        }

        Column(
            Modifier.fillMaxWidth().background(LinkUpElevated.copy(alpha = .72f)).border(1.dp, LinkUpBorder)
                .padding(start = 20.dp, end = 20.dp, top = 56.dp, bottom = 12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                FrozenLineIcon(FrozenIconKind.MAP, LinkUpRed, Modifier.size(20.dp))
                Spacer(Modifier.width(8.dp))
                Text("Map", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Black, fontSize = 24.sp)
            }
            Spacer(Modifier.height(6.dp))
            Text("Location-aware discovery", color = LinkUpTextDimmed, fontSize = 13.sp)
        }

        Column(
            Modifier.align(Alignment.Center).padding(horizontal = 24.dp).fillMaxWidth()
                .background(LinkUpElevated.copy(alpha = .94f), RoundedCornerShape(18.dp))
                .border(1.dp, LinkUpBorder, RoundedCornerShape(18.dp)).padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("MAP · v1.1", color = LinkUpRed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 11.sp)
            Text("Not active in v1.0", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 20.sp)
            Text(
                "A real Map requires canonical place identities, server viewport queries, privacy-safe coordinates and clustering. v1.0 has place/zone text only, so this build does not invent map pins or user locations.",
                color = LinkUpTextDimmed,
                fontSize = 13.sp,
            )
            Text("Use Pulse for the current server-backed discovery flow.", color = LinkUpTextMuted, fontSize = 12.sp)
            LinkUpButton("Open Pulse", onOpenPulse, Modifier.fillMaxWidth(), LinkUpButtonVariant.PRIMARY, LinkUpButtonSize.LG)
        }
    }
}
