package com.linkup.app.ui.design

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
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
fun FrozenFlyScreen(
    onOpenPulse: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier.fillMaxSize().background(Color(0xFF050506))) {
        Column(
            Modifier.fillMaxWidth().background(LinkUpElevated.copy(alpha = .72f)).border(1.dp, LinkUpBorder)
                .padding(start = 20.dp, end = 20.dp, top = 56.dp, bottom = 12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                FrozenLineIcon(FrozenIconKind.FLY, LinkUpRed, Modifier.size(20.dp))
                Spacer(Modifier.width(8.dp))
                Text("Fly Now", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Black, fontSize = 24.sp)
            }
            Spacer(Modifier.height(6.dp))
            Text("Motion and proximity discovery", color = LinkUpTextDimmed, fontSize = 13.sp)
        }

        Column(
            Modifier.padding(horizontal = 20.dp, vertical = 24.dp).fillMaxWidth()
                .background(LinkUpElevated, RoundedCornerShape(18.dp))
                .border(1.dp, LinkUpBorder, RoundedCornerShape(18.dp)).padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("FLY · v1.1", color = LinkUpRed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 11.sp)
            Text("Not active in v1.0", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 20.sp)
            Text(
                "Fly Now/Travel/Motion needs server-time TTL, proximity and motion buckets, driver/passenger safety and privacy guardrails. Those authorities do not exist in the v1.0 API, so Join/Pass is not faked here.",
                color = LinkUpTextDimmed,
                fontSize = 13.sp,
            )
            Text("Current real join/request actions are available in Pulse.", color = LinkUpTextMuted, fontSize = 12.sp)
            LinkUpButton("Open Pulse", onOpenPulse, Modifier.fillMaxWidth(), LinkUpButtonVariant.PRIMARY, LinkUpButtonSize.LG)
        }
    }
}
