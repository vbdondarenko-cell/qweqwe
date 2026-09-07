package com.linkup.app.core.network

import java.io.ByteArrayInputStream
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

class ApiResponseBoundaryTest {
    @Test
    fun readsResponseWithinLimit() {
        val body = "Привіт LinkUp".toByteArray(Charsets.UTF_8)
        assertEquals("Привіт LinkUp", readUtf8Bounded(ByteArrayInputStream(body), body.size))
    }

    @Test
    fun rejectsResponseBeyondLimit() {
        val body = ByteArray(33) { 'x'.code.toByte() }
        assertFailsWith<ResponseTooLargeException> {
            readUtf8Bounded(ByteArrayInputStream(body), 32)
        }
    }

    @Test
    fun byteLimitIsAppliedBeforeUtf8StringExpansion() {
        val body = "🙂🙂🙂".toByteArray(Charsets.UTF_8)
        assertFailsWith<ResponseTooLargeException> {
            readUtf8Bounded(ByteArrayInputStream(body), body.size - 1)
        }
    }
}
