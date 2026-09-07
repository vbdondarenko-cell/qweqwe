package com.linkup.app.ui.design

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary

@Composable
fun FrozenMapScreen(modifier: Modifier = Modifier) {
    var selected by remember { mutableStateOf<FrozenMapPin?>(null) }
    BoxWithConstraints(modifier.fillMaxSize().background(Color(0xFF08090B))) {
        FrozenMapCanvas(Modifier.fillMaxSize())

        FrozenMapPins.forEach { pin ->
            val x = maxWidth * pin.x
            val y = maxHeight * pin.y
            FrozenMapPinView(
                pin = pin,
                selected = selected === pin,
                modifier = Modifier.offset(x = x - 20.dp, y = y - 20.dp),
                onClick = { selected = pin },
            )
        }

        Row(
            Modifier.align(Alignment.TopCenter).padding(top = 56.dp)
                .clip(RoundedCornerShape(999.dp)).background(LinkUpElevated.copy(alpha = .82f))
                .border(1.dp, LinkUpBorder, RoundedCornerShape(999.dp)).padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Box(Modifier.size(8.dp).clip(CircleShape).background(LinkUpRed))
            Text("Podil · Kyiv", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
        }

        Column(
            Modifier.align(Alignment.TopEnd).padding(top = 96.dp, end = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            FrozenMapControl(FrozenIconKind.SEARCH)
            FrozenMapControl(FrozenIconKind.PULSE)
            FrozenMapControl(FrozenIconKind.ME)
        }

        Row(
            Modifier.align(Alignment.BottomCenter).padding(start = 20.dp, end = 20.dp, bottom = 12.dp)
                .fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpElevated.copy(alpha = .86f))
                .border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Box(Modifier.size(8.dp).clip(CircleShape).background(LinkUpSuccess))
                    Text("ACTIVE", color = LinkUpSuccess, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.SemiBold, fontSize = 11.sp)
                }
                Text("${FrozenMapPins.size} active LinkUps", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 14.sp)
            }
            Text("List view  ›", color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
        }
    }

    LinkUpSheet(open = selected != null, onClose = { selected = null }) {
        selected?.slot?.let { slot ->
            Row(verticalAlignment = Alignment.Top) {
                Box(
                    Modifier.size(56.dp).clip(RoundedCornerShape(16.dp)).background(com.linkup.app.ui.theme.LinkUpZone)
                        .border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)),
                    contentAlignment = Alignment.Center,
                ) { Text(slot.emoji, fontSize = 28.sp) }
                Spacer(Modifier.width(12.dp))
                Column(Modifier.weight(1f)) {
                    LinkUpStatusBadge(slot.status)
                    Spacer(Modifier.height(4.dp))
                    Text(slot.title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 18.sp)
                    Text("${slot.location} · ${slot.distance}", color = LinkUpTextDimmed, fontSize = 12.sp)
                }
            }
            Spacer(Modifier.height(16.dp))
            Text(slot.description, color = LinkUpTextDimmed, fontSize = 14.sp)
            Spacer(Modifier.height(16.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                LinkUpAvatar(slot.organizer.initials, Color(slot.organizer.color), LinkUpAvatarSize.SM)
                Spacer(Modifier.width(8.dp))
                Text(slot.organizer.name, color = LinkUpTextDimmed, fontSize = 12.sp)
            }
            Spacer(Modifier.height(16.dp))
            LinkUpProgressBar(slot.joined, slot.capacity, showLabel = true)
            Spacer(Modifier.height(16.dp))
            LinkUpButton(
                label = if (slot.approval) "Request to join" else "Join now",
                onClick = { selected = null },
                modifier = Modifier.fillMaxWidth(),
                variant = if (slot.approval) LinkUpButtonVariant.SECONDARY else LinkUpButtonVariant.PRIMARY,
                size = LinkUpButtonSize.LG,
            )
            Spacer(Modifier.height(8.dp))
        }
    }
}

@Composable
private fun FrozenMapCanvas(modifier: Modifier = Modifier) {
    Canvas(modifier) {
        val grid = 40.dp.toPx()
        var x = 0f
        while (x < size.width) {
            drawLine(Color(0x12F7F8FA), Offset(x, 0f), Offset(x, size.height), strokeWidth = 1f)
            x += grid
        }
        var y = 0f
        while (y < size.height) {
            drawLine(Color(0x12F7F8FA), Offset(0f, y), Offset(size.width, y), strokeWidth = 1f)
            y += grid
        }
        fun road(points: List<Offset>, color: Color, width: Float) {
            val p = Path().apply {
                moveTo(points.first().x, points.first().y)
                points.drop(1).forEach { lineTo(it.x, it.y) }
            }
            drawPath(p, color, style = Stroke(width = width))
        }
        road(listOf(Offset(0f,size.height*.34f),Offset(size.width*.30f,size.height*.31f),Offset(size.width*.52f,size.height*.40f),Offset(size.width,size.height*.37f)), Color(0xFF1A1C20), 7f)
        road(listOf(Offset(0f,size.height*.60f),Offset(size.width*.25f,size.height*.65f),Offset(size.width*.52f,size.height*.55f),Offset(size.width,size.height*.62f)), Color(0xFF1A1C20), 7f)
        road(listOf(Offset(size.width*.20f,0f),Offset(size.width*.24f,size.height*.30f),Offset(size.width*.35f,size.height*.50f),Offset(size.width*.28f,size.height)), Color(0xFF1A1C20), 7f)
        road(listOf(Offset(size.width*.65f,0f),Offset(size.width*.70f,size.height*.25f),Offset(size.width*.60f,size.height*.50f),Offset(size.width*.72f,size.height)), Color(0xFF1A1C20), 7f)
        road(listOf(Offset(0f,size.height*.75f),Offset(size.width*.30f,size.height*.72f),Offset(size.width*.50f,size.height*.78f),Offset(size.width,size.height*.74f)), Color(0x550F1A2E), 18f)
        drawOval(Color(0x550D1A14), topLeft = Offset(size.width*.40f,size.height*.64f), size = androidx.compose.ui.geometry.Size(size.width*.16f,size.height*.09f))
    }
}

@Composable
private fun FrozenMapPinView(pin: FrozenMapPin, selected: Boolean, modifier: Modifier, onClick: () -> Unit) {
    val color = Color(pin.color)
    Column(modifier.clickable(onClick = onClick), horizontalAlignment = Alignment.CenterHorizontally) {
        Box(
            Modifier.size(if (selected) 44.dp else 40.dp).clip(CircleShape).background(color.copy(alpha = .15f))
                .border(2.dp, color, CircleShape),
            contentAlignment = Alignment.Center,
        ) { Text(pin.slot.emoji, fontSize = 18.sp) }
        Box(
            Modifier.padding(top = 2.dp).clip(RoundedCornerShape(6.dp)).background(color.copy(alpha = .10f))
                .border(1.dp, color.copy(alpha = .25f), RoundedCornerShape(6.dp)).padding(horizontal = 6.dp, vertical = 2.dp),
        ) { Text("${pin.slot.joined}/${pin.slot.capacity}", color = color, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 9.sp) }
    }
}

@Composable
private fun FrozenMapControl(icon: FrozenIconKind) {
    Box(
        Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated.copy(alpha = .82f))
            .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)),
        contentAlignment = Alignment.Center,
    ) { FrozenLineIcon(icon, LinkUpTextDimmed, Modifier.size(18.dp)) }
}
