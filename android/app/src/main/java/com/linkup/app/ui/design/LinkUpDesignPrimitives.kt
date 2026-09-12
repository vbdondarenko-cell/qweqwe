package com.linkup.app.ui.design

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpCritical
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpInfo
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone

object LinkUpDesign {
    val radiusBadge = 6.dp
    val radiusCompact = 8.dp
    val radiusControl = 12.dp
    val radiusCard = 16.dp
    val radiusSheet = 24.dp
    val pagePadding = 20.dp
    val cardPadding = 16.dp
    val borderWidth = 1.dp
    val progressHeight = 6.dp

    // Frozen design font roles. Exact Outfit/Inter/JetBrains Mono resources
    // are not yet present in the Android repository; these role aliases keep
    // every screen structurally ready for the real bundled families.
    val displayFont = FontFamily.SansSerif
    val bodyFont = FontFamily.SansSerif
    val monoFont = FontFamily.Monospace
}

enum class LinkUpAvatarSize(val diameter: Dp, val textSize: Int) {
    SM(24.dp, 10), MD(32.dp, 12), LG(48.dp, 14), XL(80.dp, 24),
}

@Composable
fun LinkUpAvatar(
    initials: String,
    color: Color,
    size: LinkUpAvatarSize = LinkUpAvatarSize.MD,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier
            .size(size.diameter)
            .clip(CircleShape)
            .background(color.copy(alpha = 0.133f))
            .border(1.5.dp, color, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = initials,
            color = LinkUpTextPrimary,
            fontFamily = LinkUpDesign.bodyFont,
            fontWeight = FontWeight.Bold,
            fontSize = size.textSize.sp,
        )
    }
}

enum class LinkUpButtonVariant { PRIMARY, SECONDARY, GHOST, DANGER, SUCCESS }
enum class LinkUpButtonSize(val horizontal: Dp, val vertical: Dp, val fontSize: Int, val radius: Dp) {
    SM(12.dp, 6.dp, 12, 8.dp),
    MD(16.dp, 10.dp, 14, 12.dp),
    LG(20.dp, 14.dp, 16, 16.dp),
}

@Composable
fun LinkUpButton(
    label: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: LinkUpButtonVariant = LinkUpButtonVariant.PRIMARY,
    size: LinkUpButtonSize = LinkUpButtonSize.MD,
    enabled: Boolean = true,
) {
    val (background, foreground, border) = when (variant) {
        LinkUpButtonVariant.PRIMARY -> Triple(LinkUpRed, LinkUpTextPrimary, Color.Transparent)
        LinkUpButtonVariant.SECONDARY -> Triple(LinkUpElevated, LinkUpTextPrimary, LinkUpBorder)
        LinkUpButtonVariant.GHOST -> Triple(Color.Transparent, LinkUpTextDimmed, Color.Transparent)
        LinkUpButtonVariant.DANGER -> Triple(LinkUpCritical.copy(alpha = .15f), LinkUpCritical, LinkUpCritical.copy(alpha = .30f))
        LinkUpButtonVariant.SUCCESS -> Triple(LinkUpSuccess.copy(alpha = .15f), LinkUpSuccess, LinkUpSuccess.copy(alpha = .30f))
    }
    val shape = RoundedCornerShape(size.radius)
    Box(
        modifier = modifier
            .clip(shape)
            .background(if (enabled) background else LinkUpElevated)
            .then(if (border != Color.Transparent) Modifier.border(LinkUpDesign.borderWidth, border, shape) else Modifier)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = size.horizontal, vertical = size.vertical),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            label,
            color = if (enabled) foreground else LinkUpTextMuted,
            fontFamily = LinkUpDesign.bodyFont,
            fontWeight = FontWeight.SemiBold,
            fontSize = size.fontSize.sp,
        )
    }
}

@Composable
fun LinkUpChip(
    label: String,
    active: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    leading: (@Composable () -> Unit)? = null,
) {
    val shape = RoundedCornerShape(LinkUpDesign.radiusControl)
    Row(
        modifier = modifier
            .clip(shape)
            .background(if (active) LinkUpRed else LinkUpElevated)
            .then(if (active) Modifier else Modifier.border(LinkUpDesign.borderWidth, LinkUpBorder, shape))
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        leading?.invoke()
        Text(
            label,
            color = if (active) LinkUpTextPrimary else LinkUpTextDimmed,
            fontFamily = LinkUpDesign.bodyFont,
            fontWeight = FontWeight.SemiBold,
            fontSize = 12.sp,
        )
    }
}

/**
 * Frozen-design text input: flat elevated/bordered box, no Material3
 * floating-label outline animation — matches the search field already
 * built for [FrozenMapScreen]'s `MapCenterSearch`.
 */
