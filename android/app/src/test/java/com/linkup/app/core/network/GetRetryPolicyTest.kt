package com.linkup.app.core.network

import java.io.IOException
import kotlin.test.Test
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class GetRetryPolicyTest {
    @Test
    fun retriesOnlyTransientGetFailures() {
        assertTrue(shouldRetryGet("GET", IOException("reset")))
        assertTrue(shouldRetryGet("GET", ApiException(503, "unavailable", "temporary")))
        assertTrue(shouldRetryGet("GET", ApiException(429, "rate_limited", "retry")))
        assertTrue(shouldRetryGet("GET", ApiException(408, "timeout", "retry")))

        assertFalse(shouldRetryGet("GET", ApiException(401, "unauthorized", "no")))
        assertFalse(shouldRetryGet("GET", ApiException(409, "conflict", "no")))
        assertFalse(shouldRetryGet("POST", IOException("ambiguous mutation")))
        assertFalse(shouldRetryGet("PATCH", ApiException(503, "unavailable", "ambiguous mutation")))
    }
}
