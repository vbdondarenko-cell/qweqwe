package com.linkup.app.ui.me

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
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
import com.linkup.app.R
import com.linkup.app.core.network.MonetizationApiClient
import com.linkup.app.core.network.MonetizationPlan
import com.linkup.app.core.network.MonetizationSnapshotModel
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.launch

private sealed interface PlusLoadState {
    data object Loading : PlusLoadState
    data class Content(val value: MonetizationSnapshotModel) : PlusLoadState
    data class Failure(val message: String) : PlusLoadState
}

@Composable
fun LinkUpPlusScreen(
    api: MonetizationApiClient,
    onBack: () -> Unit,
) {
    var state by remember { mutableStateOf<PlusLoadState>(PlusLoadState.Loading) }
    var refreshKey by remember { mutableIntStateOf(0) }
    var referralBusy by remember { mutableStateOf(false) }
    var referralError by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val failed = stringResource(R.string.plus_load_failed)

    BackHandler(onBack = onBack)
    LaunchedEffect(refreshKey) {
        state = PlusLoadState.Loading
        referralError = null
        state = try {
            PlusLoadState.Content(api.snapshot())
        } catch (error: Exception) {
            PlusLoadState.Failure(error.message ?: failed)
        }
    }

    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 20.dp, vertical = 18.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = onBack) { Text(stringResource(R.string.plus_back), color = LinkUpRed) }
            Text(stringResource(R.string.plus_title), color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 26.sp)
        }
        Text(stringResource(R.string.plus_subtitle), color = LinkUpTextDimmed, fontSize = 13.sp)

        when (val current = state) {
            PlusLoadState.Loading -> {
                CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.align(Alignment.CenterHorizontally))
                Text(stringResource(R.string.plus_loading), color = LinkUpTextMuted, modifier = Modifier.align(Alignment.CenterHorizontally))
            }
            is PlusLoadState.Failure -> {
                PlusCard {
                    Text(current.message, color = LinkUpWarning, fontSize = 12.sp)
                    TextButton(onClick = { refreshKey++ }) { Text(stringResource(R.string.common_retry), color = LinkUpRed) }
                }
            }
            is PlusLoadState.Content -> PlusContent(
                snapshot = current.value,
                referralBusy = referralBusy,
                referralError = referralError,
                onRefresh = { refreshKey++ },
                onBindReferral = { code ->
                    if (!referralBusy) {
                        scope.launch {
                            referralBusy = true
                            referralError = null
                            try {
                                state = PlusLoadState.Content(api.bindReferral(code))
                            } catch (error: Exception) {
                                referralError = error.message ?: failed
                            } finally {
                                referralBusy = false
                            }
                        }
                    }
                },
            )
        }
        Spacer(Modifier.height(20.dp))
    }
}