@Composable
fun LinkUpTextField(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    label: String? = null,
    placeholder: String? = null,
    enabled: Boolean = true,
    singleLine: Boolean = true,
    minLines: Int = 1,
    isPassword: Boolean = false,
    isError: Boolean = false,
) {
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(6.dp)) {
        if (label != null) {
            Text(
                label,
                color = LinkUpTextDimmed,
                fontFamily = LinkUpDesign.bodyFont,
                fontWeight = FontWeight.SemiBold,
                fontSize = 12.sp,
            )
        }
        val shape = RoundedCornerShape(LinkUpDesign.radiusControl)
        val borderColor = if (isError) LinkUpWarning else LinkUpBorder
        Box(
            Modifier.fillMaxWidth()
                .let { if (minLines > 1) it.heightIn(min = (minLines * 20).dp) else it }
                .clip(shape)
                .background(LinkUpElevated)
                .border(LinkUpDesign.borderWidth, borderColor, shape)
                .padding(horizontal = 14.dp, vertical = 13.dp),
            contentAlignment = if (minLines > 1) Alignment.TopStart else Alignment.CenterStart,
        ) {
            BasicTextField(
                value = value,
                onValueChange = onValueChange,
                enabled = enabled,
                singleLine = singleLine,
                textStyle = TextStyle(color = LinkUpTextPrimary, fontFamily = LinkUpDesign.bodyFont, fontSize = 14.sp),
                visualTransformation = if (isPassword) PasswordVisualTransformation() else VisualTransformation.None,
                cursorBrush = SolidColor(LinkUpRed),
                modifier = Modifier.fillMaxWidth(),
                decorationBox = { inner ->
                    if (value.isEmpty() && placeholder != null) {
                        Text(placeholder, color = LinkUpTextMuted, fontFamily = LinkUpDesign.bodyFont, fontSize = 14.sp)
                    }
                    inner()
                },
            )
        }
    }
}

@Composable
fun LinkUpCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    verticalArrangement: Arrangement.Vertical = Arrangement.Top,
    content: @Composable ColumnScope.() -> Unit,
) {
    val shape = RoundedCornerShape(LinkUpDesign.radiusCard)
    Column(
        modifier = modifier
            .clip(shape)
            .background(LinkUpElevated)
            .border(BorderStroke(LinkUpDesign.borderWidth, LinkUpBorder), shape)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(LinkUpDesign.cardPadding),
        verticalArrangement = verticalArrangement,
        content = content,
    )
}

@Composable
fun LinkUpProgressBar(
    value: Int,
    max: Int,
    modifier: Modifier = Modifier,
    fill: Color = LinkUpRed,
    showLabel: Boolean = false,
) {
    val fraction = if (max <= 0) 0f else (value.toFloat() / max.toFloat()).coerceIn(0f, 1f)
    Row(modifier = modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier.weight(1f).height(LinkUpDesign.progressHeight)
                .clip(RoundedCornerShape(99.dp)).background(LinkUpZone),
        ) {
            Box(
                Modifier.fillMaxWidth(fraction).height(LinkUpDesign.progressHeight)
                    .clip(RoundedCornerShape(99.dp)).background(fill),
            )
        }
        if (showLabel) {
            Spacer(Modifier.width(8.dp))
            Text(
                "$value/$max",
                color = LinkUpTextDimmed,
                fontFamily = LinkUpDesign.monoFont,
                fontSize = 12.sp,
            )
        }
    }
}

enum class LinkUpVisualStatus { LIVE, OPEN, FULL, APPROVAL }

