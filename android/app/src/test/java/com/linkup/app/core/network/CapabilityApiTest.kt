package com.linkup.app.core.network

import org.json.JSONObject
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CapabilityApiTest {
    @Test
    fun `missing and unknown keys fail closed`() {
        val snapshot = parseCapabilitySnapshot(JSONObject("""{
            "revision":12,
            "capabilities":{"realtime":true,"future_unknown":true}
        }"""))
        assertTrue(snapshot.enabled(CapabilityKey.REALTIME))
        assertFalse(snapshot.enabled(CapabilityKey.MAP))
        assertFalse(snapshot.enabled(CapabilityKey.NOTIFICATIONS))
    }

    @Test(expected = IllegalArgumentException::class)
    fun `invalid revision is rejected`() {
        parseCapabilitySnapshot(JSONObject("""{"revision":-1,"capabilities":{}}"""))
    }
}
