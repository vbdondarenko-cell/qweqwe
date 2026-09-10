package com.linkup.app.core.network

import org.json.JSONObject
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

class CityRealtimeApiClientTest {
    @Test
    fun `parser accepts privacy-safe ordered invalidations`() {
        val json = JSONObject(
            """{
              "cursor":12,
              "invalidations":[
                {"sequence":10,"eventId":"00000000-0000-0000-0000-000000000010","occurredAt":"2026-09-10T00:00:00Z"},
                {"sequence":12,"eventId":"00000000-0000-0000-0000-000000000012","occurredAt":"2026-09-10T00:00:01Z"}
              ]
            }""".trimIndent(),
        )

        val batch = parseCityRealtimeBatch(json, after = 9, limit = 100)

        assertEquals(12, batch.cursor)
        assertEquals(listOf(10L, 12L), batch.invalidations.map { it.sequence })
        assertEquals(2, batch.invalidations.size)
    }

    @Test
    fun `parser accepts cursor progress without visible city invalidations`() {
        val batch = parseCityRealtimeBatch(JSONObject("""{"cursor":42,"invalidations":[]}"""), after = 40, limit = 100)
        assertEquals(42, batch.cursor)
        assertEquals(emptyList(), batch.invalidations)
    }

    @Test
    fun `parser rejects out of order or malformed invalidation`() {
        val malformed = listOf(
            """{"cursor":12,"invalidations":[{"sequence":9,"eventId":"00000000-0000-0000-0000-000000000009","occurredAt":"2026-09-10T00:00:00Z"}]}""",
            """{"cursor":12,"invalidations":[{"sequence":12,"eventId":"not-a-uuid","occurredAt":"2026-09-10T00:00:00Z"}]}""",
            """{"cursor":12,"invalidations":[{"sequence":12,"eventId":"00000000-0000-0000-0000-000000000012","occurredAt":"bad"}]}""",
        )
        malformed.forEach { raw ->
            assertFailsWith<ApiException> { parseCityRealtimeBatch(JSONObject(raw), after = 10, limit = 100) }
        }
    }
}
