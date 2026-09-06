package com.linkup.app.core.network

import java.net.URI

/** Validate the parsed authority before sending credentials, never a URL prefix. */
internal fun validatedApiRoot(baseUrl: String): String {
    val uri = try { URI(baseUrl) } catch (_: Exception) {
        throw IllegalArgumentException("Invalid API base URL")
    }
    val local = uri.host in setOf("10.0.2.2", "127.0.0.1")
    require(
        !uri.host.isNullOrBlank() &&
            uri.rawUserInfo == null && uri.rawQuery == null && uri.rawFragment == null &&
            (uri.port == -1 || uri.port in 1..65535) &&
            (uri.scheme == "https" || (uri.scheme == "http" && local)),
    ) { "API base URL must use HTTPS outside local Android development" }
    return uri.toASCIIString().trimEnd('/')
}
