package com.linkup.app.core.network

import java.net.URI
import java.net.URLDecoder
import java.util.Base64

/** Extract a credential from user-pasted text only. Never open the supplied URL. */
fun passwordResetToken(input: String): String? {
    val raw = input.trim()
    if (raw.length > 4096) return null
    val token = if (raw.startsWith("https://")) {
        try {
            val uri = URI(raw)
            if (uri.host.isNullOrBlank() || uri.userInfo != null || uri.fragment != null) return null
            val values = uri.rawQuery.orEmpty().split('&').mapNotNull { part ->
                val pair = part.split('=', limit = 2)
                if (URLDecoder.decode(pair[0], "UTF-8") == "token" && pair.size == 2)
                    URLDecoder.decode(pair[1], "UTF-8") else null
            }
            if (values.size != 1) return null
            values.single()
        } catch (_: IllegalArgumentException) { return null }
        catch (_: java.net.URISyntaxException) { return null }
    } else raw
    if (!token.matches(Regex("[A-Za-z0-9_-]{43}"))) return null
    return try {
        val bytes = Base64.getUrlDecoder().decode(token)
        if (bytes.size == 32 && Base64.getUrlEncoder().withoutPadding().encodeToString(bytes) == token) token else null
    } catch (_: IllegalArgumentException) { null }
}