@Composable
private fun PlusContent(
    snapshot: MonetizationSnapshotModel,
    referralBusy: Boolean,
    referralError: String?,
    onRefresh: () -> Unit,
    onBindReferral: (String) -> Unit,
) {
    val status = snapshot.status
    val catalog = snapshot.catalog
    val monthly = catalog.plans.firstOrNull { it.id == "monthly" }
    val annual = catalog.plans.firstOrNull { it.id == "annual" }
    var referralInput by remember { mutableStateOf("") }

    PlusCard {
        SectionTitle(stringResource(R.string.plus_status_title))
        Text(
            if (status.premiumActive) stringResource(R.string.plus_status_active) else stringResource(R.string.plus_status_inactive),
            color = if (status.premiumActive) LinkUpRed else LinkUpTextDimmed,
            fontWeight = FontWeight.Bold,
        )
        status.premiumUntilEpochMillis?.let {
            Text(stringResource(R.string.plus_until_format, formatDateTime(it)), color = LinkUpTextMuted, fontSize = 12.sp)
        }
        TextButton(onClick = onRefresh) { Text(stringResource(R.string.common_refresh), color = LinkUpRed) }
    }

    PlusCard {
        SectionTitle(stringResource(R.string.plus_plans_title))
        monthly?.let { PlanRow(stringResource(R.string.plus_monthly), it) }
        annual?.let { PlanRow(stringResource(R.string.plus_annual), it) }
        Text(
            stringResource(
                R.string.plus_annual_saving_format,
                formatUah(catalog.annualSavingsUahMinor),
                catalog.annualSavingsPercent,
            ),
            color = LinkUpRed,
            fontWeight = FontWeight.Bold,
            fontSize = 12.sp,
        )
    }

    PlusCard {
        SectionTitle(stringResource(R.string.plus_rewarded_title))
        Text(
            stringResource(
                R.string.plus_rewarded_policy_format,
                catalog.rewarded.videoIntervalSeconds / 3600,
                catalog.rewarded.videosRequired,
                catalog.rewarded.nominalCompletionHours,
            ),
            color = LinkUpTextDimmed,
            fontSize = 12.sp,
        )
        Text(stringResource(R.string.plus_rewarded_reward), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            repeat(catalog.rewarded.videosRequired) { index ->
                Box(
                    Modifier.size(width = 34.dp, height = 8.dp)
                        .clip(RoundedCornerShape(4.dp))
                        .background(if (index < status.rewarded.videosWatchedCount) LinkUpRed else LinkUpZone),
                )
            }
        }
        Text(
            stringResource(R.string.plus_rewarded_progress_format, status.rewarded.videosWatchedCount, catalog.rewarded.videosRequired),
            color = LinkUpTextMuted,
            fontSize = 11.sp,
        )
        status.rewarded.nextVideoAtEpochMillis?.let {
            Text(stringResource(R.string.plus_next_video_format, formatDateTime(it)), color = LinkUpTextMuted, fontSize = 11.sp)
        }
        status.rewarded.nextFreePremiumClaimAtEpochMillis?.let {
            Text(stringResource(R.string.plus_next_claim_format, formatDateTime(it)), color = LinkUpTextMuted, fontSize = 11.sp)
        }
        Text(stringResource(R.string.plus_rewarded_weekly_limit), color = LinkUpTextMuted, fontSize = 11.sp)
    }

    PlusCard {
        SectionTitle(stringResource(R.string.plus_referral_title))
        Text(stringResource(R.string.plus_your_referral_code), color = LinkUpTextMuted, fontSize = 11.sp)
        Text(status.referral.referralCode, color = LinkUpRed, fontWeight = FontWeight.Black, fontSize = 20.sp)
        Text(
            stringResource(R.string.plus_referral_count_format, status.referral.qualifiedReferrals),
            color = LinkUpTextPrimary,
            fontWeight = FontWeight.Bold,
        )
        Text(
            stringResource(R.string.plus_referral_deadline_format, catalog.referralDeadlineDays),
            color = LinkUpTextDimmed,
            fontSize = 12.sp,
        )
        if (status.referral.boundReferralCode != null) {
            Text(
                stringResource(R.string.plus_bound_referral_format, status.referral.boundReferralCode),
                color = LinkUpTextPrimary,
                fontWeight = FontWeight.Bold,
                fontSize = 12.sp,
            )
            status.referral.qualifyingDeadlineEpochMillis?.let {
                Text(stringResource(R.string.plus_referral_qualify_by_format, formatDateTime(it)), color = LinkUpTextMuted, fontSize = 11.sp)
            }
        } else {
            OutlinedTextField(
                value = referralInput,
                onValueChange = { referralInput = it.uppercase().filter { char -> char.isLetterOrDigit() }.take(20) },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.plus_enter_referral_code)) },
                singleLine = true,
                enabled = !referralBusy,
            )
            TextButton(
                onClick = { onBindReferral(referralInput) },
                enabled = !referralBusy && referralInput.length in 6..20,
            ) {
                Text(
                    if (referralBusy) stringResource(R.string.plus_referral_binding) else stringResource(R.string.plus_apply_referral_code),
                    color = if (referralBusy) LinkUpTextMuted else LinkUpRed,
                    fontWeight = FontWeight.Bold,
                )
            }
            referralError?.let { Text(it, color = LinkUpWarning, fontSize = 11.sp) }
        }
        catalog.referralMilestones.forEach { milestone ->
            Text(
                stringResource(
                    R.string.plus_milestone_format,
                    milestone.qualifiedReferrals,
                    milestone.inviterRewardDays,
                    milestone.inviteeRewardDays,
                ) + if (milestone.badge) " · ${stringResource(R.string.plus_badge)}" else "",
                color = if (status.referral.qualifiedReferrals >= milestone.qualifiedReferrals) LinkUpRed else LinkUpTextDimmed,
                fontSize = 12.sp,
                fontWeight = if (status.referral.qualifiedReferrals >= milestone.qualifiedReferrals) FontWeight.Bold else FontWeight.Normal,
            )
        }
    }

    if (!snapshot.capabilities.paidVerification || !snapshot.capabilities.rewardedVerification || !snapshot.capabilities.referralQualification) {
        PlusCard {
            SectionTitle(stringResource(R.string.plus_verification_title))
            Text(stringResource(R.string.plus_provider_unavailable), color = LinkUpWarning, fontSize = 12.sp)
            CapabilityRow(stringResource(R.string.plus_capability_paid), snapshot.capabilities.paidVerification)
            CapabilityRow(stringResource(R.string.plus_capability_rewarded), snapshot.capabilities.rewardedVerification)
            CapabilityRow(stringResource(R.string.plus_capability_referral), snapshot.capabilities.referralQualification)
        }
    }
}

@Composable
private fun PlanRow(label: String, plan: MonetizationPlan) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(label, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
            Text(
                stringResource(R.string.plus_effective_monthly_format, formatUah(plan.effectiveMonthlyUahMinor)),
                color = LinkUpTextMuted,
                fontSize = 11.sp,
            )
        }
        Text(formatUah(plan.priceUahMinor), color = LinkUpRed, fontWeight = FontWeight.Black)
    }
}

@Composable
private fun CapabilityRow(label: String, active: Boolean) {
    Row(Modifier.fillMaxWidth()) {
        Text(label, color = LinkUpTextDimmed, modifier = Modifier.weight(1f), fontSize = 11.sp)
        Text(
            if (active) stringResource(R.string.plus_capability_ready) else stringResource(R.string.plus_capability_not_ready),
            color = if (active) LinkUpRed else LinkUpTextMuted,
            fontSize = 11.sp,
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
private fun PlusCard(content: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpElevated)
            .border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
        content = content,
    )
}

@Composable
private fun SectionTitle(text: String) {
    Text(text, color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 17.sp)
}

private fun formatUah(minor: Int): String = "%d,%02d грн".format(minor / 100, minor % 100)

private fun formatDateTime(epochMillis: Long): String = DateTimeFormatter.ofPattern("dd.MM.yyyy HH:mm")
    .withZone(ZoneId.systemDefault())
    .format(Instant.ofEpochMilli(epochMillis))
