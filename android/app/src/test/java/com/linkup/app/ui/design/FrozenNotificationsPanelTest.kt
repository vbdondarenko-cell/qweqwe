package com.linkup.app.ui.design

import com.linkup.app.ui.theme.LinkUpInfo
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpWarning
import org.junit.Assert.assertEquals
import org.junit.Test

class FrozenNotificationsPanelTest {
    @Test
    fun socialTypesUseThePersonIconAndSuccessColor() {
        assertEquals(NotificationVisual(FrozenIconKind.ME, LinkUpSuccess), visualFor("FRIEND_REQUEST"))
        assertEquals(NotificationVisual(FrozenIconKind.ME, LinkUpSuccess), visualFor("FRIEND_ACCEPTED"))
    }

    @Test
    fun recommendationUsesInfoColor() {
        assertEquals(NotificationVisual(FrozenIconKind.PULSE, LinkUpInfo), visualFor("EVENT_RECOMMENDATION"))
    }

    @Test
    fun promoTypesUseTheFlyIconAndWarningColor() {
        assertEquals(NotificationVisual(FrozenIconKind.FLY, LinkUpWarning), visualFor("PROMO"))
        assertEquals(NotificationVisual(FrozenIconKind.FLY, LinkUpWarning), visualFor("ADVERTISEMENT"))
    }

    @Test
    fun criticalTypesUseTheBellIconAndMutedColor() {
        assertEquals(NotificationVisual(FrozenIconKind.BELL, LinkUpTextMuted), visualFor("SYSTEM"))
        assertEquals(NotificationVisual(FrozenIconKind.BELL, LinkUpTextMuted), visualFor("SECURITY"))
        assertEquals(NotificationVisual(FrozenIconKind.BELL, LinkUpTextMuted), visualFor("ACCOUNT"))
    }

    @Test
    fun eventAndMessageTypesFallBackToThePulseIconAndBrandRed() {
        assertEquals(NotificationVisual(FrozenIconKind.PULSE, LinkUpRed), visualFor("MESSAGE"))
        assertEquals(NotificationVisual(FrozenIconKind.PULSE, LinkUpRed), visualFor("EVENT"))
        assertEquals(NotificationVisual(FrozenIconKind.PULSE, LinkUpRed), visualFor("EVENT_REMINDER"))
    }

    @Test
    fun unknownTypeFailsClosedToTheBrandDefaultRatherThanCrashing() {
        assertEquals(NotificationVisual(FrozenIconKind.PULSE, LinkUpRed), visualFor("SOME_FUTURE_TYPE_NOT_YET_MAPPED"))
    }
}
