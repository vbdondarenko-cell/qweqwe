package com.linkup.app.core.network

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

class ApiEndpointTest {
    @Test
    fun acceptsHttpsAndExactDevelopmentHosts() {
        for (url in listOf("https://api.example.com", "https://api.example.com/base", "http://10.0.2.2:8080", "http://127.0.0.1:8080")) {
            assertEquals(url, validatedApiRoot("$url/"))
        }
    }

    @Test
    fun rejectsAuthorityConfusionAndInsecureRemoteEndpoints() {
        for (url in listOf(
            "http://10.0.2.2.evil.example", "http://127.0.0.1.evil.example",
            "http://10.0.2.2@evil.example", "http://127.0.0.1@evil.example",
            "https://user:password@api.example.com", "http://api.example.com",
            "https://", "https:///path", "https://api.example.com?token=secret",
            "https://api.example.com#fragment", "https://api.example.com:70000",
            "https://api.example.com:0", " https://api.example.com",
        )) {
            assertFailsWith<IllegalArgumentException>(url) { validatedApiRoot(url) }
        }
    }
}
