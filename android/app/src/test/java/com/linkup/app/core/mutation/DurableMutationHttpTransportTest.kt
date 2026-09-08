package com.linkup.app.core.mutation

import kotlin.test.Test
import kotlin.test.assertEquals

class DurableMutationHttpTransportTest {
    @Test
    fun `2xx is acknowledged`() {
        listOf(200, 201, 204, 299).forEach {
            assertEquals(DurableAttemptDisposition.ACKNOWLEDGED, durableDispositionForHttpStatus(it))
        }
    }

    @Test
    fun `definitive client errors are not auto replayed`() {
        listOf(400, 401, 403, 404, 409, 422).forEach {
            assertEquals(DurableAttemptDisposition.DEFINITIVE_FAILURE, durableDispositionForHttpStatus(it))
        }
    }

    @Test
    fun `timeout throttle and server errors remain ambiguous`() {
        listOf(408, 429, 500, 502, 503, 504).forEach {
            assertEquals(DurableAttemptDisposition.AMBIGUOUS_FAILURE, durableDispositionForHttpStatus(it))
        }
    }

    @Test
    fun `unexpected status remains fail safe ambiguous`() {
        listOf(100, 302, 399, 600).forEach {
            assertEquals(DurableAttemptDisposition.AMBIGUOUS_FAILURE, durableDispositionForHttpStatus(it))
        }
    }
}