@Composable
fun LinkUpStatusBadge(status: LinkUpVisualStatus, modifier: Modifier = Modifier) {
    val color = when (status) {
        LinkUpVisualStatus.LIVE -> LinkUpSuccess
        LinkUpVisualStatus.OPEN -> LinkUpInfo
        LinkUpVisualStatus.FULL -> LinkUpTextMuted
        LinkUpVisualStatus.APPROVAL -> LinkUpWarning
    }
    Row(
        modifier = modifier
            .clip(RoundedCornerShape(LinkUpDesign.radiusBadge))
            .background(color.copy(alpha = .15f))
            .border(1.dp, color.copy(alpha = .30f), RoundedCornerShape(LinkUpDesign.radiusBadge))
            .padding(horizontal = 8.dp, vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        if (status == LinkUpVisualStatus.LIVE) {
            Box(Modifier.size(6.dp).clip(CircleShape).background(LinkUpSuccess))
        }
        Text(
            status.name,
            color = color,
            fontFamily = LinkUpDesign.monoFont,
            fontWeight = FontWeight.SemiBold,
            fontSize = 10.sp,
        )
    }
}

@Composable
fun LinkUpEmptyState(
    title: String,
    subtitle: String,
    modifier: Modifier = Modifier,
    icon: (@Composable () -> Unit)? = null,
) {
    Column(
        modifier = modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 80.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier.size(80.dp).clip(RoundedCornerShape(24.dp))
                .background(LinkUpZone).border(1.dp, LinkUpBorder, RoundedCornerShape(24.dp)),
            contentAlignment = Alignment.Center,
        ) {
            if (icon != null) icon() else Text("—", color = LinkUpTextMuted, fontSize = 32.sp)
        }
        Spacer(Modifier.height(16.dp))
        Text(
            title,
            color = LinkUpTextPrimary,
            fontFamily = LinkUpDesign.displayFont,
            fontWeight = FontWeight.Bold,
            fontSize = 18.sp,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            subtitle,
            color = LinkUpTextDimmed,
            fontFamily = LinkUpDesign.bodyFont,
            fontSize = 14.sp,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
fun LinkUpErrorState(
    message: String,
    onRetry: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
    title: String = "Something went wrong",
) {
    Column(
        modifier = modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 80.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier.size(64.dp).clip(RoundedCornerShape(16.dp))
                .background(LinkUpCritical.copy(alpha = .10f))
                .border(1.dp, LinkUpCritical.copy(alpha = .20f), RoundedCornerShape(16.dp)),
            contentAlignment = Alignment.Center,
        ) { Text("!", color = LinkUpCritical, fontWeight = FontWeight.Bold, fontSize = 28.sp) }
        Spacer(Modifier.height(16.dp))
        Text(
            title,
            color = LinkUpTextPrimary,
            fontFamily = LinkUpDesign.displayFont,
            fontWeight = FontWeight.Bold,
            fontSize = 18.sp,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(4.dp))
        Text(message, color = LinkUpTextDimmed, fontSize = 14.sp, textAlign = TextAlign.Center)
        if (onRetry != null) {
            Spacer(Modifier.height(16.dp))
            LinkUpButton(label = "Try again", onClick = onRetry, variant = LinkUpButtonVariant.SECONDARY)
        }
    }
}

@Composable
fun LinkUpSkeleton(modifier: Modifier = Modifier) {
    Box(modifier.clip(RoundedCornerShape(12.dp)).background(LinkUpBorder.copy(alpha = .65f)))
}

@Composable
fun LinkUpSlotCardSkeleton(modifier: Modifier = Modifier) {
    Column(
        modifier = modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp))
            .background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            LinkUpSkeleton(Modifier.size(48.dp))
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                LinkUpSkeleton(Modifier.fillMaxWidth(.30f).height(12.dp))
                LinkUpSkeleton(Modifier.fillMaxWidth(.75f).height(16.dp))
                LinkUpSkeleton(Modifier.fillMaxWidth(.50f).height(12.dp))
            }
        }
        LinkUpSkeleton(Modifier.fillMaxWidth().height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            LinkUpSkeleton(Modifier.width(64.dp).height(24.dp))
            LinkUpSkeleton(Modifier.width(64.dp).height(24.dp))
        }
    }
}

@Composable
fun LinkUpSheet(
    open: Boolean,
    onClose: () -> Unit,
    modifier: Modifier = Modifier,
    title: String? = null,
    action: (@Composable () -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    if (!open) return
    Dialog(
        onDismissRequest = onClose,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Box(Modifier.fillMaxSize().background(Color.Black.copy(alpha = .60f)), contentAlignment = Alignment.BottomCenter) {
            Column(
                modifier = modifier.fillMaxWidth().clip(RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp))
                    .background(LinkUpElevated.copy(alpha = .98f))
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp)),
            ) {
                Box(Modifier.fillMaxWidth().height(12.dp), contentAlignment = Alignment.Center) {
                    Box(Modifier.width(40.dp).height(4.dp).clip(RoundedCornerShape(99.dp)).background(LinkUpBorder))
                }
                Row(
                    Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    if (title != null) {
                        Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 18.sp)
                    }
                    Spacer(Modifier.weight(1f))
                    action?.invoke()
                    Spacer(Modifier.width(8.dp))
                    Box(
                        Modifier.size(32.dp).clip(RoundedCornerShape(8.dp)).clickable(onClick = onClose),
                        contentAlignment = Alignment.Center,
                    ) { Text("×", color = LinkUpTextMuted, fontSize = 20.sp) }
                }
                Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp), content = content)
            }
        }
    }
}
